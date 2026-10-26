// Package pathmod implements the path module: path.exists(), path.join(),
// path.glob() and path.list(). They only read, so they're plain queries
// that return their answer directly.
package pathmod

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"syscall"

	"github.com/discentem/starcm/functions/base"
	"github.com/discentem/starcm/functions/glob"
	"github.com/discentem/starcm/functions/listfiles"
	"github.com/spf13/afero"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// New returns the path module, which reads the filesystem through fsys.
//
//	path.join(home_dir(), "go", "bin")         # "/Users/me/go/bin"
//	path.exists("/usr/local/bin/go")           # True or False
//	path.glob(here("third_party/*/go.mod"))    # ["/.../a/go.mod", ...]
//	path.list(here("config"), pattern = "*.star")
func New(ctx context.Context, fsys afero.Fs) *starlarkstruct.Module {
	return &starlarkstruct.Module{
		Name: "path",
		Members: starlark.StringDict{
			"exists": Exists(ctx, fsys).Builtin(),
			"join":   Join(),
			"glob":   glob.New(ctx, fsys).Builtin(),
			"list":   listfiles.New(ctx, fsys).Builtin(),
		},
	}
}

// Exists returns path.exists(). It reports whether path, an absolute path
// (~ expands), names a file or directory, following symlinks. It returns
// False only when the path doesn't exist; any other error, such as
// permission denied, stops the run, so a path starcm can't check never
// looks missing.
func Exists(ctx context.Context, fsys afero.Fs) *base.Query {
	return base.NewQuery(ctx, "path.exists", []base.ArgPair{base.Arg[string]("path")}, func(inst *base.FunctionInstance) (starlark.Value, error) {
		p, _, err := inst.AbsPath("path")
		if err != nil {
			return nil, err
		}
		_, err = fsys.Stat(p)
		switch {
		case err == nil:
			return starlark.True, nil
		// A file where the path needs a directory ("/etc/hosts/x") means
		// the path doesn't exist, too.
		case errors.Is(err, fs.ErrNotExist), errors.Is(err, syscall.ENOTDIR):
			return starlark.False, nil
		default:
			return nil, inst.Errorf("%w", err)
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
