package listfiles

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"sort"

	"github.com/discentem/starcm/functions/base"
	"github.com/spf13/afero"
	"go.starlark.net/starlark"
)

type listFilesAction struct {
	fsys afero.Fs
}

var _ base.Action = (*listFilesAction)(nil)

// Execute lists the files under dir, recursively, whose base name matches
// pattern. value is a sorted list of paths relative to dir, with forward
// slashes on every platform. dir must be absolute; a leading ~ is expanded.
func (a *listFilesAction) Execute(inst *base.FunctionInstance) (*base.Result, error) {
	dir, _, err := inst.AbsPath("dir")
	if err != nil {
		return nil, err
	}
	pattern, given := base.Get[string](inst, "pattern")
	if !given {
		pattern = "*"
	}
	if _, err := path.Match(pattern, ""); err != nil {
		return nil, fmt.Errorf("invalid pattern %q: %w", pattern, err)
	}

	var files []string
	err = afero.Walk(a.fsys, dir, func(p string, info fs.FileInfo, err error) error {
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
		return nil, err
	}
	sort.Strings(files)

	list := make([]starlark.Value, len(files))
	for i, f := range files {
		list[i] = starlark.String(f)
	}
	return &base.Result{Label: inst.Label, Success: true, Return: starlark.NewList(list)}, nil
}

// New returns the list_files() builtin.
//
//	stars = list_files(here("channel"), pattern = "*.star", label = "find config").value
func New(ctx context.Context, fsys afero.Fs) *base.Module {
	return base.NewModule(
		ctx,
		"list_files",
		[]base.ArgPair{
			base.Arg[string]("dir"),
			base.OptionalArg[string]("pattern"),
		},
		&listFilesAction{fsys: fsys},
	)
}
