package base

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/discentem/starcm/libraries/logging"
	starlarkhelpers "github.com/discentem/starcm/starlark-helpers"
	"github.com/google/deck"
	googlogger "github.com/google/logger"
	"go.starlark.net/starlark"
)

// ArgPair declares a function-specific argument. Create one with Arg,
// OptionalArg or OptionalArgAllowNone.
type ArgPair struct {
	Key string
	// newTarget returns a fresh pointer for starlark.UnpackArgs to write into,
	// so no call ever sees a value left over from an earlier one.
	newTarget func() any
}

// Arg declares a required function-specific argument that must unpack into
// a T (any type starlark.UnpackArgs accepts a pointer to).
func Arg[T any](name string) ArgPair {
	if strings.HasSuffix(name, "?") {
		panic(fmt.Sprintf("base.Arg(%q): use OptionalArg or OptionalArgAllowNone instead of a ? suffix", name))
	}
	return arg[T](name)
}

// OptionalArg declares an argument callers may leave out. Passing None is an
// error unless T accepts None, as starlark.Value does.
func OptionalArg[T any](name string) ArgPair {
	return arg[T](name + "?")
}

// OptionalArgAllowNone declares an argument callers may leave out or pass as
// None; None counts as leaving it out. Use it for arguments that callers
// forward from their own optional parameters, which default to None.
func OptionalArgAllowNone[T any](name string) ArgPair {
	return arg[T](name + "??")
}

// arg builds an ArgPair whose key uses starlark.UnpackArgs syntax: "name",
// "name?" (optional) or "name??" (optional, None counts as unset). Neither
// suffix makes an argument keyword-only.
func arg[T any](key string) ArgPair {
	return ArgPair{Key: key, newTarget: func() any { return new(T) }}
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
	// Exec, when set, is used instead of Action: it receives the
	// arguments parsed once, as a *FunctionInstance.
	Exec Action
	Ctx  context.Context
}

// Function produces a starlark Function that has common behavior which is useful for all modules like
// only_if, not_if, timeout, working_directory
func (m Module) Function() starlarkhelpers.Function {
	googlogger.SetFlags(log.Lmsgprefix)

	return func(thread *starlark.Thread, builtin *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		// Unpack targets are per call: UnpackArgs leaves targets for omitted
		// optional args untouched, so shared targets would leak values (e.g.
		// timeout, working_directory) from one call into the next.
		var (
			label            string
			notIf            starlark.Bool
			onlyIf           = starlark.Bool(true)
			timeout          string
			workingDirectory starlark.String
			whatIf           starlark.Bool
		)

		finalArgs := make([]any, 0, 2*(len(m.Args)+6))
		// names and targets let a Exec read the parsed values; a plain
		// Action re-reads args and kwargs itself, so for it unpacking here
		// only validates them.
		names := make([]string, 0, len(m.Args)+6)
		targets := make(map[string]any, len(m.Args))
		// Add arguments that are specific to this module.
		for _, arg := range m.Args {
			target := arg.newTarget()
			finalArgs = append(finalArgs, arg.Key, target)
			names = append(names, arg.Key)
			targets[argName(arg.Key)] = target
		}
		// Common arguments automatically available for all Starcm functions
		finalArgs = append(finalArgs,
			"label", &label,
			"only_if?", &onlyIf,
			"not_if?", &notIf,
			"timeout?", &timeout,
			"working_directory?", &workingDirectory,
			"what_if?", &whatIf,
		)
		names = append(names, "label", "only_if?", "not_if?", "timeout?", "working_directory?", "what_if?")

		// Name argument errors after this call's label when it has one.
		fnName := builtin.Name()
		if l, err := starlarkhelpers.FindValueinKwargs(kwargs, "label"); err == nil && l != nil {
			fnName = *l
		}
		if err := starlark.UnpackArgs(
			fnName,
			args,
			kwargs,
			finalArgs...,
		); err != nil {
			return starlark.None, err
		}
		if _, err := starlarkhelpers.FindValueinKwargs(kwargs, "label"); err != nil {
			return starlark.None, err
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
		if m.Action == nil && m.Exec == nil {
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
		var r *Result
		var err error
		if m.Exec != nil {
			logging.Log("base.go", deck.V(3), "info", "calling m.Exec.Execute(label=%q)", label)
			inst := newInstance(ctx, builtin.Name(), label, bool(whatIf), thread, names, targets, args, kwargs)
			if err := inst.checkRequired(fnName, m.Args); err != nil {
				return starlark.None, err
			}
			r, err = m.Exec.Execute(inst)
		} else {
			logging.Log("base.go", deck.V(3), "info", "calling m.Action.Run(ctx, workingDirectory=%q, label=%q, args, kwargs)", finalWorkingDir, label)
			r, err = m.Action.Run(ctx, finalWorkingDir, label, thread, args, kwargs)
		}
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

// NewActionModule is NewModule for actions that take their arguments parsed
// once, as a *FunctionInstance.
func NewActionModule(ctx context.Context, fnType string, args []ArgPair, action Action) *Module {
	return &Module{Type: fnType, Args: args, Exec: action, Ctx: ctx}
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
