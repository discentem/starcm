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

func (a *writeAction) Run(ctx context.Context, workingDirectory string, moduleName string, args starlark.Tuple, kwargs []starlark.Tuple) (base.ActionReturn, error) {
	s, err := starlarkhelpers.FindValueinKwargs(kwargs, "str")
	if err != nil {
		// If the key is not found in kwargs, just assume only values have been passed and print them all
		for _, arg := range args {
			fmt.Fprintf(a.w, "%s", arg)
		}
		return &base.Result{
			Name:    &moduleName,
			Output:  s,
			Success: true,
			Changed: false,
		}, nil
	}

	e, err := starlarkhelpers.FindValueInKwargsWithDefault(kwargs, "end", "\n")
	if err != nil {
		return nil, err
	}

	fmt.Fprintf(a.w, "%s%s", *s, *e)
	return &base.Result{
		Name:    &moduleName,
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
		&writeAction{
			w: w,
		},
	)
}
