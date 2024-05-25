package base

import (
	"fmt"
	"log"
	"time"

	"github.com/discentem/starcm/libraries/logging"
	starlarkhelpers "github.com/discentem/starcm/starlark-helpers"
	"github.com/google/deck"
	googlogger "github.com/google/logger"
	"go.starlark.net/starlark"
)

type ArgPair struct {
	Key  string
	Type any
}

type Runnable interface {
	Run(modulenName string, args starlark.Tuple, kwargs []starlark.Tuple) (*Result, error)
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
		name    string
		notIf   starlark.Bool
		onlyIf  starlark.Bool
		timeout string
	)

	// Common arguments automatically available for all modules
	commonArgs := []any{
		"name", &name,
		"only_if??", &onlyIf,
		"not_if?", &notIf,
		"timeout??", &timeout,
	}

	googlogger.SetFlags(log.Lmsgprefix)

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
			logging.Log(name, nil, "info", "skipping module %q because not_if was true", name)
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
			logging.Log(name, nil, "info", "skipping module %q because only_if was false", name)
			return starlark.None, nil
		}

		if m.Action == nil {
			return starlark.None, fmt.Errorf("no action defined for module %s", name)
		}
		deck.Infof("[%s]: Starting...\n", name)

		if !(timeout == "") {
			actionCh := make(chan Result, 1)
			go func() {
				r, err := m.Action.Run(name, args, kwargs)
				if err != nil {
					actionCh <- Result{
						Error: err,
					}
					return
				}
				actionCh <- *r
			}()

			duration, err := time.ParseDuration(timeout)
			if err != nil {
				return starlark.None, fmt.Errorf("error parsing timeout [%s]: %s", timeout, err)
			}
			select {
			case res := <-actionCh:
				return StarlarkResult(res)
			case <-time.After(duration):
				return StarlarkResult(Result{
					Error: fmt.Errorf("timeout %s exceeded", timeout),
				})
			}
		}

		// Run the module-specific behavior
		result, err := m.Action.Run(name, args, kwargs)
		if err != nil {
			return starlark.None, err
		}
		// Convert Result struct to starlark.Value
		return StarlarkResult(*result)
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
