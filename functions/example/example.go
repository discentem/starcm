package example

import (
	"github.com/discentem/starcm/functions/base"
	"go.starlark.net/starlark"
)

type action struct{}

func (m *action) Run(moduleName string, args starlark.Tuple, kwargs []starlark.Tuple) (*base.Result, error) {
	for _, kwargs := range kwargs {
		iter := kwargs.Iterate()
		defer iter.Done()
		var v starlark.Value
		for iter.Next(&v) {
			if v.String() == "\"str\"" {
				iter.Next(&v)
				return &base.Result{
					Output: func() *string {
						s := v.String()
						return &s
					}(),
					Success: true,
					Changed: true,
					Diff:    nil,
					Comment: v.String(),
				}, nil
			}
		}
	}
	return &base.Result{
		Output:  nil,
		Success: false,
		Changed: false,
		Diff:    nil,
		Comment: "str not found",
	}, nil
}

func New() *base.Module {
	var str string
	a := &action{}
	return base.NewModule(
		"example",
		[]base.ArgPair{
			{Key: "str", Type: &str},
		},
		a,
	)
}
