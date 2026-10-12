package base

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

type call struct {
	label            string
	workingDirectory string
	hasDeadline      bool
}

// recordingAction records what base.Module passed to each Run.
type recordingAction struct {
	calls []call
}

func (a *recordingAction) Run(ctx context.Context, workingDirectory string, label string, thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (*Result, error) {
	_, hasDeadline := ctx.Deadline()
	a.calls = append(a.calls, call{label: label, workingDirectory: workingDirectory, hasDeadline: hasDeadline})
	return &Result{Label: label, Success: true}, nil
}

func exec(t *testing.T, src string) (*recordingAction, error) {
	t.Helper()
	var name string
	action := &recordingAction{}
	m := NewModule(context.Background(), "test_type", []ArgPair{{Key: "name?", Type: &name}}, action)
	predeclared := starlark.StringDict{
		"fn": starlark.NewBuiltin("fn", m.Function()),
	}
	_, err := starlark.ExecFileOptions(&syntax.FileOptions{}, &starlark.Thread{}, "test.star", src, predeclared)
	return action, err
}

// Optional arguments a call omits must not keep the value from an earlier call.
func TestFunction_NoCarryOverBetweenCalls(t *testing.T) {
	action, err := exec(t, `
fn(label = "a", timeout = "1h", working_directory = "/elsewhere")
fn(label = "b")
r = fn(label = "c", only_if = False)
fn(label = "d")
fn(label = "e", not_if = True)
fn(label = "f")
`)
	require.NoError(t, err)
	require.Equal(t, []call{
		{label: "a", workingDirectory: "/elsewhere", hasDeadline: true},
		{label: "b", workingDirectory: ".", hasDeadline: false},
		// c and e are skipped
		{label: "d", workingDirectory: ".", hasDeadline: false},
		{label: "f", workingDirectory: ".", hasDeadline: false},
	}, action.calls)
}

// Argument errors are prefixed with the failing call's own label, not the
// label of whichever call last unpacked successfully.
func TestFunction_UnpackErrorUsesCallLabel(t *testing.T) {
	_, err := exec(t, `
fn(label = "previous")
fn(label = "current", name = 42)
`)
	require.Error(t, err)
	require.Contains(t, err.Error(), `current: for parameter "name": got int, want string`)
	require.NotContains(t, err.Error(), "previous")
}

// Without a label, argument errors fall back to the builtin's name.
func TestFunction_UnpackErrorWithoutLabelUsesBuiltinName(t *testing.T) {
	_, err := exec(t, `fn(name = 42)`)
	require.Error(t, err)
	require.Contains(t, err.Error(), `fn: for parameter "name": got int, want string`)
}
