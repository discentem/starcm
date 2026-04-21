package write

import (
	"context"

	"fmt"

	"io"

	"github.com/discentem/starcm/functions/base"
	starlarkhelpers "github.com/discentem/starcm/starlark-helpers"
	"go.starlark.net/starlark"
)

type writeAction struct {
	w io.Writer
}

var _ base.Runnable = (*writeAction)(nil)

func (a *writeAction) Run(ctx context.Context, workingDirectory string, moduleName string, thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (*base.Result, error) {
	// Print all positional arguments
	if len(args) > 0 {
		for _, arg := range args {
			fmt.Fprint(a.w, arg)
		}
		e, err := starlarkhelpers.FindValueInKwargsWithDefault(kwargs, "end", "\n")
		if err != nil {
			return nil, err
		}
		fmt.Fprint(a.w, *e)
		return &base.Result{
			Label:   moduleName,
			Success: true,
			Changed: false,
		}, nil
	}

	// Otherwise, get the str parameter from kwargs
	s, err := starlarkhelpers.FindRawValueInKwargs(kwargs, "str")
	if err != nil {
		return nil, fmt.Errorf("str parameter not found: %w", err)
	}

	if s == nil {
		return nil, fmt.Errorf("str parameter cannot be nil")
	}

	e, err := starlarkhelpers.FindValueInKwargsWithDefault(kwargs, "end", "\n")
	if err != nil {
		return nil, err
	}

	fmt.Fprintf(a.w, "%s%s", s.String(), *e)
	return &base.Result{
		Label:   moduleName,
		Return:  s,
		Success: true,
		Changed: false,
	}, nil
}

func New(ctx context.Context, w io.Writer) *base.Module {
	var (
		end string
	)

	return base.NewModule(
		ctx,
		"write",
		[]base.ArgPair{
			{Key: "end??", Type: &end},
		},
		&writeAction{
			w: w,
		},
	)
}
