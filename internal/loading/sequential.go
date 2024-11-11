package loading

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

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

				// if we hit a load statement in a .star file
				//  load the next module relative to the current module
				if len(thread.CallStack()) > 0 {
					modulepath = filepath.Dir(thread.CallStack().At(0).Pos.Filename())
					modulepath = path.Join(modulepath, module)
				}

				data, err := os.ReadFile(modulepath)
				if err != nil {
					return nil, fmt.Errorf("loading module %q: %s", modulepath, err)
				}

				// create a thread for the module and set Load
				thread := &starlark.Thread{Name: "exec " + module, Load: load}
				globals, err := starlark.ExecFileOptions(
					&syntax.FileOptions{
						// Allow if statements and for loops to be top-level in the module.
						TopLevelControl: true,
					},
					thread,
					module,
					data,
					nil,
				)
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
