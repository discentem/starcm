// Package here implements the plain helper here(), which builds absolute paths
// to files that ship next to a .star file.
package here

import (
	"fmt"
	"path/filepath"

	"github.com/discentem/starcm/libraries/paths"
	"go.starlark.net/starlark"
)

// New returns the here() builtin. It returns the directory of the .star file
// the here() call is written in, with the string parts joined on:
//
//	here("templates", "motd.jinja")   # "/repo/site/templates/motd.jinja"
//
// check tells where a .star file runs from. A single module loaded with
// load_remote runs from the store, where files that would ship next to it
// don't exist, so here() refuses to run in it. A module in a folder fetched
// with load_remote works as in a checkout, except that the path can't leave
// the folder. check may be nil.
func New(check paths.StoreCheck) *starlark.Builtin {
	return starlark.NewBuiltin("here", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if len(kwargs) > 0 {
			return nil, fmt.Errorf("%s: unexpected keyword argument %s", b.Name(), kwargs[0][0])
		}
		parts := make([]string, len(args))
		for i, arg := range args {
			s, ok := starlark.AsString(arg)
			if !ok {
				return nil, fmt.Errorf("%s: argument %d: got %s, want string", b.Name(), i+1, arg.Type())
			}
			if filepath.IsAbs(s) {
				return nil, fmt.Errorf("%s: argument %d: %q is absolute; here() joins parts onto the .star file's directory", b.Name(), i+1, s)
			}
			parts[i] = s
		}
		file, ok := paths.CallerFile(thread)
		if !ok {
			return nil, fmt.Errorf("%s: can't tell which .star file the call is in", b.Name())
		}
		var origin paths.Origin
		if check != nil {
			origin = check(file)
		}
		if !origin.Shipped() {
			return nil, fmt.Errorf("%s: not available in a module loaded with load_remote: it runs from starcm's store, so nothing ships next to it", b.Name())
		}
		// Slash separated, so errors read the same on every platform.
		p, err := origin.Resolve(filepath.Dir(file), filepath.ToSlash(filepath.Join(parts...)))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", b.Name(), err)
		}
		return starlark.String(p), nil
	})
}
