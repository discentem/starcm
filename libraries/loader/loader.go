package loading

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/discentem/starcm/libraries/logging"
	starlarkhelpers "github.com/discentem/starcm/starlark-helpers"
	"github.com/google/deck"
	"github.com/spf13/afero"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	"go.starlark.net/syntax"

	starcmdownload "github.com/discentem/starcm/functions/download"
	dynamicloading "github.com/discentem/starcm/functions/dynamic_loading"
	starcmshard "github.com/discentem/starcm/functions/shard"
	starcmshell "github.com/discentem/starcm/functions/shell"
	starcmtemplate "github.com/discentem/starcm/functions/template"
	starcmwrite "github.com/discentem/starcm/functions/write"
	starcmshelllib "github.com/discentem/starcm/libraries/shell"
)

func LoadFromFile(ctx context.Context, fpath string, src interface{}, load starlarkhelpers.LoaderFunc) error {
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

// Loader handles module loading for starlark modules.
type Loader struct {
	// Predeclared is used for builtin modules which are not loaded from a path.
	Predeclared func(module string) (starlark.StringDict, error)

	// WorkspacePath specifies the path to the source directory.
	WorkspacePath string

	// Fsys is the filesystem to use for loading modules.
	Fsys afero.Fs
}

// Sequential implements sequential module loading.
// Module paths starting with "//" will be loaded from WorkspacePath, which should be the mount path to the workspace source directory.
// Absolute paths and relative paths (from the caller's location) are also supported.
func (l *Loader) Sequential(ctx context.Context) func(thread *starlark.Thread, module string) (starlark.StringDict, error) {
	type entry struct {
		globals starlark.StringDict
		err     error
	}

	var cache = make(map[string]*entry)

	var load func(_ *starlark.Thread, module string) (starlark.StringDict, error)
	load = func(thread *starlark.Thread, module string) (starlark.StringDict, error) {
		if e, ok := cache[module]; ok {
			if e == nil {
				return nil, fmt.Errorf("cycle in load graph %q", module)
			}
			return e.globals, e.err
		}

		cache[module] = nil // mark as loading

		// Try resolving as a built-in module first
		if builtin, err := l.Predeclared(module); builtin != nil || err != nil {
			e := &entry{builtin, err}
			cache[module] = e
			return e.globals, e.err
		}

		if path.Ext(module) != ".star" {
			return nil, fmt.Errorf("module %q is not valid, modules should have a .star extension", module)
		}

		modulePath := l.resolveModulePath(thread, module)

		globals, err := l.execModule("exec "+module, modulePath, load)
		e := &entry{globals, err}
		cache[module] = e
		return globals, err
	}

	return load
}

// resolveModulePath determines the actual filesystem path to the module based on workspace, call stack, or absolute logic.
func (l *Loader) resolveModulePath(thread *starlark.Thread, module string) string {
	switch {
	case filepath.IsAbs(module):
		return module

	case strings.HasPrefix(module, "//"):
		// Workspace-relative
		return path.Join(l.WorkspacePath, module[2:]) // strip leading //

	case len(thread.CallStack()) > 0:
		// Relative to the caller module
		caller := thread.CallStack().At(0)
		callerDir := filepath.Dir(caller.Pos.Filename())
		return filepath.Join(callerDir, module)

	default:
		// Default fallback: relative to workspace
		return filepath.Join(l.WorkspacePath, module)
	}
}

// execModule reads, parses, and executes a Starlark module file.
func (l *Loader) execModule(threadName, modulePath string, loadFunc func(*starlark.Thread, string) (starlark.StringDict, error)) (starlark.StringDict, error) {
	data, err := afero.ReadFile(l.Fsys, modulePath)
	if err != nil {
		return nil, fmt.Errorf("loading module %q: %s", modulePath, err)
	}

	thread := &starlark.Thread{
		Name: threadName,
		Load: loadFunc,
	}

	globals, err := starlark.ExecFileOptions(
		&syntax.FileOptions{TopLevelControl: true},
		thread,
		modulePath,
		data,
		nil,
	)

	if err != nil {
		switch e := err.(type) {
		case *starlark.EvalError:
			return nil, fmt.Errorf("load failed:\n%s", e.Backtrace())
		case *syntax.Error:
			return nil, fmt.Errorf("syntax error:\n%s", e.Error())
		default:
			return nil, fmt.Errorf("unexpected load error: %w", err)
		}
	}

	return globals, nil
}

type LoaderOption func(*Loader)

func WithWorkspacePath(workspacePath string) LoaderOption {
	return func(l *Loader) {
		l.WorkspacePath = workspacePath
	}
}

func WithPredeclared(predeclared func(module string) (starlark.StringDict, error)) LoaderOption {
	return func(l *Loader) {
		l.Predeclared = predeclared
	}
}

func WithFsys(fsys afero.Fs) LoaderOption {
	return func(l *Loader) {
		l.Fsys = fsys
	}
}

func NewLoader(ctx context.Context, opts ...LoaderOption) Loader {
	l := Loader{}
	for _, opt := range opts {
		opt(&l)
	}
	return l
}

func Default(ctx context.Context, fsys afero.Fs, ex starcmshelllib.Executor, workspacePath string) Loader {
	l := NewLoader(
		ctx,
		WithWorkspacePath(workspacePath),
		WithFsys(fsys),
		WithPredeclared(func(module string) (starlark.StringDict, error) {
			switch module {
			case "starcm":
				return starlark.StringDict{
					"download": starlark.NewBuiltin(
						"download",
						starcmdownload.New(
							ctx,
							*http.DefaultClient,
							fsys,
							os.Stdout,
						).Function(),
					),
					"write": starlark.NewBuiltin(
						"write",
						starcmwrite.New(ctx, os.Stdout).Function(),
					),
					"exec": starlark.NewBuiltin(
						"exec",
						starcmshell.New(
							ctx,
							ex,
						).Function(),
					),
					"template": starlark.NewBuiltin(
						"template",
						starcmtemplate.New(ctx, fsys).Function(),
					),
					"shard": starlark.NewBuiltin(
						"shard",
						starcmshard.New(ctx).Function(),
					),
					"load_dynamic": starlark.NewBuiltin("load_dynamic", dynamicloading.Load()),
				}, nil
			case "starlarkstdlib":
				return starlark.StringDict{
					"struct": starlark.NewBuiltin("struct", starlarkstruct.Make),
				}, nil
			default:
				// set both to nil to allow the loader to load a .star file from a path.
				return nil, nil
			}
		}),
	)
	return l
}
