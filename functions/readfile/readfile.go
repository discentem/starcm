package readfile

import (
	"context"

	"github.com/discentem/starcm/functions/base"
	"github.com/spf13/afero"
	"go.starlark.net/starlark"
)

// New returns read_file(), a query that returns the content of the file at
// path as a string. path must be absolute; a leading ~ is expanded.
//
//	read_file("/etc/hosts")
func New(ctx context.Context, fsys afero.Fs) *base.Query {
	return base.NewQuery(ctx, "read_file", []base.ArgPair{base.Arg[string]("path")}, func(inst *base.FunctionInstance) (starlark.Value, error) {
		p, _, err := inst.AbsPath("path")
		if err != nil {
			return nil, err
		}
		data, err := afero.ReadFile(fsys, p)
		if err != nil {
			return nil, inst.Errorf("%w", err)
		}
		return starlark.String(data), nil
	})
}
