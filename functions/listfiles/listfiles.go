package listfiles

import (
	"context"
	"io/fs"
	"path"
	"path/filepath"
	"sort"

	"github.com/discentem/starcm/functions/base"
	"github.com/spf13/afero"
	"go.starlark.net/starlark"
)

// New returns path.list(), which lists the files under dir, recursively,
// whose base name matches pattern. It returns a sorted list of paths
// relative to dir, with forward slashes on every platform. dir must be
// absolute; a leading ~ is expanded.
//
//	stars = path.list(here("channel"), pattern = "*.star")
func New(ctx context.Context, fsys afero.Fs) *base.Query {
	return base.NewQuery(
		ctx,
		"path.list",
		[]base.ArgPair{
			base.Arg[string]("dir"),
			base.OptionalArg[string]("pattern"),
		},
		func(inst *base.FunctionInstance) (starlark.Value, error) {
			return list(fsys, inst)
		},
	)
}

func list(fsys afero.Fs, inst *base.FunctionInstance) (starlark.Value, error) {
	dir, _, err := inst.AbsPath("dir")
	if err != nil {
		return nil, err
	}
	pattern, given := base.Get[string](inst, "pattern")
	if !given {
		pattern = "*"
	}
	if _, err := path.Match(pattern, ""); err != nil {
		return nil, inst.Errorf("invalid pattern %q: %w", pattern, err)
	}
	var files []string
	err = afero.Walk(fsys, dir, func(p string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if ok, _ := path.Match(pattern, info.Name()); !ok {
			return nil
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, inst.Errorf("%w", err)
	}
	sort.Strings(files)
	out := make([]starlark.Value, len(files))
	for i, f := range files {
		out[i] = starlark.String(f)
	}
	return starlark.NewList(out), nil
}
