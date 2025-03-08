package starlarkhelpers

import "go.starlark.net/starlark"

func GoDictToStarlarkDict(dict map[string]any) *starlark.Dict {
	starlarkDict := starlark.NewDict(len(dict))
	for key, value := range dict {
		starlarkDict.SetKey(starlark.String(key), starlark.String(value.(string)))
	}
	return starlarkDict
}
