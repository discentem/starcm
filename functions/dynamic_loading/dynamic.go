package loading

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/discentem/starcm/functions/base"
	"github.com/discentem/starcm/libraries/logging"
	"github.com/google/deck"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

type LoadAction struct{}

var _ base.Action = (*LoadAction)(nil)

// Execute loads module, a bundled resource: a relative path resolves next
// to the .star file the call is written in, and a // path against the
// loader's root.
func (a *LoadAction) Execute(inst *base.FunctionInstance) (*base.Result, error) {
	thread := inst.Thread
	if thread == nil || thread.Load == nil {
		return nil, fmt.Errorf("%s: thread.Load is nil", inst.Label)
	}
	module, _ := base.Get[string](inst, "module")
	logging.Log("load_dynamic", deck.V(4), "info", "passing module path to thread.Load: %q", module)

	if !strings.HasPrefix(module, "//") {
		p, _, err := inst.ResourcePath("module")
		if err != nil {
			return nil, err
		}
		module = filepath.ToSlash(p)
	}
	d, err := loadGlobals(thread, module)
	if err != nil {
		return nil, err
	}

	return &base.Result{
		Changed: false,
		Success: true,
		Message: func() *string {
			msg := fmt.Sprintf("module %q loaded successfully", module)
			return &msg
		}(),
		Return: d,
	}, nil
}

// loadGlobals runs module through thread.Load and returns its globals as a
// frozen dict.
func loadGlobals(thread *starlark.Thread, module string) (*starlark.Dict, error) {
	globals, err := thread.Load(thread, module)
	if err != nil {
		switch e := err.(type) {
		case *starlark.EvalError:
			return nil, fmt.Errorf("load failed:\n%s", e.Backtrace())
		case *syntax.Error:
			return nil, fmt.Errorf("syntax error:\n%s", e.Error())
		default:
			return nil, err
		}
	}
	d := starlark.NewDict(len(globals))
	for k, vv := range globals {
		if err := d.SetKey(starlark.String(k), vv); err != nil {
			return nil, err
		}
	}
	d.Freeze()
	return d, nil
}

func New(ctx context.Context) *base.Module {
	return base.NewModule(
		ctx,
		"load_dynamic",
		[]base.ArgPair{
			base.Arg[string]("module"),
		},
		&LoadAction{},
	)
}
