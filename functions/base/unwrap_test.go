package base

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

// runWithResults runs src with fn (backed by a callRecorder, declared with
// callArgs) and two results: ok, which succeeded with value "/tmp/a.zip", and
// bad, which failed.
func runWithResults(t *testing.T, src string) (*callRecorder, error) {
	t.Helper()
	ok, err := Result{Label: "fetch", Success: true, Return: starlark.String("/tmp/a.zip")}.ToStarlark()
	require.NoError(t, err)
	bad, err := Result{Label: "broken", Error: errors.New("exit status 3")}.ToStarlark()
	require.NoError(t, err)
	action := &callRecorder{}
	m := NewModule(context.Background(), "fn", callArgs, action)
	predeclared := starlark.StringDict{
		"fn":  starlark.NewBuiltin("fn", m.Function()),
		"ok":  ok,
		"bad": bad,
	}
	_, err = starlark.ExecFileOptions(&syntax.FileOptions{}, &starlark.Thread{}, absFile("repo", "a.star"), src, predeclared)
	return action, err
}

func TestUnwrap_ResultStandsForItsValue(t *testing.T) {
	action, err := runWithResults(t, `
fn(ok, [], label = "positional")
fn(label = "keyword", path = ok, args = [])
fn(label = "nested", path = "p", args = [ok, ("x", ok), {"k": ok}])
`)
	require.NoError(t, err)
	for _, c := range action.calls[:2] {
		p, _ := Get[string](c, "path")
		require.Equal(t, "/tmp/a.zip", p, c.Label)
	}
	args, _ := Get[*starlark.List](action.calls[2], "args")
	require.Equal(t, `["/tmp/a.zip", ("x", "/tmp/a.zip"), {"k": "/tmp/a.zip"}]`, args.String())
}

func TestUnwrap_FailedResultStopsTheRun(t *testing.T) {
	action, err := runWithResults(t, `fn(label = "use it", path = bad, args = [])`)
	require.EqualError(t, err, `use it: path is the result of "broken", which didn't succeed (exit status 3); pass r.value to use it anyway`)
	require.Empty(t, action.calls)

	// Inside a list too.
	_, err = runWithResults(t, `fn(label = "in a list", path = "p", args = [ok, bad])`)
	require.ErrorContains(t, err, `in a list: args is the result of "broken", which didn't succeed`)
}

func TestUnwrap_SkippedCallIgnoresFailedResult(t *testing.T) {
	action, err := runWithResults(t, `
fn(label = "skipped", path = bad, args = [], only_if = False)
fn(label = "also skipped", path = bad, args = [], not_if = True)
`)
	require.NoError(t, err)
	require.Empty(t, action.calls)
}

// In only_if and not_if a result stands for its success, and one that
// didn't succeed is False rather than a failure that stops the run.
func TestUnwrap_ConditionsReadSuccess(t *testing.T) {
	action, err := runWithResults(t, `
fn(label = "only_if ok", path = "p", args = [], only_if = ok)
fn(label = "only_if bad", path = "p", args = [], only_if = bad)
fn(label = "not_if ok", path = "p", args = [], not_if = ok)
fn(label = "not_if bad", path = "p", args = [], not_if = bad)
`)
	require.NoError(t, err)
	var ran []string
	for _, c := range action.calls {
		ran = append(ran, c.Label)
	}
	require.Equal(t, []string{"only_if ok", "not_if bad"}, ran)
}

// A condition reading a result doesn't change how the other arguments read
// theirs: a failed result passed as a value still stops a call that runs.
func TestUnwrap_ConditionDoesntExcuseFailedValue(t *testing.T) {
	_, err := runWithResults(t, `fn(label = "runs", path = bad, args = [], only_if = ok)`)
	require.ErrorContains(t, err, `runs: path is the result of "broken", which didn't succeed`)

	// Skipped by a failed result in only_if: the failed value is never used.
	action, err := runWithResults(t, `fn(label = "skipped", path = bad, args = [], only_if = bad)`)
	require.NoError(t, err)
	require.Empty(t, action.calls)
}
