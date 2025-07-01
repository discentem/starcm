package loading

import (
	"fmt"
	"path/filepath"
	"regexp/syntax"

	"github.com/discentem/starcm/libraries/logging"
	starlarkhelpers "github.com/discentem/starcm/starlark-helpers"
	"github.com/google/deck"
	"go.starlark.net/starlark"
)

func Load() starlarkhelpers.Function {
	return func(thread *starlark.Thread, builtin *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var modulepath string
		var absolutePath bool
		err := starlark.UnpackArgs(
			builtin.Name(),
			args,
			kwargs,
			"module_path", &modulepath,
			"absolute_path??", &absolutePath)
		if err != nil {
			return nil, err
		}
		if !absolutePath {
			if len(thread.CallStack()) > 0 {
				logging.Log(builtin.Name(), deck.V(3), "info", "using call stack to determine relative module path")
				dirName := filepath.Dir(thread.CallStack().At(1).Pos.Filename())
				modulepath = filepath.Join(dirName, modulepath)
			}
		}
		logging.Log(builtin.Name(), deck.V(3), "info", "Loading module from path: %q", modulepath)
		modulepath, err = filepath.Abs(modulepath)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for module %q: %w", modulepath, err)
		}
		logging.Log(builtin.Name(), deck.V(3), "info", "absolute module path: %q", modulepath)
		module, err := thread.Load(thread, modulepath)
		if err != nil {
			switch e := err.(type) {
			case *starlark.EvalError:
				return nil, fmt.Errorf("load failed:\n%s", e.Backtrace())
			case *syntax.Error:
				// Handle syntax errors (e.g. parse errors in the .star file)
				return nil, fmt.Errorf("syntax errors:\n%s", e.Error()) // e is a slice of syntax.Error
			default:
				return nil, fmt.Errorf("unexpected error during load: %w", err)
			}
		}

		dict := starlark.NewDict(len(module))
		for key, val := range module {
			err = dict.SetKey(starlark.String(key), val)
			if err != nil {
				return nil, err
			}
		}
		dict.Freeze()
		return dict, err
	}
}
