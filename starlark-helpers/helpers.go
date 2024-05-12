package starlarkhelpers

import "go.starlark.net/starlark"

type Function func(*starlark.Thread, *starlark.Builtin, starlark.Tuple, []starlark.Tuple) (starlark.Value, error)
