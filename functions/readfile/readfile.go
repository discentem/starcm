package readfile

import (
	"context"
	"fmt"

	"github.com/discentem/starcm/functions/base"
	"github.com/spf13/afero"
	"go.starlark.net/starlark"
)

type readFileAction struct {
	fsys afero.Fs
}

var _ base.Action = (*readFileAction)(nil)

// Execute returns the file's content as the result's value. path must be
// absolute; a leading ~ is expanded.
func (a *readFileAction) Execute(inst *base.FunctionInstance) (*base.Result, error) {
	p, _, err := inst.AbsPath("path")
	if err != nil {
		return nil, err
	}
	data, err := afero.ReadFile(a.fsys, p)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %q: %w", p, err)
	}
	return &base.Result{Label: inst.Label, Success: true, Return: starlark.String(data)}, nil
}

// New returns the read_file() builtin: read_file("/etc/hosts", label = "...").value
func New(ctx context.Context, fsys afero.Fs) *base.Module {
	return base.NewModule(ctx, "read_file", []base.ArgPair{base.Arg[string]("path")}, &readFileAction{fsys: fsys})
}
