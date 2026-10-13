package readfile

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/discentem/starcm/functions/base"
	starlarkhelpers "github.com/discentem/starcm/starlark-helpers"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/afero"
	"go.starlark.net/starlark"
)

type readFileAction struct {
	fsys afero.Fs
}

var _ base.Runnable = (*readFileAction)(nil)

// Run returns the file's content as the result's value. Like file(), path
// may start with ~ and is otherwise relative to the current directory.
func (a *readFileAction) Run(ctx context.Context, workingDirectory string, moduleName string, thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (*base.Result, error) {
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
	resolved, err := homedir.Expand(p)
	if err != nil {
		return nil, fmt.Errorf("failed to expand home directory in path %q: %w", p, err)
	}
	if !filepath.IsAbs(resolved) {
		if resolved, err = filepath.Abs(resolved); err != nil {
			return nil, fmt.Errorf("failed to resolve path %q: %w", p, err)
		}
	}
	data, err := afero.ReadFile(a.fsys, resolved)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %q: %w", p, err)
	}
	return &base.Result{Label: moduleName, Success: true, Return: starlark.String(data)}, nil
}

// New returns the read_file() builtin: read_file("notes.txt", label = "...").value
func New(ctx context.Context, fsys afero.Fs) *base.Module {
	return base.NewModule(ctx, "read_file", []base.ArgPair{base.Arg[string]("path")}, &readFileAction{fsys: fsys})
}
