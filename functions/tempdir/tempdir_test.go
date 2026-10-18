package tempdir

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
)

// setTempDir points os.TempDir() at dir on every platform.
func setTempDir(t *testing.T, dir string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Setenv("TMP", dir)
		t.Setenv("TEMP", dir)
	} else {
		t.Setenv("TMPDIR", dir)
	}
}

func call(t *testing.T, args ...starlark.Value) (starlark.Value, error) {
	t.Helper()
	return starlark.Call(&starlark.Thread{}, New(), starlark.Tuple(args), nil)
}

func TestTempDir(t *testing.T) {
	dir := t.TempDir()
	// macOS's $TMPDIR ends in a slash; the result never does.
	setTempDir(t, dir+string(filepath.Separator))

	v, err := call(t)
	require.NoError(t, err)
	assert.Equal(t, starlark.String(dir), v)

	v, err = call(t, starlark.String("starcm"), starlark.String("motd.txt"))
	require.NoError(t, err)
	assert.Equal(t, starlark.String(filepath.Join(dir, "starcm", "motd.txt")), v)
}

func TestTempDirErrors(t *testing.T) {
	_, err := call(t, starlark.MakeInt(1))
	assert.EqualError(t, err, "temp_dir: argument 1: got int, want string")

	_, err = starlark.Call(&starlark.Thread{}, New(), nil, []starlark.Tuple{{starlark.String("x"), starlark.String("y")}})
	assert.EqualError(t, err, "temp_dir: unexpected keyword argument \"x\"")
}
