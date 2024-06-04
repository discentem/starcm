package write

import (
	"context"

	"fmt"

	"io"

	"github.com/discentem/starcm/functions/base"
	"github.com/discentem/starcm/starlark-helpers"
	"go.starlark.net/starlark"
)

type action struct {
	w io.Writer
}

func (a *action) Run(ctx context.Context, moduleName string, args starlark.Tuple, kwargs []starlark.Tuple) (*base.Result, error) {
	idx, err := starlarkhelpers.FindValueOfKeyInKwargs(kwargs, "str")
	if err != nil {
		return nil, err
	}
	if idx == starlarkhelpers.IndexNotFound {
		return nil, fmt.Errorf("'str' was not found in kwargs")
	}

	s, _, _, err := starlarkhelpers.Unquote(kwargs[idx][1].String())
	if err != nil {
		return nil, err
	}

	idx, err = starlarkhelpers.FindValueOfKeyInKwargs(kwargs, "end")
	if err != nil {
		return nil, err
	}
	var e string
	if idx == starlarkhelpers.IndexNotFound {
		e = "\n"
	} else {
		e, _, _, err = starlarkhelpers.Unquote(kwargs[idx][1].String())
		if err != nil {
			return nil, err
		}
	}

	fmt.Fprintf(a.w, "%s%s", s, e)
	return &base.Result{
		Output:  &s,
		Success: true,
		Changed: false,
	}, nil
}

func New(ctx context.Context, w io.Writer) *base.Module {
	var (
		str string
		end string
	)

	return base.NewModule(
		ctx,
		"write",
		[]base.ArgPair{
			{Key: "str", Type: &str},
			{Key: "end??", Type: &end},
		},
		&action{
			w: w,
		},
	)
}
