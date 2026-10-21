package starlarkhelpers

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

func TestValueToGo(t *testing.T) {
	inner := starlark.NewDict(1)
	require.NoError(t, inner.SetKey(starlark.String("city"), starlark.String("london")))
	set := starlark.NewSet(1)
	require.NoError(t, set.Insert(starlark.String("x")))

	tests := []struct {
		name string
		in   starlark.Value
		want any
	}{
		{"none", starlark.None, nil},
		{"string", starlark.String("s"), "s"},
		{"int", starlark.MakeInt(7), int64(7)},
		{"float", starlark.Float(1.5), 1.5},
		{"bool", starlark.True, true},
		{"empty list", starlark.NewList(nil), []any{}},
		{"tuple", starlark.Tuple{starlark.MakeInt(1), starlark.None}, []any{int64(1), nil}},
		{"set", set, []any{"x"}},
		{"nested dict", inner, map[string]any{"city": "london"}},
		{
			"struct",
			starlarkstruct.FromStringDict(starlark.String("r"), starlark.StringDict{
				"value":   starlark.NewList([]starlark.Value{inner}),
				"success": starlark.True,
			}),
			map[string]any{"value": []any{map[string]any{"city": "london"}}, "success": true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValueToGo(tt.in)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestValueToGo_Errors(t *testing.T) {
	intKey := starlark.NewDict(1)
	require.NoError(t, intKey.SetKey(starlark.MakeInt(1), starlark.String("one")))
	outer := starlark.NewDict(1)
	require.NoError(t, outer.SetKey(starlark.String("m"), intKey))
	_, err := ValueToGo(outer)
	require.EqualError(t, err, `"m": dict keys must be strings, got int key 1`)

	fn := starlark.NewBuiltin("f", nil)
	_, err = ValueToGo(starlark.NewList([]starlark.Value{starlark.None, fn}))
	require.EqualError(t, err, `[1]: unsupported type builtin_function_or_method`)

	huge := starlark.MakeInt(1).Lsh(70)
	_, err = ValueToGo(huge)
	require.Error(t, err)
}

func TestFindOptionalBoolInKwargs(t *testing.T) {
	kw := func(v starlark.Value) []starlark.Tuple {
		return []starlark.Tuple{{starlark.String("require_notarized"), v}}
	}
	for _, c := range []struct {
		name   string
		kwargs []starlark.Tuple
		want   bool
	}{
		{"missing", nil, true},
		{"None", kw(starlark.None), true},
		{"True", kw(starlark.True), true},
		{"False", kw(starlark.False), false},
	} {
		got, err := FindOptionalBoolInKwargs(c.kwargs, "require_notarized", true)
		require.NoError(t, err, c.name)
		require.Equal(t, c.want, got, c.name)
	}
}
