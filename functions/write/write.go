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
	s, err := starlarkhelpers.FindValueInKwargsWithDefault(kwargs, "str", "")
	if err != nil {
		for _, arg := range args {
			fmt.Fprintf(a.w, "%s", arg)
		}
		return &base.Result{
			Label:   moduleName,
			Success: true,
			Changed: false,
			Error:   err,
			Return:  starlark.String(*s),
		}, err
	}

	e, err := starlarkhelpers.FindValueInKwargsWithDefault(kwargs, "end", "\n")
	if err != nil {
		return nil, err
	}

	fmt.Fprintf(a.w, "%s%s", *s, *e)
	return &base.Result{
		Label:   moduleName,
		Return:  starlark.String(*s),
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
		&writeAction{
			w: w,
		},
	)
}
