package loading

import (
	"path/filepath"

	"github.com/discentem/starcm/libraries/logging"
	starlarkhelpers "github.com/discentem/starcm/starlark-helpers"
	"github.com/google/deck"
	"go.starlark.net/starlark"
)

func DynamicLoadfunc() starlarkhelpers.Function {
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
				dirName := filepath.Dir(thread.CallStack().At(1).Pos.Filename())
				modulepath = filepath.Join(dirName, modulepath)
			}
		}
		logging.Log(builtin.Name(), deck.V(3), "info", "Loading module from path: %q", modulepath)
		module, err := thread.Load(thread, modulepath)
		if err != nil {
			return nil, err
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
