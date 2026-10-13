package sha256

import (
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// echo -n hello | shasum -a 256
const helloSHA = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"

func TestFileAndFromReaderAgree(t *testing.T) {
	fsys := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fsys, "f", []byte("hello"), 0o644))

	got, err := File(fsys, "f")
	require.NoError(t, err)
	require.Equal(t, helloSHA, got)

	got, err = FromReader(strings.NewReader("hello"))
	require.NoError(t, err)
	require.Equal(t, helloSHA, got)

	h := New()
	h.Write([]byte("hel"))
	h.Write([]byte("lo"))
	require.Equal(t, helloSHA, Hex(h))

	_, err = File(fsys, "missing")
	require.Error(t, err)
}
