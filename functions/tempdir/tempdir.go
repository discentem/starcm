// Package tempdir implements temp_dir(), which returns the system's
// temporary directory.
package tempdir

import (
	"fmt"
	"os"
	"path/filepath"

	"go.starlark.net/starlark"
)

// Path is os.TempDir() joined with parts and cleaned, so it has no trailing
// separator: $TMPDIR (or /tmp) on macOS and Linux, %TMP% / %TEMP% on Windows.
func Path(parts ...string) string {
	return filepath.Join(append([]string{os.TempDir()}, parts...)...)
}

// New returns the temp_dir() builtin. Positional string arguments are
// joined onto the directory with the platform's separator:
//
//	temp_dir()                     # "/var/folders/.../T" on macOS
//	temp_dir("starcm", "motd.txt") # ".../T/starcm/motd.txt"
func New() *starlark.Builtin {
	return starlark.NewBuiltin("temp_dir", func(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if len(kwargs) > 0 {
			return nil, fmt.Errorf("%s: unexpected keyword argument %s", b.Name(), kwargs[0][0])
		}
		parts := make([]string, len(args))
		for i, arg := range args {
			s, ok := starlark.AsString(arg)
			if !ok {
				return nil, fmt.Errorf("%s: argument %d: got %s, want string", b.Name(), i+1, arg.Type())
			}
			parts[i] = s
		}
		return starlark.String(Path(parts...)), nil
	})
}
