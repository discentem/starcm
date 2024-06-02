package base

import (
	"context"
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
	Run(ctx context.Context, moduleName string, args starlark.Tuple, kwargs []starlark.Tuple) (*Result, error)
}

type Module struct {
	Args   []ArgPair
	Action Runnable
	Ctx    context.Context
}

// Function produces a starlark Function that has common behavior that useful for all modules like only_if, not_if, and after
func (m Module) Function() starlarkhelpers.Function {
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
		"only_if?", &onlyIf,
		"not_if?", &notIf,
		"timeout?", &timeout,
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
		idx, err := starlarkhelpers.FindValueOfKeyInKwargs(kwargs, "not_if")
		if err != nil {
			return nil, err
		}
		if idx != starlarkhelpers.IndexNotFound {
			logging.Log(name, deck.V(2), "info", "not_if was: %q", kwargs[idx][1].String())
		} else {
			notIf = starlark.False
		}

		// skip module if not_if is true
		if notIf.Truth() {
			logging.Log(name, nil, "info", "skipping module %q because not_if was true", name)
			return starlark.None, nil
		}

		idx, err = starlarkhelpers.FindValueOfKeyInKwargs(kwargs, "only_if")
		if err != nil {
			return nil, err
		}
		if idx != starlarkhelpers.IndexNotFound {
			logging.Log(name, deck.V(2), "info", "only_if was: %q", kwargs[idx][1].String())
		} else {
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
			dur, err := time.ParseDuration(timeout)
			if err != nil {
				return starlark.None, fmt.Errorf("error parsing timeout [%s]: %s", timeout, err)
			}
			ctx, cancel := context.WithTimeout(m.Ctx, dur)
			defer cancel()
			r, err := m.Action.Run(ctx, name, args, kwargs)
			if r == nil && err != nil {
				return starlark.None, err
			}
			return StarlarkResult(*r)
		}

		// Run the module-specific behavior
		result, err := m.Action.Run(m.Ctx, name, args, kwargs)
		if result == nil && err != nil {
			return starlark.None, err
		}
		// Convert Result struct to starlark.Value
		return StarlarkResult(*result)
	}

}

func NewModule(ctx context.Context, name string, args []ArgPair, action Runnable) *Module {
	m := &Module{
		Args:   args,
		Action: action,
		Ctx:    ctx,
	}
	m.Args = args
	m.Action = action
	return m
}
