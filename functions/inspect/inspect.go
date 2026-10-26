package inspect

import (
	"context"
	encodingjson "encoding/json"

	"github.com/discentem/starcm/functions/base"
	"github.com/discentem/starcm/libraries/inspect"
	"github.com/spf13/afero"
	"go.starlark.net/lib/json"
	"go.starlark.net/starlark"
)

// New returns inspect(), a query that reports what the file at path loads
// and calls, without running it. Its value has the same shape as one entry
// of `starcm inspect`'s JSON. path must be absolute; a leading ~ is
// expanded.
//
//	r = inspect(here("bootstrap.star"))
//	r["loads"], r["calls"], r["values"]
func New(ctx context.Context, fsys afero.Fs) *base.Query {
	return base.NewQuery(ctx, "inspect", []base.ArgPair{base.Arg[string]("path")}, func(inst *base.FunctionInstance) (starlark.Value, error) {
		p, _, err := inst.AbsPath("path")
		if err != nil {
			return nil, err
		}
		src, err := afero.ReadFile(fsys, p)
		if err != nil {
			return nil, inst.Errorf("%w", err)
		}
		report, err := inspect.File(p, src)
		if err != nil {
			return nil, inst.Errorf("%w", err)
		}
		// Round-trip through JSON so the value matches the CLI output exactly.
		b, err := encodingjson.Marshal(report)
		if err != nil {
			return nil, err
		}
		return starlark.Call(inst.Thread, json.Module.Members["decode"], starlark.Tuple{starlark.String(b)}, nil)
	})
}
