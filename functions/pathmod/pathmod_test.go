package pathmod

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
)

func call(t *testing.T, b *starlark.Builtin, args ...starlark.Value) (starlark.Value, error) {
	t.Helper()
	return starlark.Call(&starlark.Thread{}, b, starlark.Tuple(args), nil)
}

func TestExists(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "f.txt")
	require.NoError(t, os.WriteFile(file, nil, 0o644))
	exists := Exists(afero.NewOsFs())

	for _, tc := range []struct {
		path string
		want bool
	}{
		{file, true},
		{dir, true},
		{filepath.Join(dir, "missing"), false},
		{filepath.Join(dir, "missing", "deeper"), false},
		// A file where a directory should be.
		{filepath.Join(file, "x"), false},
	} {
		v, err := call(t, exists, starlark.String(tc.path))
		require.NoError(t, err, tc.path)
		assert.Equal(t, starlark.Bool(tc.want), v, tc.path)
	}
}

func TestExistsTilde(t *testing.T) {
	homedir.DisableCache = true
	t.Cleanup(func() { homedir.DisableCache = false })
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	require.NoError(t, os.WriteFile(filepath.Join(home, "f.txt"), nil, 0o644))
	v, err := call(t, Exists(afero.NewOsFs()), starlark.String("~/f.txt"))
	require.NoError(t, err)
	assert.Equal(t, starlark.True, v)
}

func TestExistsSymlink(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(dir, "link")
	if err := os.Symlink(filepath.Join(dir, "missing"), link); err != nil {
		t.Skipf("can't create a symlink here: %v", err)
	}
	// A link to nothing doesn't exist: path.exists follows links.
	v, err := call(t, Exists(afero.NewOsFs()), starlark.String(link))
	require.NoError(t, err)
	assert.Equal(t, starlark.False, v)
}

// An error other than "doesn't exist" stops the run instead of returning False.
func TestExistsUnreadable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permissions work differently on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root can read any directory")
	}
	dir := t.TempDir()
	locked := filepath.Join(dir, "locked")
	require.NoError(t, os.MkdirAll(filepath.Join(locked, "inner"), 0o755))
	require.NoError(t, os.Chmod(locked, 0o000))
	t.Cleanup(func() { os.Chmod(locked, 0o755) })
	_, err := call(t, Exists(afero.NewOsFs()), starlark.String(filepath.Join(locked, "inner")))
	assert.ErrorIs(t, err, os.ErrPermission)
	assert.ErrorContains(t, err, "path.exists: ")
}

func TestExistsErrors(t *testing.T) {
	exists := Exists(afero.NewOsFs())
	_, err := call(t, exists, starlark.String("relative.txt"))
	assert.ErrorContains(t, err, `path.exists: "relative.txt" is not an absolute path`)
	_, err = call(t, exists)
	assert.Error(t, err)
	_, err = call(t, exists, starlark.MakeInt(1))
	assert.Error(t, err)
	_, err = call(t, exists, starlark.String("/a"), starlark.String("/b"))
	assert.Error(t, err)
}

func TestJoin(t *testing.T) {
	sep := string(filepath.Separator)
	for _, tc := range []struct {
		parts []string
		want  string
	}{
		{[]string{"a", "b", "c.txt"}, "a" + sep + "b" + sep + "c.txt"},
		{[]string{"a/", "/b"}, "a" + sep + "b"},
		{[]string{"a", "..", "b"}, "b"},
		{[]string{"a", "", "b"}, "a" + sep + "b"},
		{[]string{""}, ""},
		{[]string{"a/b/"}, "a" + sep + "b"},
	} {
		args := make([]starlark.Value, len(tc.parts))
		for i, p := range tc.parts {
			args[i] = starlark.String(p)
		}
		v, err := call(t, Join(), args...)
		require.NoError(t, err)
		assert.Equal(t, starlark.String(tc.want), v, "%q", tc.parts)
	}
}

func TestJoinErrors(t *testing.T) {
	_, err := call(t, Join())
	assert.EqualError(t, err, "path.join: got no arguments, want at least one")
	_, err = call(t, Join(), starlark.String("a"), starlark.MakeInt(1))
	assert.EqualError(t, err, "path.join: argument 2: got int, want string")
	_, err = starlark.Call(&starlark.Thread{}, Join(), nil, []starlark.Tuple{{starlark.String("x"), starlark.String("y")}})
	assert.EqualError(t, err, `path.join: unexpected keyword argument "x"`)
}

func TestModule(t *testing.T) {
	m := New(afero.NewOsFs())
	assert.Equal(t, "path", m.Name)
	assert.ElementsMatch(t, []string{"exists", "join"}, m.Members.Keys())
}
