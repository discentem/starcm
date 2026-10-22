package write

import (
	"context"
	"errors"

	"fmt"

	"io"

	"github.com/discentem/starcm/functions/base"
	"go.starlark.net/starlark"
)

type writeAction struct {
	w io.Writer
}

var _ base.Action = (*writeAction)(nil)

// text renders v the way print() does: strings as their raw contents,
// everything else as its Starlark representation.
func text(v starlark.Value) string {
	if s, ok := v.(starlark.String); ok {
		return s.GoString()
	}
	return v.String()
}

func (a *writeAction) Execute(inst *base.FunctionInstance) (*base.Result, error) {
	v, given := base.Get[starlark.Value](inst, "str")
	if !given {
		return nil, errors.New("write requires a value, either positionally or as str=")
	}
	end, given := base.Get[string](inst, "end")
	if !given {
		end = "\n"
	}
	fmt.Fprintf(a.w, "%s%s", text(v), end)
	return &base.Result{
		Label:   inst.Label,
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
