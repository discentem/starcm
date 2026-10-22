package checksum

import (
	"context"

	"github.com/discentem/starcm/functions/base"
	starcmsha256 "github.com/discentem/starcm/libraries/sha256"
	"github.com/spf13/afero"
	"go.starlark.net/starlark"
)

type sha256Action struct {
	fsys afero.Fs
}

var _ base.Action = (*sha256Action)(nil)

// Execute hashes the file at path with the same implementation download()
// uses, so the value can be passed straight to download(sha256 = ...). path
// must be absolute; a leading ~ is expanded.
func (a *sha256Action) Execute(inst *base.FunctionInstance) (*base.Result, error) {
	p, _, err := inst.AbsPath("path")
	if err != nil {
		return nil, err
	}
	sum, err := starcmsha256.File(a.fsys, p)
	if err != nil {
		return nil, err
	}
	return &base.Result{Label: inst.Label, Success: true, Return: starlark.String(sum)}, nil
}

// New returns the sha256() builtin: sha256("file", label = "...").value is
// the file's lowercase hex sha256.
func New(ctx context.Context, fsys afero.Fs) *base.Module {
	return base.NewModule(ctx, "sha256", []base.ArgPair{base.Arg[string]("path")}, &sha256Action{fsys: fsys})
}
