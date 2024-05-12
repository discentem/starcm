package example

import (
	"github.com/discentem/starcm/modules/base"
	"go.starlark.net/starlark"
)

type action struct{}

func (m *action) Run(args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	for _, kwargs := range kwargs {
		iter := kwargs.Iterate()
		defer iter.Done()
		var v starlark.Value
		for iter.Next(&v) {
			if v.String() == "\"str\"" {
				iter.Next(&v)
				return v, nil
			}
		}
	}
	return starlark.None, nil
}

func New() (*base.Module, error) {
	var str string
	a := &action{}
	m, err := base.NewModule(
		"example",
		[]base.ArgPair{
			{Key: "str", Type: &str},
		},
		a,
	)
	return m, err
}
