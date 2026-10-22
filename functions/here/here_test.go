package here

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
)

func abs(parts ...string) string {
	root := "/"
	if runtime.GOOS == "windows" {
		root = `C:\`
	}
	return filepath.Join(append([]string{root}, parts...)...)
}

func run(t *testing.T, inStore func(string) bool, file, src string, extra starlark.StringDict) (starlark.StringDict, error) {
	t.Helper()
	predeclared := starlark.StringDict{"here": New(inStore)}
	for k, v := range extra {
		predeclared[k] = v
	}
	return starlark.ExecFile(&starlark.Thread{}, file, src, predeclared)
}

func TestHere(t *testing.T) {
	file := abs("repo", "site", "web.star")
	globals, err := run(t, nil, file, `
dir = here()
motd = here("motd.jinja")
nested = here("templates", "a.jinja")
sibling = here("..", "shared", "b.jinja")
`, nil)
	require.NoError(t, err)
	require.Equal(t, starlark.String(abs("repo", "site")), globals["dir"])
	require.Equal(t, starlark.String(abs("repo", "site", "motd.jinja")), globals["motd"])
	require.Equal(t, starlark.String(abs("repo", "site", "templates", "a.jinja")), globals["nested"])
	require.Equal(t, starlark.String(abs("repo", "shared", "b.jinja")), globals["sibling"])
}

// here() resolves next to the file the call is written in: a helper in lib/
// that calls here() gets lib/, and a caller in site/ that calls here() and
// passes the result to the helper gets site/.
func TestHereIsLexical(t *testing.T) {
	lib, err := run(t, nil, abs("repo", "lib", "render.star"), `
def bundled():
    return here("banner.jinja")

def passthrough(p):
    return p
`, nil)
	require.NoError(t, err)

	globals, err := run(t, nil, abs("repo", "site", "web.star"), `
from_lib = bundled()
from_site = passthrough(here("motd.jinja"))
`, starlark.StringDict{"bundled": lib["bundled"], "passthrough": lib["passthrough"]})
	require.NoError(t, err)
	require.Equal(t, starlark.String(abs("repo", "lib", "banner.jinja")), globals["from_lib"])
	require.Equal(t, starlark.String(abs("repo", "site", "motd.jinja")), globals["from_site"])
}

func TestHereRefusesStoreModules(t *testing.T) {
	store := abs("store")
	inStore := func(p string) bool { return strings.HasPrefix(p, store+string(filepath.Separator)) }
	_, err := run(t, inStore, abs("store", "sha256", "abc", "remote.star"), `here("x")`, nil)
	require.ErrorContains(t, err, "not available in a module loaded with load_remote")

	globals, err := run(t, inStore, abs("repo", "local.star"), `p = here("x")`, nil)
	require.NoError(t, err)
	require.Equal(t, starlark.String(abs("repo", "x")), globals["p"])
}

func TestHereArgumentErrors(t *testing.T) {
	file := abs("repo", "a.star")
	for _, c := range []struct{ src, want string }{
		{`here("a", 1)`, "here: argument 2: got int, want string"},
		{`here(` + strings.ReplaceAll(`"`+abs("etc")+`"`, `\`, `\\`) + `)`, "is absolute"},
		{`here(name = "a")`, `here: unexpected keyword argument "name"`},
	} {
		_, err := run(t, nil, file, c.src, nil)
		require.ErrorContains(t, err, c.want, c.src)
	}
}
