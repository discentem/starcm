package glob

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tree creates files (slash-separated, relative to a new temporary
// directory) and returns that directory.
func tree(t *testing.T, files ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, f := range files {
		p := filepath.Join(dir, filepath.FromSlash(f))
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, nil, 0o644))
	}
	return dir
}

// rel returns matches relative to dir, slash-separated, for comparing.
func rel(t *testing.T, dir string, matches []string) []string {
	t.Helper()
	out := []string{}
	for _, m := range matches {
		r, err := filepath.Rel(dir, m)
		require.NoError(t, err)
		out = append(out, filepath.ToSlash(r))
	}
	return out
}

func TestGlob(t *testing.T) {
	dir := tree(t,
		"third_party/a/go.mod",
		"third_party/b/go.mod",
		"third_party/c/README",
		"third_party/notes.txt",
		"third_party/.hidden/go.mod",
		"lib/x1.star",
		"lib/x2.star",
		"lib/y.star",
	)
	fsys := afero.NewOsFs()
	tests := []struct {
		pattern string
		want    []string
	}{
		{"third_party/*/go.mod", []string{"third_party/.hidden/go.mod", "third_party/a/go.mod", "third_party/b/go.mod"}},
		{"third_party/[ab]/go.mod", []string{"third_party/a/go.mod", "third_party/b/go.mod"}},
		{"third_party/*", []string{"third_party/.hidden", "third_party/a", "third_party/b", "third_party/c", "third_party/notes.txt"}},
		{"*/*/README", []string{"third_party/c/README"}},
		{"lib/x?.star", []string{"lib/x1.star", "lib/x2.star"}},
		{"lib/[^x]*.star", []string{"lib/y.star"}},
		// No wildcards: the path itself, if it exists.
		{"lib/y.star", []string{"lib/y.star"}},
		{"lib", []string{"lib"}},
		// Nothing matches.
		{"lib/*.txt", []string{}},
		{"missing/*/go.mod", []string{}},
		{"missing", []string{}},
		// A file where the pattern needs a directory matches nothing.
		{"third_party/notes.txt/*", []string{}},
		{"lib/y.star/x", []string{}},
		// The pattern is cleaned first.
		{"lib/../lib//y.star", []string{"lib/y.star"}},
	}
	for _, tc := range tests {
		t.Run(tc.pattern, func(t *testing.T) {
			got, err := Glob(fsys, filepath.Join(dir, filepath.FromSlash(tc.pattern)))
			require.NoError(t, err)
			assert.Equal(t, tc.want, rel(t, dir, got))
		})
	}
}

func TestGlobRoot(t *testing.T) {
	root := string(filepath.Separator)
	if runtime.GOOS == "windows" {
		root = filepath.VolumeName(os.TempDir()) + root
	}
	got, err := Glob(afero.NewOsFs(), root)
	require.NoError(t, err)
	assert.Equal(t, []string{root}, got)
}

func TestGlobEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip(`\ is the separator on Windows, so there are no escapes`)
	}
	dir := tree(t, "a*b/x", "aXb/x")
	got, err := Glob(afero.NewOsFs(), filepath.Join(dir, `a\*b`, "x"))
	require.NoError(t, err)
	assert.Equal(t, []string{"a*b/x"}, rel(t, dir, got))
}

func TestGlobErrors(t *testing.T) {
	fsys := afero.NewOsFs()
	dir := tree(t, "a/x")
	bad := []string{"[", "a/[", "*/[", "*[", "a*[", "a/[]", "[a-"}
	if runtime.GOOS != "windows" {
		// A trailing escape; on Windows \ is a separator instead.
		bad = append(bad, `a\`)
	}
	for _, p := range bad {
		_, err := Glob(fsys, filepath.Join(dir, p))
		assert.ErrorContains(t, err, "invalid pattern", p)
		assert.ErrorIs(t, err, filepath.ErrBadPattern, p)
	}
	_, err := Glob(fsys, "a/*")
	assert.EqualError(t, err, `pattern "a/*" is not absolute`)
}

// A directory starcm can't read is an error, not an empty match.
func TestGlobUnreadable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permissions work differently on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root can read any directory")
	}
	dir := tree(t, "locked/a/go.mod")
	locked := filepath.Join(dir, "locked")
	require.NoError(t, os.Chmod(locked, 0o000))
	t.Cleanup(func() { os.Chmod(locked, 0o755) })
	_, err := Glob(afero.NewOsFs(), filepath.Join(locked, "*", "go.mod"))
	assert.ErrorIs(t, err, os.ErrPermission)
}
