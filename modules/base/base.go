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

func (m *Module) Function() (starlarkhelpers.Function, error) {
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

	finalArgs = append(
		finalArgs,
		"name", &name,
		"only_if?", &onlyIf,
		"after?", &after,
		"not_if?", &notIf,
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
			fmt.Println(afterResult)
		}
		if m.Action == nil {
			return starlark.None, fmt.Errorf("no action defined for module %s", name)
		}
		fmt.Println("Running module: ", name)
		return m.Action.Run(args, kwargs)
	}, nil

}

func NewModule(
	name string,
	args []ArgPair,
	action Runnable) (*Module, error) {
	m := &Module{}
	m.Args = args
	m.Action = action
	return m, nil
}
