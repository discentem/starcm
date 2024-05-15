package base

import (
	"fmt"

	starlarkhelpers "github.com/discentem/starcm/starlark-helpers"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

type ArgPair struct {
	Key  string
	Type any
}

type Runnable interface {
	Run(args starlark.Tuple, kwargs []starlark.Tuple) (*Result, error)
}

type Module struct {
	Args   []ArgPair
	Action Runnable
}

// Function produces a starlark Function that has common behavior that useful for all modules like only_if, not_if, and after
func (m *Module) Function() starlarkhelpers.Function {
	finalArgs := make([]any, 0)
	// Add arguments that are specific to this module
	for _, arg := range m.Args {
		finalArgs = append(finalArgs, arg.Key, arg.Type)
	}
	var (
		name   string
		notIf  starlark.Bool
		onlyIf starlark.Bool
		before starlark.Callable
	)

	// Common arguments automatically available for all modules
	commonArgs := []any{
		"name", &name,
		"only_if??", &onlyIf,
		"before?", &before,
		"not_if?", &notIf,
	}

	finalArgs = append(
		finalArgs,
		commonArgs...,
	)

	return func(thread *starlark.Thread, builtin *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := starlark.UnpackArgs(
			name,
			args,
			kwargs,
			finalArgs...,
		); err != nil {
			return starlark.None, err
		}
		// skip module if not_if is true
		if notIf.Truth() {
			fmt.Printf("skipping module %s because not_if was true", name)
			return starlark.None, nil
		}

		isAbsent := func(value starlark.Value) bool {
			return value != starlark.None
		}
		// If onlyIf is absent, it is assumed to be true meaning the module will run
		if isAbsent(onlyIf) {
			onlyIf = starlark.True
		}

		if onlyIf.Truth() == starlark.False {
			fmt.Println(onlyIf)
			fmt.Printf("skipping module %s because only_if was false", name)
			return starlark.None, nil
		}

		beforeResult, err := starlark.Call(thread, before, args, kwargs)
		if err != nil {
			return starlark.None, err
		}
		if m.Action == nil {
			return starlark.None, fmt.Errorf("no action defined for module %s", name)
		}
		fmt.Println("Running module: ", name)
		result, err := m.Action.Run(args, kwargs)
		if err != nil {
			return starlark.None, err
		}
		var sdiff starlark.String
		diff := result.Diff
		if diff == nil {
			sdiff = starlark.String("")
		}

		name = ""

		ss := starlarkstruct.FromKeywords(
			starlark.String("result"),
			[]starlark.Tuple{
				{
					starlark.String("output"),
					starlark.String(*result.Output),
				},
				{
					starlark.String("success"),
					starlark.Bool(result.Success),
				},
				{
					starlark.String("changed"),
					starlark.Bool(result.Changed),
				},
				{
					starlark.String("diff"),
					sdiff,
				},
				{
					starlark.String("before_result"),
					beforeResult,
				},
			},
		)
		return ss, nil
	}

}

func NewModule(
	name string,
	args []ArgPair,
	action Runnable) *Module {
	m := &Module{}
	m.Args = args
	m.Action = action
	return m
}
