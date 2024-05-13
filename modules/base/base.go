package base

import (
	"fmt"

	starlarkhelpers "github.com/discentem/starcm/starlark-helpers"
	"go.starlark.net/starlark"
)

type ArgPair struct {
	Key  string
	Type any
}

type Runnable interface {
	Run(args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error)
}

type Module struct {
	Args   []ArgPair
	Action Runnable
}

// Function produces a starlark Function that has common behavior that useful for all modules like only_if, not_if, and after
func (m *Module) Function() starlarkhelpers.Function {
	finalArgs := make([]any, 0)
	for _, arg := range m.Args {
		finalArgs = append(finalArgs, arg.Key, arg.Type)
	}
	var (
		name   string
		notIf  starlark.Bool
		onlyIf starlark.Bool
		after  starlark.Callable
	)

	// Common arguments automatically available for all modules
	commonArgs := []any{
		"name", &name,
		"only_if?", &onlyIf,
		"after?", &after,
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

		isDefined := func(value starlark.Value) bool {
			return value != starlark.None
		}
		// If onlyIf is not defined, it is assumed to be true
		if !isDefined(onlyIf) {
			onlyIf = starlark.True
		}

		if onlyIf.Truth() == starlark.True {
			fmt.Printf("skipping module %s because only_if was false", name)
			return starlark.None, nil
		}

		if after != nil {
			afterResult, err := starlark.Call(thread, after, args, kwargs)
			if err != nil {
				return starlark.None, err
			}
			if afterResult.String() != `"<nil>"` {
				fmt.Println("after result:", afterResult)
			}
		}
		if m.Action == nil {
			return starlark.None, fmt.Errorf("no action defined for module %s", name)
		}
		fmt.Println("Running module: ", name)
		return m.Action.Run(args, kwargs)
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
