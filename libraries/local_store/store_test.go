package localstore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	starcmsha256 "github.com/discentem/starcm/libraries/sha256"
	"github.com/stretchr/testify/require"
)

const content = "starcm store fixture\n"

var contentHash = func() string {
	h, err := starcmsha256.FromReader(strings.NewReader(content))
	if err != nil {
		panic(err)
	}
	return h
}()

func openStore(t *testing.T) *Store {
	t.Helper()
	perms, err := NewPermissions()
	require.NoError(t, err)
	s, err := Open(filepath.Join(t.TempDir(), "store"), perms)
	require.NoError(t, err)
	return s
}

func TestPutThenGet(t *testing.T) {
	s := openStore(t)

	_, ok, err := s.Get(contentHash, "payload.txt")
	require.NoError(t, err)
	require.False(t, ok)

	path, err := s.Put(contentHash, "payload.txt", strings.NewReader(content))
	require.NoError(t, err)
	require.Equal(t, filepath.Join(s.Dir(), "sha256", contentHash, "payload.txt"), path)

	got, ok, err := s.Get(strings.ToUpper(contentHash), "payload.txt")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, path, got)

	b, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, content, string(b))
}

func TestPutWrongHashLeavesNothing(t *testing.T) {
	s := openStore(t)
	wrong := strings.Repeat("0", 64)

	_, err := s.Put(wrong, "payload.txt", strings.NewReader(content))
	require.ErrorContains(t, err, "expected sha256 hash "+wrong)

	entries, err := os.ReadDir(filepath.Join(s.Dir(), "sha256", wrong))
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestGetRefusesChangedFile(t *testing.T) {
	s := openStore(t)
	path, err := s.Put(contentHash, "payload.txt", strings.NewReader(content))
	require.NoError(t, err)

	require.NoError(t, os.Chmod(path, 0o644))
	require.NoError(t, os.WriteFile(path, []byte("tampered\n"), 0o644))

	_, _, err = s.Get(contentHash, "payload.txt")
	require.ErrorContains(t, err, "was changed after it was stored")
}

func TestPathRejectsBadInput(t *testing.T) {
	s := openStore(t)
	for _, tc := range []struct{ hash, name, want string }{
		{"abc", "f", "not 64 hex characters"},
		{strings.Repeat("g", 64), "f", "not 64 hex characters"},
		{contentHash, "", "plain file name"},
		{contentHash, "..", "plain file name"},
		{contentHash, "a/b", "plain file name"},
		{contentHash, `a\b`, "plain file name"},
	} {
		_, err := s.Path(tc.hash, tc.name)
		require.ErrorContains(t, err, tc.want, "hash=%q name=%q", tc.hash, tc.name)
	}
}

func TestLazyOpensOnce(t *testing.T) {
	perms, err := NewPermissions()
	require.NoError(t, err)
	dir := filepath.Join(t.TempDir(), "store")
	open := Lazy(dir, perms)

	_, err = os.Stat(dir)
	require.True(t, os.IsNotExist(err), "Lazy must not create the store before it's used")

	a, err := open()
	require.NoError(t, err)
	b, err := open()
	require.NoError(t, err)
	require.Same(t, a, b)
}
