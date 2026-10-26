package base

import (
	"context"

	"github.com/discentem/starcm/libraries/logging"
	"github.com/google/deck"
	"go.starlark.net/starlark"
)

// QueryFunc is what a query does. It returns its answer directly; an error
// stops the run.
type QueryFunc func(inst *FunctionInstance) (starlark.Value, error)

// Query is a starcm function that only reads, such as path.glob() or
// read_file(). It returns its value directly rather than a starcm_result,
// and takes no common arguments: no label, and no only_if or not_if (use
// if). It runs under what_if like any other read, so a dry run sees the
// real state.
//
// It checks its arguments the way a Module does, so queries get the same
// path rule, ~ expansion, positional and keyword handling, and result
// unwrapping. Its FunctionInstance has an empty Label.
type Query struct {
	Name string
	Args []ArgPair
	Run  QueryFunc
	Ctx  context.Context
}

// NewQuery returns the query name, which takes args and runs run.
func NewQuery(ctx context.Context, name string, args []ArgPair, run QueryFunc) *Query {
	return &Query{Name: name, Args: args, Run: run, Ctx: ctx}
}

// Builtin returns the query as a Starlark builtin.
func (q *Query) Builtin() *starlark.Builtin {
	return starlark.NewBuiltin(q.Name, func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		return q.Call(thread, args, kwargs)
	})
}

// Call checks args and kwargs once and runs the query with them. A
// starcm_result passed as an argument stands for its value; one that didn't
// succeed stops the run, since a query has no only_if to skip it.
func (q *Query) Call(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	finalArgs, names, targets := declare(q.Args, 0)
	args, kwargs, failed := unwrapArgs(names, args, kwargs)
	if failed != nil {
		return nil, failed.err(q.Name)
	}
	if err := starlark.UnpackArgs(q.Name, args, kwargs, finalArgs...); err != nil {
		return nil, err
	}
	inst := newInstance(q.Ctx, q.Name, "", false, thread, names, targets, args, kwargs)
	if err := inst.checkRequired(q.Name, q.Args); err != nil {
		return nil, err
	}
	logging.Log("base.go", deck.V(3), "info", "calling %s", q.Name)
	v, err := q.Run(inst)
	logging.Log("base.go", deck.V(3), "info", "finished %s", q.Name)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return starlark.None, nil
	}
	return v, nil
}
