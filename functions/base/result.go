package base

import (
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
	var sname starlark.String
	if r.Name != nil {
		sname = starlark.String(*r.Name)
	} else {
		sname = starlark.String("not_provided")
	}

	var soutput starlark.String
	if r.Output != nil {
		out := *r.Output
		// soutput = starlark.String(strings.TrimSuffix(out, "\n"))
		soutput = starlark.String(out)
	} else {
		soutput = starlark.String("")
	}

	var serror starlark.Value
	if r.Error != nil {
		serror = starlark.String(r.Error.Error())
	} else {
		serror = starlark.None
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
