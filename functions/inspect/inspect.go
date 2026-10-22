package inspect

import (
	"context"
	encodingjson "encoding/json"

	"github.com/discentem/starcm/functions/base"
	"github.com/discentem/starcm/libraries/inspect"
	"github.com/spf13/afero"
	"go.starlark.net/lib/json"
	"go.starlark.net/starlark"
)

type inspectAction struct {
	fsys afero.Fs
}

var _ base.Action = (*inspectAction)(nil)

// Execute reports what the file at path loads and calls, without running it.
// value has the same shape as one entry of `starcm inspect`'s JSON. path must
// be absolute; a leading ~ is expanded.
func (a *inspectAction) Execute(inst *base.FunctionInstance) (*base.Result, error) {
	p, _, err := inst.AbsPath("path")
	if err != nil {
		return nil, err
	}
	src, err := afero.ReadFile(a.fsys, p)
	if err != nil {
		return nil, err
	}
	report, err := inspect.File(p, src)
	if err != nil {
		return nil, err
	}
	// Round-trip through JSON so value matches the CLI output exactly.
	b, err := encodingjson.Marshal(report)
	if err != nil {
		return nil, err
	}
	v, err := starlark.Call(inst.Thread, json.Module.Members["decode"], starlark.Tuple{starlark.String(b)}, nil)
	if err != nil {
		return nil, err
	}
	return &base.Result{Label: inst.Label, Success: true, Return: v}, nil
}

// New returns the inspect() builtin.
//
//	r = inspect(here("bootstrap.star"), label = "check bootstrap").value
//	r["loads"], r["calls"], r["values"]
func New(ctx context.Context, fsys afero.Fs) *base.Module {
	return base.NewModule(ctx, "inspect", []base.ArgPair{base.Arg[string]("path")}, &inspectAction{fsys: fsys})
}
