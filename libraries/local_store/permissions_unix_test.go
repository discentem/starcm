//go:build !windows

package localstore

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenSetsModes(t *testing.T) {
	// Modes must not depend on the umask.
	old := syscall.Umask(0o077)
	defer syscall.Umask(old)

	s := openStore(t)
	fi, err := os.Stat(s.Dir())
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o755), fi.Mode().Perm())

	path, err := s.Put(contentHash, "payload.txt", strings.NewReader(content))
	require.NoError(t, err)
	fi, err = os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o555), fi.Mode().Perm())
}

func TestCheckRefusesWritableByOthers(t *testing.T) {
	perms, err := NewPermissions()
	require.NoError(t, err)

	for _, mode := range []os.FileMode{0o775, 0o757, 0o777 | os.ModeSticky} {
		dir := filepath.Join(t.TempDir(), "store")
		require.NoError(t, os.Mkdir(dir, 0o755))
		require.NoError(t, os.Chmod(dir, mode))

		_, err := Open(dir, perms)
		require.ErrorContains(t, err, "writable by group or others", "mode %v", mode)
	}
}

func TestCheckRefusesWritableParent(t *testing.T) {
	perms, err := NewPermissions()
	require.NoError(t, err)

	parent := t.TempDir()
	require.NoError(t, os.Chmod(parent, 0o777))
	_, err = Open(filepath.Join(parent, "store"), perms)
	require.ErrorContains(t, err, "writable by group or others")
}

func TestCheckAllowsStickyParent(t *testing.T) {
	perms, err := NewPermissions()
	require.NoError(t, err)

	parent := t.TempDir()
	require.NoError(t, os.Chmod(parent, 0o777|os.ModeSticky))
	_, err = Open(filepath.Join(parent, "store"), perms)
	require.NoError(t, err)
}

func TestOpenResolvesSymlinks(t *testing.T) {
	perms, err := NewPermissions()
	require.NoError(t, err)

	tmp := t.TempDir()
	real := filepath.Join(tmp, "real")
	require.NoError(t, os.Mkdir(real, 0o755))
	link := filepath.Join(tmp, "link")
	require.NoError(t, os.Symlink(real, link))

	s, err := Open(link, perms)
	require.NoError(t, err)
	want, err := filepath.EvalSymlinks(real)
	require.NoError(t, err)
	require.Equal(t, want, s.Dir())
}
