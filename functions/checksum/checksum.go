package checksum

import (
	"context"

	"github.com/discentem/starcm/functions/base"
	starcmsha256 "github.com/discentem/starcm/libraries/sha256"
	starlarkhelpers "github.com/discentem/starcm/starlark-helpers"
	"github.com/spf13/afero"
	"go.starlark.net/starlark"
)

type sha256Action struct {
	fsys afero.Fs
}

var _ base.Runnable = (*sha256Action)(nil)

// Run hashes the file at path with the same implementation download()
// uses, so the value can be passed straight to download(sha256 = ...).
func (a *sha256Action) Run(ctx context.Context, workingDirectory string, moduleName string, thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (*base.Result, error) {
	var p string
	if args.Len() == 1 {
		p = args.Index(0).(starlark.String).GoString()
	} else {
		v, err := starlarkhelpers.FindValueinKwargs(kwargs, "path")
		if err != nil {
			return nil, err
		}
		p = *v
	}
	sum, err := starcmsha256.File(a.fsys, p)
	if err != nil {
		return nil, err
	}
	return &base.Result{Label: moduleName, Success: true, Return: starlark.String(sum)}, nil
}

// New returns the sha256() builtin: sha256("file", label = "...").value is
// the file's lowercase hex sha256.
func New(ctx context.Context, fsys afero.Fs) *base.Module {
	return base.NewModule(ctx, "sha256", []base.ArgPair{base.Arg[string]("path")}, &sha256Action{fsys: fsys})
}
