package base

import (
	"github.com/discentem/starcm/libraries/logging"
	"github.com/google/deck"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

type Result struct {
	Name    *string
	Output  *string
	Error   error
	Success bool
	Changed bool
	Diff    *string
	Comment string
}

func StarlarkResult(r Result) (starlark.Value, error) {
	logging.Log("", deck.V(3), "info", "StarlarkResult: %v", r)
	var sname starlark.String
	if r.Name != nil {
		sname = starlark.String(*r.Name)
	} else {
		sname = starlark.String("not_provided")
	}

	var soutput starlark.String
	if r.Output != nil {
		soutput = starlark.String(*r.Output)
	} else {
		soutput = starlark.String("")
	}

	var serror starlark.String
	if r.Error != nil {
		serror = starlark.String(r.Error.Error())
	} else {
		serror = starlark.String("")
	}

	var sdiff starlark.String
	diff := r.Diff
	if diff == nil {
		sdiff = starlark.String("")
	}
	return starlarkstruct.FromKeywords(
		starlark.String("result"),
		[]starlark.Tuple{
			{
				starlark.String("name"),
				sname,
			},
			{
				starlark.String("output"),
				soutput,
			},
			{
				starlark.String("error"),
				serror,
			},
			{
				starlark.String("success"),
				starlark.Bool(r.Success),
			},
			{
				starlark.String("changed"),
				starlark.Bool(r.Changed),
			},
			{
				starlark.String("diff"),
				sdiff,
			},
		},
	), nil
}
