package write

import (
	"context"

	"fmt"

	"io"

	"github.com/discentem/starcm/functions/base"
	starlarkhelpers "github.com/discentem/starcm/starlark-helpers"
	"go.starlark.net/starlark"
)

type action struct {
	w io.Writer
}

func (a *action) Run(ctx context.Context, moduleName string, args starlark.Tuple, kwargs []starlark.Tuple) (*base.Result, error) {
	s, err := starlarkhelpers.FindValueinKwargs(kwargs, "str")
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, fmt.Errorf("str is required in write() module")
	}

	e, err := starlarkhelpers.FindValueInKwargsWithDefault(kwargs, "end", "\n")
	if err != nil {
		return nil, err
	}

	fmt.Fprintf(a.w, "%s%s", *s, *e)
	return &base.Result{
		Output:  s,
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
