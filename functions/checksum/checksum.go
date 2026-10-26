package checksum

import (
	"context"

	"github.com/discentem/starcm/functions/base"
	starcmsha256 "github.com/discentem/starcm/libraries/sha256"
	"github.com/spf13/afero"
	"go.starlark.net/starlark"
)

// New returns sha256(), a query that returns the lowercase hex sha256 of
// the file at path. It uses the same implementation download() does, so
// the value can be passed straight to download(sha256 = ...). path must be
// absolute; a leading ~ is expanded.
func New(ctx context.Context, fsys afero.Fs) *base.Query {
	return base.NewQuery(ctx, "sha256", []base.ArgPair{base.Arg[string]("path")}, func(inst *base.FunctionInstance) (starlark.Value, error) {
		p, _, err := inst.AbsPath("path")
		if err != nil {
			return nil, err
		}
		sum, err := starcmsha256.File(fsys, p)
		if err != nil {
			return nil, inst.Errorf("%w", err)
		}
		return starlark.String(sum), nil
	})
}
