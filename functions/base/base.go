package base

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
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
	Run(
		ctx context.Context,
		workingDirectory string,
		moduleName string,
		args starlark.Tuple,
		kwargs []starlark.Tuple,
	) (*Result, error)
}

type Module struct {
	Type   string
	Args   []ArgPair
	Action Runnable
	Ctx    context.Context
}

// Function produces a starlark Function that has common behavior which is useful for all modules like
// only_if, not_if, timeout, working_directory
func (m Module) Function() starlarkhelpers.Function {
	finalArgs := make([]any, 0)
	// Add arguments that are specific to this module
	for _, arg := range m.Args {
		finalArgs = append(finalArgs, arg.Key, arg.Type)
	}
	var (
		name             string
		notIf            starlark.Bool
		onlyIf           starlark.Bool
		timeout          string
		workingDirectory starlark.String
		whatIf           starlark.Bool
	)

	// Common arguments automatically available for all Starcm functions
	commonArgs := []any{
		"name", &name,
		"only_if?", &onlyIf,
		"not_if?", &notIf,
		"timeout?", &timeout,
		"working_directory?", &workingDirectory,
		"what_if?", &whatIf,
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
		_, err := starlarkhelpers.FindValueinKwargs(kwargs, "name")
		if err != nil {
			return starlark.None, fmt.Errorf("%v for %q argument", err, "name")
		}

		notIf, err := starlarkhelpers.FindBoolInKwargs(kwargs, "not_if", false)
		if err != nil {
			return starlark.None, fmt.Errorf("%v for %q argument", err, "not_if")
		}

		// skip module if not_if is true
		if notIf {
			logging.Log(name, nil, "info", "skipping %s(name=%q) because not_if was true", m.Type, name)
			sr, err := StarlarkResult(Result{})
			if err != nil {
				return nil, err
			}
			return sr, nil
		}

		onlyIf, err := starlarkhelpers.FindBoolInKwargs(kwargs, "only_if", true)
		if err != nil {
			return starlark.None, fmt.Errorf("%v for %q argument", err, "only_if")
		}

		if !onlyIf {
			logging.Log(name, nil, "info", "skipping %s(name=%q) because only_if was false", m.Type, name)
			sr, err := StarlarkResult(Result{})
			if err != nil {
				return nil, err
			}
			return sr, nil
		}

		var finalWorkingDir string
		// If working directory is not set, use the parent directory of the file that called the module
		if workingDirectory.Truth() == starlark.False {
			if len(thread.CallStack()) > 0 {
				dirName := filepath.Dir(thread.CallStack().At(1).Pos.Filename())
				finalWorkingDir = filepath.Join(dirName, workingDirectory.GoString())
			}

		} else {
			finalWorkingDir = workingDirectory.GoString()
		}

		if m.Action == nil {
			return starlark.None, fmt.Errorf("no action defined for module %s", name)
		}

		var ctx context.Context
		var cancel context.CancelFunc
		if !(timeout == "") {
			dur, err := time.ParseDuration(timeout)
			if err != nil {
				return starlark.None, fmt.Errorf("error parsing timeout [%s]: %s", timeout, err)
			}
			ctx, cancel = context.WithTimeout(m.Ctx, dur)
			defer cancel()
		} else {
			ctx = m.Ctx
		}
		logging.Log("base.go", deck.V(3), "info", "calling m.Action.Run(ctx, workingDirectory=%q, moduleName=%q, args, kwargs)", finalWorkingDir, name)
		r, err := m.Action.Run(ctx, finalWorkingDir, name, args, kwargs)
		if r == nil && err != nil {
			return starlark.None, err
		}
		if r == nil {
			return starlark.None, fmt.Errorf("no result returned from module %s", name)
		}
		return StarlarkResult(*r)
	}

}

func NewModule(ctx context.Context, fnType string, args []ArgPair, action Runnable) *Module {
	m := &Module{
		Type:   fnType,
		Args:   args,
		Action: action,
		Ctx:    ctx,
	}
	m.Args = args
	m.Action = action
	return m
}
