package base

import (
	"fmt"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

type Result struct {
	Label   string
	Changed bool
	Success bool
	Diff    *string
	Message *string
	Error   error
	Return  starlark.Value
}

func (r Result) ToStarlark() (*starlarkstruct.Struct, error) {
	msg := ""
	if r.Message != nil {
		msg = *r.Message
	}

	ret := r.Return
	if r.Return == nil {
		ret = starlark.None
	}

	// Freeze payload if it's a container
	if f, ok := ret.(interface{ Freeze() }); ok {
		f.Freeze()
	}

	fields := starlark.StringDict{
		"label":   starlark.String(r.Label),
		"changed": starlark.Bool(r.Changed),
		"success": starlark.Bool(r.Success),
		"message": starlark.String(msg),
		"error":   starlark.String(fmt.Sprint(r.Error)),
		"return":  ret,
	}

	s := starlarkstruct.FromStringDict(starlark.String("starcm_result"), fields)
	s.Freeze()
	return s, nil
}
