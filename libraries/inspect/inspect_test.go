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

func TestInputs(t *testing.T) {
	src := `load("starcm", "exec", "read_file", "vars", sh = "exec")

exec(label = "literal", cmd = "ls", args = ["-l"])
exec(label = "direct", cmd = "scutil", args = ["--set", "ComputerName", vars["name"]])
greeting = "say hello " + vars["user"]
words = ["-c"]
words.append(greeting)
sh(label = "through names", cmd = "sh", args = words)
cmd = read_file("cmd.txt")
exec(label = "from a file", cmd = cmd, args = [], timeout = vars["timeout"])

def quoted(s):
    return "'%s'" % s

def run(c):
    exec(label = "inside a function", cmd = "sh", args = ["-c", c])

run(quoted(vars["script"]))
for f in [vars["a"], "b"]:
    exec("positional", "echo", [f])
opts = {"cmd": "echo"}
opts["args"] = [x for x in vars["list"]]
exec(label = "unpacked", **opts)
value = vars["unrelated"]
exec(label = "field names aren't names", cmd = read_file("x"), args = [])
`
	r, err := File("main.star", []byte(src))
	require.NoError(t, err)

	vars := `{"module": "starcm", "function": "vars"}`
	readFile := `{"module": "starcm", "function": "read_file"}`
	got, err := json.Marshal(r.Calls)
	require.NoError(t, err)
	require.JSONEq(t, `[
	  {"module": "starcm", "function": "exec", "line": 3},
	  {"module": "starcm", "function": "exec", "line": 4, "inputs": [{"arg": "args", "uses": [`+vars+`]}]},
	  {"module": "starcm", "function": "exec", "line": 8, "inputs": [{"arg": "args", "uses": [`+vars+`]}]},
	  {"module": "starcm", "function": "read_file", "line": 9},
	  {"module": "starcm", "function": "exec", "line": 10, "inputs": [
	    {"arg": "cmd", "uses": [`+readFile+`]},
	    {"arg": "timeout", "uses": [`+vars+`]}
	  ]},
	  {"module": "starcm", "function": "exec", "line": 16, "inputs": [{"arg": "args", "uses": [`+vars+`]}]},
	  {"module": "starcm", "function": "exec", "line": 20, "inputs": [{"arg": "2", "uses": [`+vars+`]}]},
	  {"module": "starcm", "function": "exec", "line": 23, "inputs": [{"arg": "**", "uses": [`+vars+`]}]},
	  {"module": "starcm", "function": "exec", "line": 25, "inputs": [{"arg": "cmd", "uses": [`+readFile+`]}]},
	  {"module": "starcm", "function": "read_file", "line": 25}
	]`, string(got))
}

func TestLoadRemoteURL(t *testing.T) {
	src := `load("starcm", "install_remote_mac_pkg", "load_remote", "vars")

load_remote("https://config.example.com/a.star", sha256 = "h", label = "literal")
load_remote(label = "kwarg", url = "s3://config/b.star", sha256 = "h")
load_remote(vars["url"], sha256 = "h", label = "computed")
install_remote_mac_pkg("https://releases.example.com/agent.pkg", sha256 = "h", signer = "s", label = "pkg")
`
	r, err := File("main.star", []byte(src))
	require.NoError(t, err)
	got, err := json.Marshal(r.Calls)
	require.NoError(t, err)
	require.JSONEq(t, `[
	  {"module": "starcm", "function": "load_remote", "line": 3, "url": "https://config.example.com/a.star"},
	  {"module": "starcm", "function": "load_remote", "line": 4, "url": "s3://config/b.star"},
	  {"module": "starcm", "function": "load_remote", "line": 5, "url": null,
	   "inputs": [{"arg": "0", "uses": [{"module": "starcm", "function": "vars"}]}]},
	  {"module": "starcm", "function": "install_remote_mac_pkg", "line": 6, "url": "https://releases.example.com/agent.pkg"}
	]`, string(got))
}
