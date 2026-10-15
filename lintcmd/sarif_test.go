package lintcmd

import (
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

func TestSARIFColumnUsesUTF16CodeUnits(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source.go")
	const source = "aé😀x\n"
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}

	// The byte columns for x is 8, while its one-based UTF-16 column is 5.
	pos := token.Position{Filename: path, Line: 1, Column: 8, Offset: 7}
	got := (&sarifFormatter{}).sarifColumn(pos)
	if got != 5 {
		t.Fatalf("sarifColumn(%+v) = %d, want 5", pos, got)
	}
}

func TestSARIFColumnFallsBackWhenSourceIsUnavailable(t *testing.T) {
	pos := token.Position{Filename: filepath.Join(t.TempDir(), "missing.go"), Line: 1, Column: 7, Offset: 6}
	got := (&sarifFormatter{}).sarifColumn(pos)
	if got != pos.Column {
		t.Fatalf("sarifColumn(%+v) = %d, want fallback %d", pos, got, pos.Column)
	}
}
