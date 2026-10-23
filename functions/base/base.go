package base

import (
	"context"
	"fmt"
	"log"
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

// Module is a starcm function: its name, its own arguments, and the Action
// that runs it.
type Module struct {
	// Type is the function's name, as scripts call it.
	Type   string
	Args   []ArgPair
	Action Action
	Ctx    context.Context
}

// Function produces a starlark Function that adds the common arguments
// every starcm function takes (label, only_if, not_if, timeout, what_if)
// and returns the Result as a starcm_result.
func (m Module) Function() starlarkhelpers.Function {
	googlogger.SetFlags(log.Lmsgprefix)

	return func(thread *starlark.Thread, builtin *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		r, err := m.Call(thread, args, kwargs)
		if r == nil && err != nil {
			return starlark.None, err
		}
		starResult, err := r.ToStarlark()
		if err != nil {
			return starlark.None, fmt.Errorf("error converting result to starlark for %s: %v", m.Type, err)
		}
		return starResult, nil
	}
}

// Call checks args and kwargs once and runs the Action with them. A call
// skipped by only_if or not_if returns an empty Result. Function wraps it;
// Go tests call it directly to get the Result.
func (m Module) Call(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (*Result, error) {
	// Unpack targets are per call: UnpackArgs leaves targets for omitted
	// optional args untouched, so shared targets would leak values (e.g.
	// timeout) from one call into the next.
	var (
		label   string
		notIf   starlark.Bool
		onlyIf  = starlark.Bool(true)
		timeout string
		whatIf  starlark.Bool
	)

	finalArgs := make([]any, 0, 2*(len(m.Args)+5))
	// names and targets let the FunctionInstance read the parsed values.
	names := make([]string, 0, len(m.Args)+5)
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
		"what_if?", &whatIf,
	)
	names = append(names, "label", "only_if?", "not_if?", "timeout?", "what_if?")

	// Name argument errors after this call's label when it has one.
	fnName := m.Type
	if l, err := starlarkhelpers.FindValueinKwargs(kwargs, "label"); err == nil && l != nil {
		fnName = *l
	}
	// A starcm_result passed as an argument stands for its value. One that
	// didn't succeed stops the run, unless only_if or not_if skip the call.
	args, kwargs, failed := unwrapArgs(names, args, kwargs)
	if err := starlark.UnpackArgs(fnName, args, kwargs, finalArgs...); err != nil {
		if failed == nil {
			return nil, err
		}
		// The failed result's value didn't fit the argument. A call that's
		// skipped anyway shouldn't fail over it.
		if skippedByKwargs(kwargs) {
			logging.Log(fnName, nil, "info", "skipping %s(label=%q) because of only_if or not_if", m.Type, fnName)
			return &Result{}, nil
		}
		return nil, failed.err(fnName)
	}
	if _, err := starlarkhelpers.FindValueinKwargs(kwargs, "label"); err != nil {
		return nil, err
	}

	if notIf {
		logging.Log(label, nil, "info", "skipping %s(label=%q) because not_if was true", m.Type, label)
		return &Result{}, nil
	}
	if !onlyIf {
		logging.Log(label, nil, "info", "skipping %s(label=%q) because only_if was false", m.Type, label)
		return &Result{}, nil
	}
	if failed != nil {
		return nil, failed.err(fnName)
	}
	if m.Action == nil {
		return nil, fmt.Errorf("no action defined for module %s", label)
	}

	ctx := m.Ctx
	if timeout != "" {
		dur, err := time.ParseDuration(timeout)
		if err != nil {
			return nil, fmt.Errorf("error parsing timeout [%s]: %s", timeout, err)
		}
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(m.Ctx, dur)
		defer cancel()
	}
	inst := newInstance(ctx, m.Type, label, bool(whatIf), thread, names, targets, args, kwargs)
	if err := inst.checkRequired(fnName, m.Args); err != nil {
		return nil, err
	}
	logging.Log("base.go", deck.V(3), "info", "calling %s(label=%q)", m.Type, label)
	r, err := m.Action.Execute(inst)
	logging.Log("base.go", deck.V(3), "info", "finished %s(label=%q)", m.Type, label)
	return r, err
}

// NewModule returns the starcm function fnType, which takes args (plus the
// common arguments) and runs action.
func NewModule(ctx context.Context, fnType string, args []ArgPair, action Action) *Module {
	return &Module{Type: fnType, Args: args, Action: action, Ctx: ctx}
}
