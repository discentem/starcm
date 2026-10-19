// Package dedent implements dedent(), which lets multi-line strings be
// indented with the code around them.
package dedent

import (
	"strings"

	"go.starlark.net/starlark"
)

// Dedent removes a leading newline, the indentation all non-blank lines
// share, and the indentation before the closing quotes, so
//
//	dedent("""
//	    server = x
//	      indented = y
//	    """)
//
// is "server = x\n  indented = y\n". Blank lines become empty; tabs and
// spaces count as different characters.
func Dedent(s string) string {
	s = strings.TrimPrefix(s, "\n")
	lines := strings.Split(s, "\n")
	// The last line is what follows the final newline: the closing quotes'
	// indentation, or text with no newline after it.
	last := len(lines) - 1
	if strings.TrimSpace(lines[last]) == "" {
		lines[last] = ""
	}
	prefix := ""
	first := true
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
		if first {
			prefix, first = indent, false
			continue
		}
		for !strings.HasPrefix(indent, prefix) {
			prefix = prefix[:len(prefix)-1]
		}
	}
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			lines[i] = ""
		} else {
			lines[i] = strings.TrimPrefix(line, prefix)
		}
	}
	return strings.Join(lines, "\n")
}

// New returns the dedent() builtin: dedent(text) returns Dedent(text).
func New() *starlark.Builtin {
	return starlark.NewBuiltin("dedent", func(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var text string
		if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &text); err != nil {
			return nil, err
		}
		return starlark.String(Dedent(text)), nil
	})
}
