package base

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

// runQuery runs src as the file at path with q, a query over callArgs that
// returns its path argument as an absolute path, and the results ok and bad
// from runWithResults.
func runQuery(t *testing.T, path, src string) (starlark.StringDict, error) {
	t.Helper()
	q := NewQuery(context.Background(), "q", callArgs, func(inst *FunctionInstance) (starlark.Value, error) {
		p, _, err := inst.AbsPath("path")
		if err != nil {
			return nil, err
		}
		return starlark.String(p), nil
	})
	ok, err := Result{Label: "fetch", Success: true, Return: starlark.String(absFile("tmp", "a.zip"))}.ToStarlark()
	require.NoError(t, err)
	bad, err := Result{Label: "broken", Error: errors.New("exit status 3")}.ToStarlark()
	require.NoError(t, err)
	predeclared := starlark.StringDict{"q": q.Builtin(), "ok": ok, "bad": bad}
	return starlark.ExecFileOptions(&syntax.FileOptions{}, &starlark.Thread{}, path, src, predeclared)
}

func TestQuery_ReturnsValueDirectly(t *testing.T) {
	p := filepath.ToSlash(absFile("etc", "hosts"))
	globals, err := runQuery(t, absFile("repo", "a.star"), `
pos = q("`+p+`", [])
kw = q(args = [], path = "`+p+`")
from_result = q(ok, [])
`)
	require.NoError(t, err)
	require.Equal(t, starlark.String(absFile("etc", "hosts")), globals["pos"])
	require.Equal(t, globals["pos"], globals["kw"])
	require.Equal(t, starlark.String(absFile("tmp", "a.zip")), globals["from_result"])
}

func TestQuery_Errors(t *testing.T) {
	file := absFile("repo", "a.star")
	for src, want := range map[string]string{
		// No common arguments.
		`q("/x", [], label = "x")`:    `q: unexpected keyword argument "label"`,
		`q("/x", [], only_if = True)`: `q: unexpected keyword argument "only_if"`,
		// Required arguments and types are checked as for a Module.
		`q(path = "/x")`:  `q: missing argument for args`,
		`q("/x", "nope")`: `q: for parameter args: got string, want list`,
		// Errors from the path accessors name the query without a label.
		`q("rel", [])`: `q: path: "rel" is not an absolute path`,
		// A result that didn't succeed stops the run: there's nothing to skip.
		`q(bad, [])`: `q: path is the result of "broken", which didn't succeed (exit status 3); pass r.value to use it anyway`,
	} {
		_, err := runQuery(t, file, src)
		require.ErrorContains(t, err, want, src)
	}
}

func TestInstanceErrorf(t *testing.T) {
	step := &FunctionInstance{fn: "file", Label: "config"}
	require.EqualError(t, step.Errorf("%s: %w", "/x", errors.New("boom")), `file(label="config"): /x: boom`)
	query := &FunctionInstance{fn: "read_file"}
	require.EqualError(t, query.Errorf("%s", "/x"), `read_file: /x`)
}
