package template

import (
	"context"
	"path/filepath"

	"fmt"

	"github.com/discentem/starcm/functions/base"
	starlarkhelpers "github.com/discentem/starcm/starlark-helpers"
	"github.com/noirbizarre/gonja"
	"github.com/spf13/afero"
	"go.starlark.net/starlark"
	// TODO (discentem): consider replacing with a different template engine
)

type action struct {
	fsys afero.Fs
}

func (a *action) Run(ctx context.Context, moduleName string, args starlark.Tuple, kwargs []starlark.Tuple) (*base.Result, error) {
	template, err := starlarkhelpers.FindValueinKwargs(kwargs, "template")
	if err != nil {
		return nil, err
	}
	if template == nil {
		return nil, fmt.Errorf("template is required in template() module")
	}

	if !absolutePath {
		if len(thread.CallStack()) > 0 {
			dirName := filepath.Dir(thread.CallStack().At(1).Pos.Filename())
			modulepath = filepath.Join(dirName, modulepath)
		}
	}

	keyValsIdx, err := starlarkhelpers.FindIndexOfValueInKwargs(kwargs, "key_vals")
	if err != nil {
		return nil, err
	}
	if keyValsIdx == starlarkhelpers.IndexNotFound {
		return nil, fmt.Errorf("key_vals is required in template() module")
	}
	keyVals := kwargs[keyValsIdx][1].(*starlark.Dict)
	gokv := starlarkhelpers.DictToGoMap(keyVals)

	f, err := a.fsys.Open(*template)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := afero.ReadAll(f)
	if err != nil {
		return nil, err
	}
	tmpl, err := gonja.FromBytes(b)
	if err != nil {
		return nil, err
	}
	renderedTemplate, err := tmpl.Execute(gokv)

	return &base.Result{
		Output: func() *string {
			s := fmt.Sprint(renderedTemplate)
			return &s
		}(),
		Success: true,
		Changed: false,
		Error:   err,
	}, err
}

func New(ctx context.Context, fsys afero.Fs) *base.Module {
	var (
		str     string
		keyVals *starlark.Dict
	)

	return base.NewModule(
		ctx,
		"write",
		[]base.ArgPair{
			{Key: "template", Type: &str},
			{Key: "key_vals", Type: &keyVals},
		},
		&action{
			fsys: fsys,
		},
	)
}