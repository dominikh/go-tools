package format

import (
	"encoding/json"
	"fmt"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"text/tabwriter"

	"honnef.co/go/tools/lint"
)

func shortPath(path string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return path
	}
	if rel, err := filepath.Rel(cwd, path); err == nil && len(rel) < len(path) {
		return rel
	}
	return path
}

func relativePositionString(pos token.Position) string {
	s := shortPath(pos.Filename)
	if pos.IsValid() {
		if s != "" {
			s += ":"
		}
		s += fmt.Sprintf("%d:%d", pos.Line, pos.Column)
	}
	if s == "" {
		s = "-"
	}
	return s
}

type Flusher interface {
	Flush()
}

type Formatter interface {
	Format(p lint.Problem)
}

type Text struct {
	W io.Writer
}

func (o Text) Format(p lint.Problem) {
	fmt.Fprintf(o.W, "%v: %s\n", relativePositionString(p.Position), p.String())
}

type JSON struct {
	W io.Writer
}

func (o JSON) Format(p lint.Problem) {
	type location struct {
		File   string `json:"file"`
		Line   int    `json:"line"`
		Column int    `json:"column"`
	}
	jp := struct {
		Checker  string   `json:"checker"`
		Code     string   `json:"code"`
		Severity string   `json:"severity,omitempty"`
		Location location `json:"location"`
		Message  string   `json:"message"`
		Ignored  bool     `json:"ignored"`
	}{
		p.Checker,
		p.Check,
		"", // TODO(dh): support severity
		location{
			p.Position.Filename,
			p.Position.Line,
			p.Position.Column,
		},
		p.Text,
		p.Ignored,
	}
	_ = json.NewEncoder(o.W).Encode(jp)
}

type Stylish struct {
	W io.Writer

	prevFile string
	tw       *tabwriter.Writer
}

func (o *Stylish) Format(p lint.Problem) {
	if p.Position.Filename != o.prevFile {
		if o.prevFile != "" {
			o.tw.Flush()
			fmt.Fprintln(o.W)
		}
		fmt.Fprintln(o.W, p.Position.Filename)
		o.prevFile = p.Position.Filename
		o.tw = tabwriter.NewWriter(o.W, 0, 4, 2, ' ', 0)
	}
	fmt.Fprintf(o.tw, "  (%d, %d)\t%s\t%s\n", p.Position.Line, p.Position.Column, p.Check, p.Text)
}

func (o *Stylish) Flush() {
	if o.tw != nil {
		o.tw.Flush()
	}
}
