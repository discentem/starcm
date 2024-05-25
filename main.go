package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/discentem/starcm/libraries/shell"
	starcmexampleMod "github.com/discentem/starcm/modules/example"
	starcmshell "github.com/discentem/starcm/modules/shell"
	"github.com/google/deck"
	"github.com/google/deck/backends/logger"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

type loaderFunc func(_ *starlark.Thread, module string) (starlark.StringDict, error)

func LoadFromFile(ctx context.Context, filepath string, src interface{}, load loaderFunc) error {
	thread := &starlark.Thread{
		Load:  load,
		Name:  "my_program_main_thread",
		Print: func(_ *starlark.Thread, msg string) { fmt.Println(msg) },
	}
	if _, err := starlark.ExecFile(thread, filepath, src, nil); err != nil {
		if evalErr, ok := err.(*starlark.EvalError); ok {
			return fmt.Errorf(evalErr.Backtrace())
		}
		return fmt.Errorf("load at path: %q: %s", filepath, err)
	}
	return nil
}

// Loader handles module loading for starlark modules.
type Loader struct {
	// Predeclared is used for builtin modules which are not loaded from a path.
	Predeclared func(module string) (starlark.StringDict, error)

	// WorkspacePath specifies the path to the source directory.
	WorkspacePath string
}

// Sequential implements sequential module loading.
// Module paths starting with "//" will be loaded from WorkspacePath, which should be the mount path to the workspace source directory.
func (l *Loader) Sequential(ctx context.Context) func(thread *starlark.Thread, module string) (starlark.StringDict, error) {
	type entry struct {
		globals starlark.StringDict
		err     error
	}

	var cache = make(map[string]*entry)

	// load is set for the thread.Load when caching a new entry.
	var load func(_ *starlark.Thread, module string) (starlark.StringDict, error)
	load = func(thread *starlark.Thread, module string) (starlark.StringDict, error) {
		e, ok := cache[module]
		if e == nil {
			if ok {
				return nil, fmt.Errorf("cycle in load graph %q", module)
			}

			cache[module] = nil

			builtin, err := l.Predeclared(module)
			if builtin != nil || err != nil {
				e = &entry{builtin, err}
			} else {
				if path.Ext(module) != ".star" {
					return nil, fmt.Errorf("module %q is not valid, modules should have a .star extension", module)
				}

				// shorthand for a workspace path
				modulepath := module
				if strings.HasPrefix(module, "//") {
					modulepath = path.Join(l.WorkspacePath, module)
				}

				data, err := os.ReadFile(modulepath)
				if err != nil {
					return nil, fmt.Errorf("loading module %q: %s", modulepath, err)
				}

				// create a thread for the module and set Load
				thread := &starlark.Thread{Name: "exec " + module, Load: load}
				globals, err := starlark.ExecFile(thread, module, data, nil)
				if err != nil {
					return nil, fmt.Errorf("executing module %q: %s", module, err)
				}
				// Step 2: Extract relevant information from the function

				e = &entry{globals, err}
			}

			cache[module] = e

		}
		return e.globals, e.err
	}
	return load
}

func main() {
	f := flag.String("file", "example.star", "path to the starlark file")
	verbosity := flag.Int("v", 2, "verbosity level")
	flag.Parse()
	if f == nil {
		args := os.Args
		if len(args) == 0 {
			log.Fatal("no .star file provided")
		}
		f = &args[0]
	}
	deck.Add(logger.Init(os.Stdout, 0))
	deck.Info("starting starcm...")
	deck.SetVerbosity(*verbosity)

	loader := Loader{
		WorkspacePath: ".",
		Predeclared: func(module string) (starlark.StringDict, error) {
			switch module {
			case "shellout":
				return starlark.StringDict{
					"exec": starlark.NewBuiltin(
						"exec",
						starcmshell.New(
							&shell.RealExecutor{},
						).Function(),
					),
				}, nil
			case "struct":
				return starlark.StringDict{
					"struct": starlark.NewBuiltin("struct", starlarkstruct.Make),
				}, nil
			case "example":
				return starlark.StringDict{
					"example": starlark.NewBuiltin("example", starcmexampleMod.New().Function()),
				}, nil
			default:
				// set both to nil to allow the loader to load a .star file from a path.
				return nil, nil
			}
		},
	}

	err := LoadFromFile(
		context.Background(),
		*f,
		nil,
		loader.Sequential(context.Background()),
	)
	if err != nil {
		fmt.Println(err)
	}

}
