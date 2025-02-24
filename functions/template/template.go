package template

import (
	"context"
	"path/filepath"

	"fmt"

	"github.com/discentem/starcm/functions/base"
	"github.com/discentem/starcm/libraries/logging"
	starlarkhelpers "github.com/discentem/starcm/starlark-helpers"
	"github.com/google/deck"

	// TODO (discentem): consider replacing with a different template engine
	"github.com/noirbizarre/gonja"
	"github.com/spf13/afero"
	"go.starlark.net/starlark"
)

type templateAction struct {
	fsys afero.Fs
}

func (a *templateAction) Run(ctx context.Context, workingDirectory string, moduleName string, args starlark.Tuple, kwargs []starlark.Tuple) (base.ActionReturn, error) {
	template, err := starlarkhelpers.FindValueinKwargs(kwargs, "template")
	if err != nil {
		return nil, err
	}

	// TODO (discentem): add argument for output file

	if template == nil {
		return nil, fmt.Errorf("template is required in template() module")
	}
	keyWordStr := "data"

	keyValsIdx, err := starlarkhelpers.FindIndexOfValueInKwargs(kwargs, keyWordStr)
	if err != nil {
		logging.Log("template", deck.V(3), "error", "failed to find index of %s in kwargs: %v", keyWordStr, err)
		return nil, err
	}
	if keyValsIdx == starlarkhelpers.IndexNotFound {
		return nil, fmt.Errorf("%s is required in template() module", keyWordStr)
	}
	keyVals := kwargs[keyValsIdx][1].(*starlark.Dict)
	gokv := starlarkhelpers.DictToGoMap(keyVals)

	f, err := a.fsys.Open(filepath.Join(workingDirectory, *template))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := afero.ReadAll(f)
	if err != nil {
		return nil, err
	}
	logging.Log(moduleName, deck.V(2), "info", "%v before rendering: %v", *template, string(b))
	logging.Log(moduleName, deck.V(2), "info", "data: %v", gokv)
	tmpl, err := gonja.FromBytes(b)
	if err != nil {
		// If it fails here, it's likely a problem with the .tmpl file itself such as unexpected symbols
		logging.Log("template", deck.V(3), "error", "failed to parse template", err)
		return nil, err
	}
	renderedTemplate, err := tmpl.Execute(gokv)
	if err != nil {
		logging.Log("template", deck.V(3), "error", "failed to render template", err)
		return nil, err
	}

	return &base.Result{
		Name: &moduleName,
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
		str  string
		data *starlark.Dict
	)

	return base.NewModule(
		ctx,
		"template",
		[]base.ArgPair{
			{Key: "template", Type: &str},
			{Key: "data", Type: &data},
		},
		&templateAction{
			fsys: fsys,
		},
	)
}
