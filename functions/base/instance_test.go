package base

import (
	"context"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/discentem/starcm/libraries/paths"

	"github.com/mitchellh/go-homedir"
	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

// callRecorder is a Action that keeps every FunctionInstance it gets.
type callRecorder struct {
	calls []*FunctionInstance
}

func (a *callRecorder) Execute(inst *FunctionInstance) (*Result, error) {
	a.calls = append(a.calls, inst)
	return &Result{Label: inst.Label, Success: true}, nil
}

var callArgs = []ArgPair{
	Arg[string]("path"),
	Arg[*starlark.List]("args"),
	OptionalArg[string]("dest"),
	OptionalArgAllowNone[bool]("flag"),
}

func absFile(parts ...string) string {
	root := "/"
	if runtime.GOOS == "windows" {
		root = `C:\`
	}
	return filepath.Join(append([]string{root}, parts...)...)
}

// runCalls executes src as the file at path with fn backed by a callRecorder.
func runCalls(t *testing.T, path, src string) (*callRecorder, error) {
	t.Helper()
	action := &callRecorder{}
	m := NewModule(context.Background(), "fn", callArgs, action)
	predeclared := starlark.StringDict{"fn": starlark.NewBuiltin("fn", m.Function())}
	_, err := starlark.ExecFileOptions(&syntax.FileOptions{}, &starlark.Thread{}, path, src, predeclared)
	return action, err
}

func snapshot(c *FunctionInstance) map[string]any {
	path, _ := Get[string](c, "path")
	args, _ := Get[*starlark.List](c, "args")
	dest, destGiven := Get[string](c, "dest")
	flag, flagGiven := Get[bool](c, "flag")
	return map[string]any{
		"path": path, "args": args.String(),
		"dest": dest, "destGiven": destGiven,
		"flag": flag, "flagGiven": flagGiven,
	}
}

// Positional and keyword calls give the action the same values.
func TestInstance_PositionalMatchesKeyword(t *testing.T) {
	action, err := runCalls(t, absFile("repo", "a.star"), `
fn(label = "kw", path = "a.txt", args = ["x"], dest = "d")
fn("a.txt", ["x"], "d", label = "pos")
fn("a.txt", args = ["x"], dest = "d", label = "mixed")
`)
	require.NoError(t, err)
	require.Len(t, action.calls, 3)
	want := snapshot(action.calls[0])
	for _, c := range action.calls[1:] {
		require.Equal(t, want, snapshot(c), c.Label)
	}
	require.Equal(t, map[string]any{
		"path": "a.txt", "args": `["x"]`, "dest": "d", "destGiven": true, "flag": false, "flagGiven": false,
	}, want)
}

// Omitted optional arguments, and None for OptionalArgAllowNone, aren't given.
func TestInstance_Given(t *testing.T) {
	action, err := runCalls(t, absFile("repo", "a.star"), `
fn(label = "omitted", path = "p", args = [])
fn(label = "none", path = "p", args = [], flag = None)
fn(label = "set", path = "p", args = [], flag = True)
`)
	require.NoError(t, err)
	require.False(t, action.calls[0].Given("dest"))
	require.False(t, action.calls[0].Given("flag"))
	require.False(t, action.calls[1].Given("flag"))
	flag, given := Get[bool](action.calls[2], "flag")
	require.True(t, flag)
	require.True(t, given)
}

// A skipped call never reaches the action.
func TestInstance_SkippedCallsDontRun(t *testing.T) {
	action, err := runCalls(t, absFile("repo", "a.star"), `
fn(label = "a", path = "p", args = [], only_if = False)
fn(label = "b", path = "p", args = [], not_if = True)
`)
	require.NoError(t, err)
	require.Empty(t, action.calls)
}

func TestInstance_Paths(t *testing.T) {
	home, err := homedir.Expand("~")
	require.NoError(t, err)
	file := absFile("repo", "site", "web.star")
	action, err := runCalls(t, file, `
fn(label = "rel", path = "motd.jinja", args = [])
fn(label = "abs", path = "`+filepath.ToSlash(absFile("etc", "motd"))+`", args = [])
fn(label = "tilde", path = "~/x", args = [])
`)
	require.NoError(t, err)
	rel, abs, tilde := action.calls[0], action.calls[1], action.calls[2]
	require.Equal(t, absFile("repo", "site"), rel.Dir)

	p, given, err := rel.ResourcePath("path")
	require.NoError(t, err)
	require.True(t, given)
	require.Equal(t, absFile("repo", "site", "motd.jinja"), p)

	_, _, err = rel.AbsPath("path")
	require.ErrorContains(t, err, `fn(label="rel"): path: "motd.jinja" is not an absolute path`)

	p, _, err = abs.AbsPath("path")
	require.NoError(t, err)
	require.Equal(t, absFile("etc", "motd"), p)

	p, _, err = tilde.AbsPath("path")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(home, "x"), p)

	p, given, err = rel.AbsPath("dest")
	require.NoError(t, err)
	require.False(t, given)
	require.Empty(t, p)
}

// Arguments declared with Arg are required even after optional ones, which
// UnpackArgs alone would treat as optional.
func TestInstance_RequiredAfterOptional(t *testing.T) {
	action := &callRecorder{}
	m := NewModule(context.Background(), "fn", []ArgPair{
		OptionalArg[string]("first"),
		Arg[string]("needed"),
	}, action)
	predeclared := starlark.StringDict{"fn": starlark.NewBuiltin("fn", m.Function())}
	_, err := starlark.ExecFileOptions(&syntax.FileOptions{}, &starlark.Thread{}, absFile("repo", "a.star"), `fn(label = "x", first = "a")`, predeclared)
	require.ErrorContains(t, err, "x: missing argument for needed")
	require.Empty(t, action.calls)
}

// Asking for an undeclared name or the wrong type is a programming error.
func TestGet_Panics(t *testing.T) {
	action, err := runCalls(t, absFile("repo", "a.star"), `fn(label = "a", path = "p", args = [])`)
	require.NoError(t, err)
	c := action.calls[0]
	require.PanicsWithValue(t, `base.Get: fn has no argument "nope"`, func() { Get[string](c, "nope") })
	require.Panics(t, func() { Get[int](c, "path") })
}

// In a module loaded with load_remote, which runs from the store, a relative
// resource path gets an error that says why instead of a missing file.
func TestInstance_ResourcePathInStore(t *testing.T) {
	store := absFile("store")
	file := filepath.Join(store, "sha256", "abc", "settings.star")
	action := &callRecorder{}
	m := NewModule(context.Background(), "fn", callArgs, action)
	thread := &starlark.Thread{}
	paths.SetStoreCheck(thread, func(p string) bool { return strings.HasPrefix(p, store) })
	_, err := starlark.ExecFileOptions(&syntax.FileOptions{}, thread, file, `
fn(label = "rel", path = "motd.jinja", args = [])
fn(label = "abs", path = "`+filepath.ToSlash(absFile("etc", "motd.jinja"))+`", args = [])
`, starlark.StringDict{"fn": starlark.NewBuiltin("fn", m.Function())})
	require.NoError(t, err)

	_, _, err = action.calls[0].ResourcePath("path")
	require.EqualError(t, err, `fn(label="rel"): path: "motd.jinja" is relative, but this .star file was loaded with load_remote and runs from starcm's store, where nothing ships next to it: fetch the file with download() or pass its absolute path`)

	p, _, err := action.calls[1].ResourcePath("path")
	require.NoError(t, err)
	require.Equal(t, absFile("etc", "motd.jinja"), p)
}
