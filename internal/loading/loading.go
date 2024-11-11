package loading

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"errors"

	starcmdownload "github.com/discentem/starcm/functions/download"
	starcmshard "github.com/discentem/starcm/functions/shard"
	starcmshell "github.com/discentem/starcm/functions/shell"
	starcmtemplate "github.com/discentem/starcm/functions/template"
	starcmwrite "github.com/discentem/starcm/functions/write"
	"github.com/discentem/starcm/libraries/logging"
	"github.com/discentem/starcm/libraries/shell"
	starlarkhelpers "github.com/discentem/starcm/starlark-helpers"
	"github.com/google/deck"
	"github.com/spf13/afero"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	"go.starlark.net/syntax"
)

// Loader handles module loading for starlark modules.
type Loader struct {
	// Predeclared is used for builtin modules which are not loaded from a path.
	Predeclared func(module string) (starlark.StringDict, error)

	// WorkspacePath specifies the path to the source directory.
	WorkspacePath string
}

func DefaultLoader(ctx context.Context, fsys afero.Fs, executor shell.Executor, workspacePath string) Loader {
	loader := Loader{
		WorkspacePath: workspacePath,
		Predeclared: func(module string) (starlark.StringDict, error) {
			switch module {
			case "starcm":
				return starlark.StringDict{
					"download": starlark.NewBuiltin(
						"download",
						starcmdownload.New(
							ctx,
							*http.DefaultClient,
							fsys,
						).Function(),
					),
					"write": starlark.NewBuiltin(
						"write",
						starcmwrite.New(ctx, os.Stdout).Function(),
					),
					"exec": starlark.NewBuiltin(
						"exec",
						starcmshell.New(ctx, executor).Function(),
					),
					"template": starlark.NewBuiltin(
						"template",
						starcmtemplate.New(ctx, fsys).Function(),
					),
					"shard": starlark.NewBuiltin(
						"shard",
						starcmshard.New(ctx).Function(),
					),
					"load_dynamic": starlark.NewBuiltin("load_dynamic", DynamicLoadfunc()),
				}, nil
			case "starlarkstdlib":
				return starlark.StringDict{
					"struct": starlark.NewBuiltin("struct", starlarkstruct.Make),
				}, nil
			default:
				// set both to nil to allow the loader to load a .star file from a path.
				return nil, nil
			}
		},
	}
	return loader
}

func FromFile(ctx context.Context, fpath string, src interface{}, load starlarkhelpers.LoaderFunc) error {
	logging.Log("LoadFromFile", deck.V(2), "info", "loading file %q", fpath)
	thread := &starlark.Thread{
		Load:  load,
		Name:  "starcm",
		Print: func(_ *starlark.Thread, msg string) { fmt.Println(msg) },
	}

	var currentDir string
	if len(thread.CallStack()) > 0 {
		currentDir = filepath.Dir(thread.CallStack().At(0).Pos.Filename())
	} else {
		// Fallback if there are no call frames, assuming the initial script directory
		currentDir = fpath
	}

	logging.Log("LoadFromFile", deck.V(3), "info", "current starlark execution dir %q", currentDir)
	if currentDir != fpath {
		fpath = filepath.Join(currentDir, fpath)
	}

	if _, err := starlark.ExecFileOptions(
		&syntax.FileOptions{
			// Allow if statements and for loops to be top-level in the module.
			TopLevelControl: true,
		},
		thread,
		fpath,
		src, nil,
	); err != nil {
		if evalErr, ok := err.(*starlark.EvalError); ok {
			return errors.New(evalErr.Backtrace())
		}
		return fmt.Errorf("load at path: %q: %s", fpath, err)
	}
	return nil
}
