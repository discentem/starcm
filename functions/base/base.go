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
		label string,
		thread *starlark.Thread,
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
		label            string
		notIf            starlark.Bool
		onlyIf           starlark.Bool
		timeout          string
		workingDirectory starlark.String
		whatIf           starlark.Bool
	)

	// Common arguments automatically available for all Starcm functions
	commonArgs := []any{
		"label", &label,
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
			label,
			args,
			kwargs,
			finalArgs...,
		); err != nil {
			return starlark.None, err
		}
		_, err := starlarkhelpers.FindValueinKwargs(kwargs, "label")
		if err != nil {
			return starlark.None, err
		}

		notIf, err := starlarkhelpers.FindBoolInKwargs(kwargs, "not_if", false)
		if err != nil {
			return starlark.None, fmt.Errorf("%v for %q argument", err, "not_if")
		}

		// skip module if not_if is true
		if notIf {
			logging.Log(label, nil, "info", "skipping %s(label=%q) because not_if was true", m.Type, label)
			sr, err := Result{}.ToStarlark()
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
			logging.Log(label, nil, "info", "skipping %s(label=%q) because only_if was false", m.Type, label)
			sr, err := Result{}.ToStarlark()
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
			return starlark.None, fmt.Errorf("no action defined for module %s", label)
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
		logging.Log("base.go", deck.V(3), "info", "calling m.Action.Run(ctx, workingDirectory=%q, label=%q, args, kwargs)", finalWorkingDir, label)
		r, err := m.Action.Run(ctx, finalWorkingDir, label, thread, args, kwargs)
		if r == nil && err != nil {
			return starlark.None, err
		}
		starResult, err := r.ToStarlark()
		if err != nil {
			return starlark.None, fmt.Errorf("error converting result to starlark for module %s: %v", label, err)
		}
		logging.Log("base.go", deck.V(3), "info", "finished m.Action.Run for label=%q", label)
		return starResult, nil
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
