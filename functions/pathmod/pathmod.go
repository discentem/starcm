// Package pathmod implements the path module: the plain helpers
// path.exists() and path.join().
package pathmod

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"syscall"

	"github.com/discentem/starcm/libraries/paths"
	"github.com/spf13/afero"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// New returns the path module, which reads the filesystem through fsys.
//
//	path.join(home_dir(), "go", "bin")   # "/Users/me/go/bin"
//	path.exists("/usr/local/bin/go")     # True or False
func New(fsys afero.Fs) *starlarkstruct.Module {
	return &starlarkstruct.Module{
		Name: "path",
		Members: starlark.StringDict{
			"exists": Exists(fsys),
			"join":   Join(),
		},
	}
}

// Exists returns the path.exists() builtin. It reports whether p, an
// absolute path (~ expands), names a file or directory, following symlinks.
// It returns False only when p doesn't exist; any other error, such as
// permission denied, stops the run, so a path starcm can't check never looks
// missing.
func Exists(fsys afero.Fs) *starlark.Builtin {
	return starlark.NewBuiltin("path.exists", func(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var p string
		if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &p); err != nil {
			return nil, err
		}
		abs, err := paths.Abs(p)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", b.Name(), err)
		}
		_, err = fsys.Stat(abs)
		switch {
		case err == nil:
			return starlark.True, nil
		// A file where p needs a directory ("/etc/hosts/x") means p
		// doesn't exist, too.
		case errors.Is(err, fs.ErrNotExist), errors.Is(err, syscall.ENOTDIR):
			return starlark.False, nil
		default:
			return nil, fmt.Errorf("%s: %w", b.Name(), err)
		}
	})
}

// Join returns the path.join() builtin. It joins its string parts with the
// platform's separator and cleans the result, as Go's filepath.Join does.
// Empty parts are ignored. It doesn't touch the filesystem or expand ~.
func Join() *starlark.Builtin {
	return starlark.NewBuiltin("path.join", func(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if len(kwargs) > 0 {
			return nil, fmt.Errorf("%s: unexpected keyword argument %s", b.Name(), kwargs[0][0])
		}
		if len(args) == 0 {
			return nil, fmt.Errorf("%s: got no arguments, want at least one", b.Name())
		}
		parts := make([]string, len(args))
		for i, arg := range args {
			s, ok := starlark.AsString(arg)
			if !ok {
				return nil, fmt.Errorf("%s: argument %d: got %s, want string", b.Name(), i+1, arg.Type())
			}
			parts[i] = s
		}
		return starlark.String(filepath.Join(parts...)), nil
	})
}
