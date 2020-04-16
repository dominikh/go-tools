package lint

import (
	"strings"

	"honnef.co/go/tools/facts"
	"honnef.co/go/tools/runner"
)

func parseDirectives(dirs []facts.SerializedDirective) ([]ignore, []Problem) {
	var ignores []ignore
	var problems []Problem

	for _, dir := range dirs {
		cmd := dir.Command
		args := dir.Arguments
		switch cmd {
		case "ignore", "file-ignore":
			if len(args) < 2 {
				p := Problem{
					Diagnostic: runner.Diagnostic{
						Position: dir.NodePosition,
						Message:  "malformed linter directive; missing the required reason field?",
						Category: "compile",
					},
					Severity: Error,
				}
				problems = append(problems, p)
				continue
			}
		default:
			// unknown directive, ignore
			continue
		}
		checks := strings.Split(args[0], ",")
		pos := dir.NodePosition
		var ig ignore
		switch cmd {
		case "ignore":
			ig = &lineIgnore{
				File:   pos.Filename,
				Line:   pos.Line,
				Checks: checks,
				Pos:    dir.DirectivePosition,
			}
		case "file-ignore":
			ig = &fileIgnore{
				File:   pos.Filename,
				Checks: checks,
			}
		}
		ignores = append(ignores, ig)
	}

	return ignores, problems
}
