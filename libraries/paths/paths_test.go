package paths

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mitchellh/go-homedir"
	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
)

// abs returns a platform-appropriate absolute path for the given parts.
func abs(parts ...string) string {
	root := "/"
	if runtime.GOOS == "windows" {
		root = `C:\`
	}
	return filepath.Join(append([]string{root}, parts...)...)
}

func home(t *testing.T) string {
	t.Helper()
	h, err := homedir.Expand("~")
	require.NoError(t, err)
	return h
}

func TestAbs(t *testing.T) {
	got, err := Abs(abs("etc", "..", "etc", "motd"))
	require.NoError(t, err)
	require.Equal(t, abs("etc", "motd"), got)

	got, err = Abs("~/x.txt")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(home(t), "x.txt"), got)

	_, err = Abs("x.txt")
	require.ErrorContains(t, err, `"x.txt" is not an absolute path`)
	require.ErrorContains(t, err, `here("x.txt")`)
}

func TestResource(t *testing.T) {
	dir := abs("repo", "site")

	got, err := Resource(dir, "motd.jinja")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dir, "motd.jinja"), got)

	got, err = Resource(dir, filepath.Join("..", "shared", "a.jinja"))
	require.NoError(t, err)
	require.Equal(t, abs("repo", "shared", "a.jinja"), got)

	got, err = Resource(dir, abs("elsewhere", "b.jinja"))
	require.NoError(t, err)
	require.Equal(t, abs("elsewhere", "b.jinja"), got)

	got, err = Resource(dir, "~/c.jinja")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(home(t), "c.jinja"), got)

	_, err = Resource("", "motd.jinja")
	require.ErrorContains(t, err, "no .star file")
}

// CallerDir reports the file the builtin call is written in, including when
// the call is inside a function defined in another file.
func TestCallerDir(t *testing.T) {
	where := starlark.NewBuiltin("where", func(thread *starlark.Thread, _ *starlark.Builtin, _ starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
		dir, ok := CallerDir(thread)
		if !ok {
			return starlark.None, nil
		}
		return starlark.String(dir), nil
	})
	predeclared := starlark.StringDict{"where": where}
	thread := &starlark.Thread{}

	libFile := abs("repo", "lib", "lib.star")
	lib, err := starlark.ExecFile(thread, libFile, "def f():\n    return where()\n", predeclared)
	require.NoError(t, err)

	mainFile := abs("repo", "site", "main.star")
	globals, err := starlark.ExecFile(thread, mainFile, "here_top = where()\nin_lib = f()\n",
		starlark.StringDict{"where": where, "f": lib["f"]})
	require.NoError(t, err)

	require.Equal(t, starlark.String(abs("repo", "site")), globals["here_top"])
	require.Equal(t, starlark.String(abs("repo", "lib")), globals["in_lib"])
}
