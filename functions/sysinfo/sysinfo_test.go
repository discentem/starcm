package sysinfo

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mitchellh/go-homedir"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

func init() {
	// go-homedir caches the first answer it finds; these tests change HOME.
	homedir.DisableCache = true
}

func call(t *testing.T, b *starlark.Builtin, args ...starlark.Value) (starlark.Value, error) {
	t.Helper()
	return starlark.Call(&starlark.Thread{}, b, starlark.Tuple(args), nil)
}

func TestPlatform(t *testing.T) {
	v, err := starlark.Call(&starlark.Thread{}, Platform(), nil, nil)
	require.NoError(t, err)
	p, ok := v.(*starlarkstruct.Struct)
	require.True(t, ok, "platform() returns %s, want struct", v.Type())

	goos, err := p.Attr("os")
	require.NoError(t, err)
	assert.Equal(t, starlark.String(runtime.GOOS), goos)

	goarch, err := p.Attr("arch")
	require.NoError(t, err)
	assert.Equal(t, starlark.String(runtime.GOARCH), goarch)
}

func TestPlatformErrors(t *testing.T) {
	_, err := call(t, Platform(), starlark.String("darwin"))
	assert.EqualError(t, err, "platform: takes no arguments")

	_, err = starlark.Call(&starlark.Thread{}, Platform(), nil, []starlark.Tuple{{starlark.String("os"), starlark.String("darwin")}})
	assert.EqualError(t, err, "platform: takes no arguments")
}

func TestEnv(t *testing.T) {
	const name = "STARCM_SYSINFO_TEST"

	t.Setenv(name, "us-west-2")
	v, err := call(t, Env(), starlark.String(name))
	require.NoError(t, err)
	assert.Equal(t, starlark.String("us-west-2"), v)

	// A set variable wins over default, even when it's empty.
	t.Setenv(name, "")
	v, err = starlark.Call(&starlark.Thread{}, Env(), starlark.Tuple{starlark.String(name)}, []starlark.Tuple{{starlark.String("default"), starlark.String("fallback")}})
	require.NoError(t, err)
	assert.Equal(t, starlark.String(""), v)

	// An unset variable returns default, or None when there is none.
	require.NoError(t, os.Unsetenv(name))
	v, err = starlark.Call(&starlark.Thread{}, Env(), starlark.Tuple{starlark.String(name)}, []starlark.Tuple{{starlark.String("default"), starlark.String("fallback")}})
	require.NoError(t, err)
	assert.Equal(t, starlark.String("fallback"), v)

	v, err = call(t, Env(), starlark.String(name))
	require.NoError(t, err)
	assert.Equal(t, starlark.None, v)
}

func TestEnvErrors(t *testing.T) {
	_, err := call(t, Env())
	assert.Error(t, err)

	_, err = call(t, Env(), starlark.MakeInt(1))
	assert.Error(t, err)

	_, err = starlark.Call(&starlark.Thread{}, Env(), starlark.Tuple{starlark.String("X")}, []starlark.Tuple{{starlark.String("default"), starlark.MakeInt(1)}})
	assert.EqualError(t, err, "env: default: got int, want string or None")
}

func TestHomeDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	v, err := call(t, HomeDir())
	require.NoError(t, err)
	assert.Equal(t, starlark.String(home), v)

	// A trailing separator in HOME doesn't end up in the result.
	t.Setenv("HOME", home+string(filepath.Separator))
	v, err = call(t, HomeDir())
	require.NoError(t, err)
	assert.Equal(t, starlark.String(home), v)

	v, err = call(t, HomeDir(), starlark.String(".config"), starlark.String("agent.toml"))
	require.NoError(t, err)
	assert.Equal(t, starlark.String(filepath.Join(home, ".config", "agent.toml")), v)
}

func TestHomeDirErrors(t *testing.T) {
	_, err := call(t, HomeDir(), starlark.MakeInt(1))
	assert.EqualError(t, err, "home_dir: argument 1: got int, want string")

	_, err = starlark.Call(&starlark.Thread{}, HomeDir(), nil, []starlark.Tuple{{starlark.String("x"), starlark.String("y")}})
	assert.EqualError(t, err, "home_dir: unexpected keyword argument \"x\"")
}
