package loading

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/discentem/starcm/functions/base"
	"github.com/discentem/starcm/libraries/logging"
	starlarkhelpers "github.com/discentem/starcm/starlark-helpers"
	"github.com/google/deck"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

type LoadAction struct{}

func (a *LoadAction) Run(
	ctx context.Context,
	workingDirectory string,
	moduleName string,
	thread *starlark.Thread,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (*base.Result, error) {
	if thread == nil || thread.Load == nil {
		return nil, fmt.Errorf("%s: thread.Load is nil", moduleName)
	}
	var module string
	v, err := starlarkhelpers.FindValueinKwargs(kwargs, "module")
	if err != nil {
		if args.Len() > 1 {
			return nil, fmt.Errorf("%s: expected at most one positional argument", moduleName)
		}
		module = args.Index(0).(starlark.String).GoString()
	} else {
		module = *v
	}

	logging.Log("load_dynamic", deck.V(4), "info", "passing module path to thread.Load: %q", module)

	// For relative paths (not //, not absolute), resolve them relative to the caller's directory
	// Use At(1) to skip the current load_dynamic call and get to the actual caller
	if !filepath.IsAbs(module) && !strings.HasPrefix(module, "//") && len(thread.CallStack()) > 1 {
		callerPos := thread.CallStack().At(1)
		callerDir := filepath.Dir(callerPos.Pos.Filename())
		module = filepath.Join(callerDir, module)
		module = filepath.ToSlash(module)
	}
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

func New(ctx context.Context) *base.Module {
	var module string

	return base.NewModule(
		ctx,
		"load",
		[]base.ArgPair{
			{Key: "module",
				Type: &module,
			},
		},
		&LoadAction{},
	)
}
