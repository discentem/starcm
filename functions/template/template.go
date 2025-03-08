package template

import (
	"context"
	"os"
	"path/filepath"

	"fmt"

	"github.com/discentem/starcm/functions/base"
	starcmfileutils "github.com/discentem/starcm/libraries/fileutils"
	"github.com/discentem/starcm/libraries/logging"
	starlarkhelpers "github.com/discentem/starcm/starlark-helpers"
	"github.com/google/deck"

	// TODO (discentem): consider replacing with a different template engine
	"github.com/noirbizarre/gonja"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/spf13/afero"
	"go.starlark.net/starlark"
)

type templateAction struct {
	fsys afero.Fs
}

func (a *templateAction) writeTemplate(path string, data []byte) error {
	// Ensure parent directories exist
	dir := filepath.Dir(path)
	if err := a.fsys.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory structure for template: %w", err)
	}

	// Create or truncate the file
	f, err := a.fsys.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file for writing template: %w", err)
	}
	defer f.Close()

	// Write data to the file
	n, err := f.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write template data: %w", err)
	}

	// Check if all data was written
	if n != len(data) {
		return fmt.Errorf("incomplete write: wrote %d bytes out of %d", n, len(data))
	}

	// Ensure data is flushed to disk
	if err := f.Sync(); err != nil {
		return fmt.Errorf("failed to sync template data to disk: %w", err)
	}

	return nil
}

type parsedArgs struct {
	templatePath string
	data         map[string]any
	destination  string
}

func (a *templateAction) parseArgs(_ starlark.Tuple, kwargs []starlark.Tuple) (*parsedArgs, error) {
	template, err := starlarkhelpers.FindValueinKwargs(kwargs, "template")
	if err != nil {
		return nil, err
	}
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

	destination, err := starlarkhelpers.FindValueinKwargs(kwargs, "destination")
	if err != nil {
		return nil, err
	}
	if destination == nil {
		return nil, fmt.Errorf("destination is required in template() module")
	}

	return &parsedArgs{
		templatePath: *template,
		data:         gokv,
		destination:  *destination,
	}, nil
}

func (a *templateAction) Run(ctx context.Context, workingDirectory string, moduleName string, args starlark.Tuple, kwargs []starlark.Tuple) (*base.Result, error) {
	parsedArgs, err := a.parseArgs(args, kwargs)
	if err != nil {
		return nil, err
	}
	template := parsedArgs.templatePath
	gokv := parsedArgs.data
	destination := parsedArgs.destination

	isDir, err := starcmfileutils.IsDir(a.fsys, destination)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if isDir {
		return nil, fmt.Errorf("destination must be a file, not a directory")
	}

	f, err := a.fsys.Open(filepath.Join(workingDirectory, template))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := afero.ReadAll(f)
	if err != nil {
		return nil, err
	}
	logging.Log(moduleName, deck.V(2), "info", "%v before rendering: %v", template, string(b))
	logging.Log(moduleName, deck.V(2), "info", "data: %v", gokv)
	tmpl, err := gonja.FromBytes(b)
	if err != nil {
		// If it fails here, it's likely a problem with the .tmpl file itself such as unexpected symbols
		logging.Log("template", deck.V(1), "error", "failed to parse template", err)
		return nil, err
	}
	renderedTemplate, err := tmpl.Execute(gokv)
	if err != nil {
		logging.Log("template", deck.V(1), "error", "failed to render template", err)
		return nil, err
	}

	destinationPath := filepath.Join(workingDirectory, destination)

	// Check if destination file exists
	destinationExists := true
	info, err := a.fsys.Stat(destinationPath)
	if err != nil {
		if os.IsNotExist(err) {
			logging.Log(moduleName, deck.V(2), "info", "destination file does not exist, will create it")
			destinationExists = false
		} else {
			// If error is not "file doesn't exist", return the error
			return nil, err
		}
	} else {
		logging.Log(moduleName, deck.V(2), "info", "a.fsys.Stat(%s): %v", destinationPath, info.Name())
	}

	// Handle case where file doesn't exist
	if !destinationExists {
		// Create file with rendered template
		if err := a.writeTemplate(destinationPath, []byte(renderedTemplate)); err != nil {
			return &base.Result{
				Name:    &moduleName,
				Output:  nil,
				Success: false,
				Changed: false,
				Error:   err,
			}, err
		}

		// Return success after creating new file
		return &base.Result{
			Name:    &moduleName,
			Output:  &renderedTemplate,
			Success: true,
			Changed: true,
			Error:   nil,
			Diff:    &renderedTemplate,
		}, nil
	}

	// At this point, we know the file exists, so read its contents
	destinationBefore, err := afero.ReadFile(a.fsys, destinationPath)
	if err != nil {
		return nil, err
	}

	// If the file exists and the contents are the same, return a success
	if string(destinationBefore) == renderedTemplate {
		emptyString := ""
		renderedStr := fmt.Sprint(renderedTemplate)
		return &base.Result{
			Name:    &moduleName,
			Output:  &renderedStr,
			Success: true,
			Changed: false,
			Error:   nil,
			Diff:    &emptyString,
		}, nil
	}

	if err := a.writeTemplate(destinationPath, []byte(renderedTemplate)); err != nil {
		return &base.Result{
			Name:    &moduleName,
			Output:  nil,
			Success: false,
			Changed: false,
			Error:   err,
		}, err
	}

	diff := diffmatchpatch.New().DiffMain(string(destinationBefore), renderedTemplate, false)
	logging.Log(moduleName, deck.V(2), "info", "diff: %v", diff)

	renderedStr := fmt.Sprint(renderedTemplate)
	return &base.Result{
		Name:    &moduleName,
		Output:  &renderedStr,
		Success: true,
		Changed: true,
		Error:   nil,
		Diff:    &diff[0].Text,
	}, nil
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
			{Key: "destination", Type: &str},
		},
		&templateAction{
			fsys: fsys,
		},
	)
}
