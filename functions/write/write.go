package write

import (
	"context"
	"errors"

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

// text renders v the way print() does: strings as their raw contents,
// everything else as its Starlark representation.
func text(v starlark.Value) string {
	if s, ok := v.(starlark.String); ok {
		return s.GoString()
	}
	return v.String()
}

func (a *writeAction) Run(ctx context.Context, workingDirectory string, moduleName string, thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (*base.Result, error) {
	// The value comes from either the positional argument or str=.
	// Passing both is rejected by base.Module's argument unpacking.
	var v starlark.Value
	switch {
	case len(args) > 1:
		return nil, fmt.Errorf("write takes at most one positional argument, got %d", len(args))
	case len(args) == 1:
		v = args[0]
	default:
		s, err := starlarkhelpers.FindRawValueInKwargs(kwargs, "str")
		if err != nil && !errors.Is(err, starlarkhelpers.ErrIndexNotFound) {
			return nil, err
		}
		if s == nil {
			return nil, errors.New("write requires a value, either positionally or as str=")
		}
		v = s
	}

	e, err := starlarkhelpers.FindValueInKwargsWithDefault(kwargs, "end", "\n")
	if err != nil {
		return nil, err
	}

	fmt.Fprintf(a.w, "%s%s", text(v), *e)
	return &base.Result{
		Label:   moduleName,
		Return:  v,
		Success: true,
		Changed: false,
	}, nil
}

func New(ctx context.Context, w io.Writer) *base.Module {
	return base.NewModule(
		ctx,
		"write",
		[]base.ArgPair{
			// str is declared first so write("x") binds "x" to str. Optional
			// arguments aren't keyword-only, so if end came first a positional
			// value would bind to end.
			base.OptionalArg[starlark.Value]("str"),
			base.OptionalArgAllowNone[string]("end"),
		},
		&writeAction{
			w: w,
		},
	)
}
