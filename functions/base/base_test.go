package base

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

type call struct {
	label       string
	name        string
	hasDeadline bool
}

// recordingAction records what base.Module passed to each Execute.
type recordingAction struct {
	calls []call
}

func (a *recordingAction) Execute(inst *FunctionInstance) (*Result, error) {
	_, hasDeadline := inst.Ctx.Deadline()
	name, _ := Get[string](inst, "name")
	a.calls = append(a.calls, call{label: inst.Label, name: name, hasDeadline: hasDeadline})
	return &Result{Label: inst.Label, Success: true}, nil
}

func exec(t *testing.T, src string) (*recordingAction, error) {
	t.Helper()
	action := &recordingAction{}
	m := NewModule(context.Background(), "fn", []ArgPair{OptionalArg[string]("name")}, action)
	predeclared := starlark.StringDict{
		"fn": starlark.NewBuiltin("fn", m.Function()),
	}
	_, err := starlark.ExecFileOptions(&syntax.FileOptions{}, &starlark.Thread{}, "test.star", src, predeclared)
	return action, err
}

// Optional arguments a call omits must not keep the value from an earlier call.
func TestFunction_NoCarryOverBetweenCalls(t *testing.T) {
	action, err := exec(t, `
fn(label = "a", timeout = "1h", name = "x")
fn(label = "b")
r = fn(label = "c", only_if = False)
fn(label = "d")
fn(label = "e", not_if = True)
fn(label = "f")
`)
	require.NoError(t, err)
	require.Equal(t, []call{
		{label: "a", name: "x", hasDeadline: true},
		{label: "b"},
		// c and e are skipped
		{label: "d"},
		{label: "f"},
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

// working_directory was removed: outputs take absolute paths instead.
func TestFunction_NoWorkingDirectory(t *testing.T) {
	_, err := exec(t, `fn(label = "a", working_directory = "/elsewhere")`)
	require.ErrorContains(t, err, `unexpected keyword argument "working_directory"`)
}

// Without a label, argument errors fall back to the function's name.
func TestFunction_UnpackErrorWithoutLabelUsesBuiltinName(t *testing.T) {
	_, err := exec(t, `fn(name = 42)`)
	require.Error(t, err)
	require.Contains(t, err.Error(), `fn: for parameter "name": got int, want string`)
}

// Each call unpacks function-specific arguments into its own target.
func TestArg_FreshTargetPerCall(t *testing.T) {
	a := OptionalArg[string]("name")
	first, second := a.newTarget(), a.newTarget()
	require.IsType(t, new(string), first)
	require.NotSame(t, first, second)
}

func TestArgKinds(t *testing.T) {
	require.Equal(t, "name", Arg[string]("name").Key)
	require.Equal(t, "name?", OptionalArg[string]("name").Key)
	require.Equal(t, "name??", OptionalArgAllowNone[string]("name").Key)
	require.Panics(t, func() { Arg[string]("name?") })
	require.Panics(t, func() { Arg[string]("name??") })
}

// OptionalArg rejects an explicit None; OptionalArgAllowNone treats it as
// left out.
func TestOptionalArgNone(t *testing.T) {
	unpack := func(a ArgPair) error {
		return starlark.UnpackArgs("fn", nil, []starlark.Tuple{{starlark.String("name"), starlark.None}}, a.Key, a.newTarget())
	}
	require.ErrorContains(t, unpack(OptionalArg[string]("name")), `for parameter "name": got NoneType, want string`)
	require.NoError(t, unpack(OptionalArgAllowNone[string]("name")))

	// Leaving either out is fine.
	for _, a := range []ArgPair{OptionalArg[string]("name"), OptionalArgAllowNone[string]("name")} {
		require.NoError(t, starlark.UnpackArgs("fn", nil, nil, a.Key, a.newTarget()))
	}
}
