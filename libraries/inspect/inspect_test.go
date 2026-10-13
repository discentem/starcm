package inspect

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFile(t *testing.T) {
	src := `load("starcm", "download", ld = "load_dynamic", run = "exec")
load("lib/winpe.star", "ps")

download(label = "x", url = "u", save_to = "s", sha256 = "h")
ld("main.star", label = "literal")
ld(label = "module kwarg", module = "other.star")
name = "computed"
ld(name + ".star", label = "computed")
f = ld
def setup():
    run(label = "y", cmd = "cmd.exe", args = [], download = 1)
    return ps("z", "Get-Date")
print("not loaded")
`
	r, err := File("main.star", []byte(src))
	require.NoError(t, err)

	got, err := json.Marshal(r)
	require.NoError(t, err)
	require.JSONEq(t, `{
	  "file": "main.star",
	  "loads": [
	    {"module": "starcm", "line": 1, "names": {"download": "download", "ld": "load_dynamic", "run": "exec"}},
	    {"module": "lib/winpe.star", "line": 2, "names": {"ps": "ps"}}
	  ],
	  "calls": [
	    {"module": "starcm", "function": "download", "line": 4},
	    {"module": "starcm", "function": "load_dynamic", "line": 5, "path": "main.star"},
	    {"module": "starcm", "function": "load_dynamic", "line": 6, "path": "other.star"},
	    {"module": "starcm", "function": "load_dynamic", "line": 8, "path": null},
	    {"module": "starcm", "function": "exec", "line": 11},
	    {"module": "lib/winpe.star", "function": "ps", "line": 12}
	  ],
	  "values": [
	    {"module": "starcm", "function": "load_dynamic", "line": 9}
	  ]
	}`, string(got))
}

func TestFileEmptyListsNotNull(t *testing.T) {
	r, err := File("empty.star", []byte("x = 1\n"))
	require.NoError(t, err)
	got, err := json.Marshal(r)
	require.NoError(t, err)
	require.JSONEq(t, `{"file": "empty.star", "loads": [], "calls": [], "values": []}`, string(got))
}

func TestFileTopLevelControl(t *testing.T) {
	// starcm allows if/for at the top level; inspect must parse what starcm runs.
	_, err := File("loop.star", []byte("for x in [1]:\n    pass\n"))
	require.NoError(t, err)
}

func TestFileSyntaxError(t *testing.T) {
	_, err := File("bad.star", []byte("def (\n"))
	require.ErrorContains(t, err, "bad.star:1")
}
