package loading

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	localstore "github.com/discentem/starcm/libraries/local_store"
	"github.com/discentem/starcm/libraries/logging"
	"github.com/discentem/starcm/libraries/signing"
	starlarkhelpers "github.com/discentem/starcm/starlark-helpers"
	"github.com/google/deck"
	"github.com/spf13/afero"
	starlarkjson "go.starlark.net/lib/json"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	"go.starlark.net/syntax"

	starcmchecksum "github.com/discentem/starcm/functions/checksum"
	starcmdate "github.com/discentem/starcm/functions/date"
	starcmdownload "github.com/discentem/starcm/functions/download"
	dynamicloading "github.com/discentem/starcm/functions/dynamic_loading"
	starcmFile "github.com/discentem/starcm/functions/file"
	starcminspect "github.com/discentem/starcm/functions/inspect"
	starcmlistfiles "github.com/discentem/starcm/functions/listfiles"
	starcmmacpkg "github.com/discentem/starcm/functions/macpkg"
	starcmreadfile "github.com/discentem/starcm/functions/readfile"
	starcmshell "github.com/discentem/starcm/functions/shell"
	starcmsign "github.com/discentem/starcm/functions/sign"
	starcmtemplate "github.com/discentem/starcm/functions/template"
	starcmverify "github.com/discentem/starcm/functions/verify"
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

	// Keyring, when set, requires every module to have a valid signature
	// file next to it before it runs.
	Keyring *signing.Keyring

	// Vars are the --var values, exposed to Starlark as the "vars" dict.
	Vars map[string]string

	// Store opens the local store that download() saves into when save_to
	// is omitted. Nil means there's no store and save_to is required.
	Store func() (*localstore.Store, error)

	// StoreOnly makes download() refuse save_to.
	StoreOnly bool

	// StoreCheck verifies exec()'s cmd and args that name files in the
	// local store. Nil means no check.
	StoreCheck func(path string) error
}

// CheckSignature verifies data, the content of the module at path, when the
// loader has a Keyring. Without one it does nothing.
func (l *Loader) CheckSignature(path string, data []byte) error {
	if l.Keyring == nil {
		return nil
	}
	return l.Keyring.CheckFile(l.Fsys, path, data)
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
	case strings.HasPrefix(module, "//"):
		logging.Log("Loader.resolveModulePath", deck.V(4), "info", "resolving workspace-relative module path %q with WorkspacePath %q", module, l.WorkspacePath)
		// Workspace-relative
		stripped := module[2:]
		resolved := filepath.Join(l.WorkspacePath, stripped)
		logging.Log("Loader.resolveModulePath", deck.V(4), "info", "stripped to %q, resolved to %q", stripped, resolved)
		return resolved
	case filepath.IsAbs(module):
		logging.Log("Loader.resolveModulePath", deck.V(4), "info", "resolving absolute module path %q", module)
		return module
	case len(thread.CallStack()) > 0:
		logging.Log("Loader.resolveModulePath", deck.V(4), "info", "resolving caller-relative module path %q", module)
		// Relative to the caller module
		caller := thread.CallStack().At(0)
		callerDir := filepath.Dir(caller.Pos.Filename())
		return filepath.Join(callerDir, module)

	default:
		logging.Log("Loader.resolveModulePath", deck.V(4), "info", "resolving default workspace-relative module path %q", module)
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
	if err := l.CheckSignature(modulePath, data); err != nil {
		return nil, err
	}

	thread := &starlark.Thread{
		Name:  threadName,
		Load:  loadFunc,
		Print: func(_ *starlark.Thread, msg string) { fmt.Println(msg) },
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

func WithKeyring(k *signing.Keyring) LoaderOption {
	return func(l *Loader) {
		l.Keyring = k
	}
}

func WithVars(vars map[string]string) LoaderOption {
	return func(l *Loader) {
		l.Vars = vars
	}
}

func WithStore(open func() (*localstore.Store, error)) LoaderOption {
	return func(l *Loader) {
		l.Store = open
	}
}

func WithStoreOnly(storeOnly bool) LoaderOption {
	return func(l *Loader) {
		l.StoreOnly = storeOnly
	}
}

func WithStoreCheck(check func(path string) error) LoaderOption {
	return func(l *Loader) {
		l.StoreCheck = check
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

func Default(ctx context.Context, fsys afero.Fs, ex starcmshelllib.Executor, workspacePath string, opts ...LoaderOption) Loader {
	var l Loader
	l = NewLoader(
		ctx,
		WithWorkspacePath(workspacePath),
		WithFsys(fsys),
		WithPredeclared(func(module string) (starlark.StringDict, error) {
			switch module {
			case "starcm":
				names := make([]string, 0, len(l.Vars))
				for name := range l.Vars {
					names = append(names, name)
				}
				sort.Strings(names)
				vars := starlark.NewDict(len(names))
				for _, name := range names {
					if err := vars.SetKey(starlark.String(name), starlark.String(l.Vars[name])); err != nil {
						return nil, err
					}
				}
				vars.Freeze()
				return starlark.StringDict{
					"download": starlark.NewBuiltin(
						"download",
						starcmdownload.New(
							ctx,
							*http.DefaultClient,
							fsys,
							os.Stdout,
							starcmdownload.WithStore(l.Store),
							starcmdownload.StoreOnly(l.StoreOnly),
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
							starcmshell.WithStoreCheck(l.StoreCheck),
						).Function(),
					),
					"file": starlark.NewBuiltin(
						"file",
						starcmFile.New(ctx, fsys).Function(),
					),
					"template": starlark.NewBuiltin(
						"template",
						starcmtemplate.New(ctx, fsys).Function(),
					),
					"date": starlark.NewBuiltin(
						"date",
						starcmdate.New(ctx).Function(),
					),
					"load_dynamic": starlark.NewBuiltin(
						"load_dynamic",
						dynamicloading.New(ctx).Function(),
					),
					"verify": starlark.NewBuiltin(
						"verify",
						starcmverify.New(ctx, fsys, l.Keyring).Function(),
					),
					"list_files": starlark.NewBuiltin(
						"list_files",
						starcmlistfiles.New(ctx, fsys).Function(),
					),
					"read_file": starlark.NewBuiltin(
						"read_file",
						starcmreadfile.New(ctx, fsys).Function(),
					),
					"install_mac_pkg": starlark.NewBuiltin(
						"install_mac_pkg",
						starcmmacpkg.New(
							ctx,
							ex,
							starcmmacpkg.WithStoreCheck(l.StoreCheck),
						).Function(),
					),
					"sha256": starlark.NewBuiltin(
						"sha256",
						starcmchecksum.New(ctx, fsys).Function(),
					),
					"inspect": starlark.NewBuiltin(
						"inspect",
						starcminspect.New(ctx, fsys).Function(),
					),
					"sign": starlark.NewBuiltin(
						"sign",
						// Disabled under --trusted-keys: deployed runs never sign.
						starcmsign.New(ctx, fsys, l.Keyring != nil).Function(),
					),
					"json": starlarkjson.Module,
					"vars": vars,
				}, nil
			case "stdlib":
				return starlark.StringDict{
					"struct": starlark.NewBuiltin("struct", starlarkstruct.Make),
				}, nil
			default:
				// set both to nil to allow the loader to load a .star file from a path.
				return nil, nil
			}
		}),
	)
	// Applied last so Predeclared above sees the final Keyring, Vars and Store.
	for _, opt := range opts {
		opt(&l)
	}
	return l
}
