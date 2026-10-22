package template

import (
	"context"
	"os"
	"path/filepath"

	"fmt"

	"github.com/discentem/starcm/functions/base"
	"github.com/discentem/starcm/libraries/diffutils"
	starcmfileutils "github.com/discentem/starcm/libraries/fileutils"
	"github.com/discentem/starcm/libraries/logging"
	starlarkhelpers "github.com/discentem/starcm/starlark-helpers"
	"github.com/google/deck"

	// TODO (discentem): consider replacing with a different template engine
	"github.com/noirbizarre/gonja"
	gonjaconfig "github.com/noirbizarre/gonja/config"
	"github.com/spf13/afero"
	"go.starlark.net/starlark"
)

// renderEnv is gonja's default environment, except it keeps a template's
// trailing newline (Jinja2 drops it by default) so rendered files end the
// way their template does.
var renderEnv = func() *gonja.Environment {
	cfg := gonjaconfig.NewConfig()
	cfg.KeepTrailingNewline = true
	return gonja.NewEnvironment(cfg, gonja.DefaultLoader)
}()

type templateAction struct {
	fsys afero.Fs
}

type writeTemplateOptions struct {
	persist bool
}

func (a *templateAction) writeTemplate(path string, data []byte, opts writeTemplateOptions) error {
	if !opts.persist {
		logging.Log("template", deck.V(2), "info", "skipping write to disk because persist is false")
		return nil
	}

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
	// templatePath is the template file, or "" when source is given.
	templatePath string
	// source is the template text itself, when given instead of a file.
	source      *string
	data        map[string]any
	destination string
	whatIf      bool
}

// parseArgs reads the call's arguments. A relative template path is a
// bundled resource: it resolves next to the .star file the call is written
// in. destination is an output, so it must be absolute.
func (a *templateAction) parseArgs(inst *base.FunctionInstance) (*parsedArgs, error) {
	template, hasTemplate, err := inst.ResourcePath("template")
	if err != nil {
		return nil, err
	}
	sourceText, hasSource := base.Get[string](inst, "source")
	var source *string
	switch {
	case hasTemplate && hasSource:
		return nil, fmt.Errorf("template() takes template or source, not both")
	case !hasTemplate && !hasSource:
		return nil, fmt.Errorf("template() needs template (a template file) or source (the template text)")
	case hasSource:
		source = &sourceText
	}

	keyVals, _ := base.Get[*starlark.Dict](inst, "data")
	gokv, err := starlarkhelpers.DictToGoMap(keyVals)
	if err != nil {
		return nil, fmt.Errorf("template() data: %w", err)
	}

	destination, hasDestination, err := inst.AbsPath("destination")
	if err != nil {
		return nil, err
	}
	if !hasDestination && !inst.WhatIf {
		return nil, fmt.Errorf("destination is required in template() module if what_if is false")
	}

	return &parsedArgs{
		templatePath: template,
		source:       source,
		data:         gokv,
		destination:  destination,
		whatIf:       inst.WhatIf,
	}, nil
}

var _ base.Action = (*templateAction)(nil)

func (a *templateAction) Execute(inst *base.FunctionInstance) (*base.Result, error) {
	moduleName := inst.Label
	parsedArgs, err := a.parseArgs(inst)
	if err != nil {
		return nil, err
	}
	template := parsedArgs.templatePath
	gokv := parsedArgs.data
	destination := parsedArgs.destination
	whatIf := parsedArgs.whatIf

	isDir, err := starcmfileutils.IsDir(a.fsys, destination)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if isDir {
		return nil, fmt.Errorf("destination must be a file, not a directory")
	}

	var b []byte
	if parsedArgs.source != nil {
		b = []byte(*parsedArgs.source)
		template = "source"
	} else {
		f, err := a.fsys.Open(template)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		if b, err = afero.ReadAll(f); err != nil {
			return nil, err
		}
	}
	logging.Log(moduleName, deck.V(2), "info", "%v before rendering: %v", template, string(b))
	logging.Log(moduleName, deck.V(2), "info", "data: %v", gokv)
	tmpl, err := renderEnv.FromBytes(b)
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

	destinationPath := destination

	// Check if destination file exists
	destinationExists := true
	info, err := a.fsys.Stat(destinationPath)
	if err != nil {
		if os.IsNotExist(err) {
			if !whatIf {
				logging.Log(moduleName, deck.V(2), "info", "destination file does not exist, will create it")
			}
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
		if err := a.writeTemplate(
			destinationPath,
			[]byte(renderedTemplate),
			writeTemplateOptions{
				persist: !whatIf,
			},
		); err != nil {
			return &base.Result{
				Label:   moduleName,
				Message: nil,
				Success: false,
				Changed: false,
				Error:   err,
			}, err
		}

		// Return success after creating new file
		return &base.Result{
			Label:   moduleName,
			Message: &renderedTemplate,
			Success: true,
			Changed: true,
			Error:   nil,
		}, nil
	}

	// At this point, we know the file exists, so read its contents
	destinationBefore, err := afero.ReadFile(a.fsys, destinationPath)
	if err != nil {
		return nil, err
	}

	// If the file exists and the contents are the same, return a success
	if string(destinationBefore) == renderedTemplate {
		renderedStr := fmt.Sprint(renderedTemplate)
		return &base.Result{
			Label:   moduleName,
			Message: &renderedStr,
			Success: true,
			Changed: false,
			Error:   nil,
		}, nil
	}

	if err := a.writeTemplate(
		destinationPath,
		[]byte(renderedTemplate),
		writeTemplateOptions{
			persist: !whatIf,
		},
	); err != nil {
		return &base.Result{
			Label:   moduleName,
			Message: nil,
			Success: false,
			Changed: false,
			Error:   err,
		}, err
	}

	diff := diffutils.GitDiff(string(destinationBefore), renderedTemplate)
	logging.Log(moduleName, deck.V(2), "info", "diff: %v", diff)

	renderedStr := fmt.Sprint(renderedTemplate)

	return &base.Result{
		Label:   moduleName,
		Message: &renderedStr,
		Success: true,
		Changed: true,
		Error:   nil,
	}, nil
}

// templateArgs are template()'s own arguments.
var templateArgs = []base.ArgPair{
	base.OptionalArg[string]("template"),
	base.OptionalArg[string]("source"),
	base.Arg[*starlark.Dict]("data"),
	base.OptionalArg[string]("destination"),
}

func New(ctx context.Context, fsys afero.Fs) *base.Module {
	return base.NewModule(ctx, "template", templateArgs, &templateAction{fsys: fsys})
}
