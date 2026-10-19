package dedent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.starlark.net/starlark"
)

func TestDedent(t *testing.T) {
	for _, tc := range []struct{ name, in, want string }{
		{"indented block", "\n    server = x\n    channel = y\n", "server = x\nchannel = y\n"},
		{"closing quotes indented", "\n        server = x\n        channel = y\n    ", "server = x\nchannel = y\n"},
		{"keeps relative indentation", "\n    <dict>\n        <key>Label</key>\n    </dict>\n", "<dict>\n    <key>Label</key>\n</dict>\n"},
		{"blank lines don't count", "\n    a\n\n    b\n", "a\n\nb\n"},
		{"whitespace-only lines become empty", "\n    a\n      \n    b\n", "a\n\nb\n"},
		{"no newline at the end", "\n    a\n    b", "a\nb"},
		{"already flush", "a\nb\n", "a\nb\n"},
		{"only one leading newline is dropped", "\n\n    a\n", "\na\n"},
		{"tabs and spaces differ", "\n\ta\n    b\n", "\ta\n    b\n"},
		{"empty", "", ""},
	} {
		assert.Equal(t, tc.want, Dedent(tc.in), tc.name)
	}
}

func TestBuiltin(t *testing.T) {
	v, err := starlark.Call(&starlark.Thread{}, New(), starlark.Tuple{starlark.String("\n    a\n    ")}, nil)
	assert.NoError(t, err)
	assert.Equal(t, starlark.String("a\n"), v)

	_, err = starlark.Call(&starlark.Thread{}, New(), starlark.Tuple{starlark.MakeInt(1)}, nil)
	assert.ErrorContains(t, err, "dedent")
	_, err = starlark.Call(&starlark.Thread{}, New(), nil, nil)
	assert.ErrorContains(t, err, "dedent")
}
