package extract

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"github.com/discentem/starcm/functions/base"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
)

// member is a file in a test archive.
type member struct {
	name string
	data string
	mode os.FileMode
	// typ is a tar type flag, for entries that aren't regular files.
	typ  byte
	link string
}

func writeZip(t *testing.T, path string, ms []member) {
	t.Helper()
	f, err := os.Create(path)
	require.NoError(t, err)
	zw := zip.NewWriter(f)
	for _, m := range ms {
		h := &zip.FileHeader{Name: m.name, Method: zip.Deflate}
		h.SetMode(m.mode)
		if m.typ == tar.TypeSymlink {
			h.SetMode(os.ModeSymlink | 0o777)
		}
		w, err := zw.CreateHeader(h)
		require.NoError(t, err)
		if m.typ == tar.TypeSymlink {
			_, err = w.Write([]byte(m.link))
		} else {
			_, err = w.Write([]byte(m.data))
		}
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	require.NoError(t, f.Close())
}

func writeTarGz(t *testing.T, path string, ms []member) {
	t.Helper()
	f, err := os.Create(path)
	require.NoError(t, err)
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	for _, m := range ms {
		h := &tar.Header{Name: m.name, Mode: int64(m.mode), Typeflag: tar.TypeReg, Size: int64(len(m.data))}
		if m.typ != 0 {
			h.Typeflag, h.Linkname, h.Size = m.typ, m.link, 0
		}
		if m.mode.IsDir() {
			h.Typeflag, h.Mode = tar.TypeDir, int64(m.mode.Perm())
		}
		require.NoError(t, tw.WriteHeader(h))
		if h.Size > 0 {
			_, err = tw.Write([]byte(m.data))
			require.NoError(t, err)
		}
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	require.NoError(t, f.Close())
}

func strList(ss ...string) *starlark.List {
	var vs []starlark.Value
	for _, s := range ss {
		vs = append(vs, starlark.String(s))
	}
	return starlark.NewList(vs)
}

func kw(archive, dest string, members *starlark.List) []starlark.Tuple {
	k := []starlark.Tuple{
		{starlark.String("archive"), starlark.String(archive)},
		{starlark.String("dest"), starlark.String(dest)},
	}
	if members != nil {
		k = append(k, starlark.Tuple{starlark.String("members"), members})
	}
	return k
}

// call runs extract(label = "unpack", ...) with a's configuration.
func call(a *extractAction, args starlark.Tuple, kwargs []starlark.Tuple) (*base.Result, error) {
	kwargs = append(kwargs, starlark.Tuple{starlark.String("label"), starlark.String("unpack")})
	return base.NewModule(context.Background(), "extract", extractArgs, a).Call(nil, args, kwargs)
}

func run(a *extractAction, archive, dest string, members *starlark.List) (changed bool, msg string, paths []string, err error) {
	r, err := call(a, nil, kw(archive, dest, members))
	if err != nil {
		return false, "", nil, err
	}
	l := r.Return.(*starlark.List)
	for i := 0; i < l.Len(); i++ {
		paths = append(paths, string(l.Index(i).(starlark.String)))
	}
	return r.Changed, *r.Message, paths, nil
}

var normal = []member{
	{name: "pkg/", mode: os.ModeDir | 0o755},
	{name: "pkg/tool", data: "#!/bin/sh\necho hi\n", mode: 0o755},
	{name: "pkg/docs/README.txt", data: "read me\n", mode: 0o644},
	{name: "LICENSE", data: "mit\n"},
}

// archives lists the formats under test.
var archives = []struct {
	name  string
	write func(*testing.T, string, []member)
}{
	{"a.zip", writeZip},
	{"a.tar.gz", writeTarGz},
	{"a.TGZ", writeTarGz},
}

func TestExtract(t *testing.T) {
	for _, ar := range archives {
		t.Run(ar.name, func(t *testing.T) {
			dir := t.TempDir()
			archive := filepath.Join(dir, ar.name)
			ar.write(t, archive, normal)
			dest := filepath.Join(dir, "out", "nested")
			a := &extractAction{goos: runtime.GOOS}

			changed, msg, paths, err := run(a, archive, dest, nil)
			require.NoError(t, err)
			assert.True(t, changed)
			assert.Equal(t, fmt.Sprintf("extracted 3 of 3 files to %q", dest), msg)
			assert.Equal(t, []string{
				filepath.Join(dest, "pkg", "tool"),
				filepath.Join(dest, "pkg", "docs", "README.txt"),
				filepath.Join(dest, "LICENSE"),
			}, paths)
			b, err := os.ReadFile(filepath.Join(dest, "pkg", "tool"))
			require.NoError(t, err)
			assert.Equal(t, "#!/bin/sh\necho hi\n", string(b))

			// Second run: nothing to do.
			changed, msg, _, err = run(a, archive, dest, nil)
			require.NoError(t, err)
			assert.False(t, changed)
			assert.Equal(t, fmt.Sprintf("%q already up to date", dest), msg)

			// A changed file is put back.
			require.NoError(t, os.WriteFile(filepath.Join(dest, "LICENSE"), []byte("tampered"), 0o644))
			changed, _, _, err = run(a, archive, dest, nil)
			require.NoError(t, err)
			assert.True(t, changed)
			b, _ = os.ReadFile(filepath.Join(dest, "LICENSE"))
			assert.Equal(t, "mit\n", string(b))
		})
	}
}

func TestExtractMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows has no permission bits")
	}
	for _, ar := range archives {
		t.Run(ar.name, func(t *testing.T) {
			dir := t.TempDir()
			archive := filepath.Join(dir, ar.name)
			ar.write(t, archive, []member{
				{name: "tool", data: "x", mode: 0o755 | os.ModeSetuid},
				{name: "plain", data: "y"},
				{name: "private", data: "z", mode: 0o600},
			})
			dest := filepath.Join(dir, "out")
			a := &extractAction{goos: runtime.GOOS}
			_, _, _, err := run(a, archive, dest, nil)
			require.NoError(t, err)
			modes := map[string]os.FileMode{"tool": 0o755, "plain": 0o644, "private": 0o600}
			for name, want := range modes {
				info, err := os.Lstat(filepath.Join(dest, name))
				require.NoError(t, err)
				assert.Equal(t, want, info.Mode(), name)
			}

			// A file with the right content and the wrong mode is fixed.
			require.NoError(t, os.Chmod(filepath.Join(dest, "tool"), 0o700))
			changed, _, _, err := run(a, archive, dest, nil)
			require.NoError(t, err)
			assert.True(t, changed)
			info, _ := os.Stat(filepath.Join(dest, "tool"))
			assert.Equal(t, os.FileMode(0o755), info.Mode())
		})
	}
}

func TestExtractMembers(t *testing.T) {
	for _, ar := range archives {
		t.Run(ar.name, func(t *testing.T) {
			dir := t.TempDir()
			archive := filepath.Join(dir, ar.name)
			ar.write(t, archive, normal)
			dest := filepath.Join(dir, "out")
			a := &extractAction{goos: runtime.GOOS}

			changed, msg, paths, err := run(a, archive, dest, strList("pkg/tool"))
			require.NoError(t, err)
			assert.True(t, changed)
			assert.Equal(t, fmt.Sprintf("extracted 1 of 3 files to %q", dest), msg)
			assert.Equal(t, []string{filepath.Join(dest, "pkg", "tool")}, paths)
			assert.FileExists(t, filepath.Join(dest, "pkg", "tool"))
			assert.NoFileExists(t, filepath.Join(dest, "LICENSE"))
			assert.NoDirExists(t, filepath.Join(dest, "pkg", "docs"))

			changed, _, _, err = run(a, archive, dest, strList("pkg/tool"))
			require.NoError(t, err)
			assert.False(t, changed)
		})
	}
}

func TestMissingMember(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "a.zip")
	writeZip(t, archive, normal)
	dest := filepath.Join(dir, "out")
	_, _, _, err := run(&extractAction{goos: runtime.GOOS}, archive, dest, strList("pkg/tool", "nope"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), `extract(label="unpack"): member "nope" not found`)
	assert.NoDirExists(t, dest)
}

func TestRefusedEntries(t *testing.T) {
	cases := []struct {
		name    string
		bad     member
		contain string
	}{
		{"traversal", member{name: "../evil", data: "x"}, "refusing entry \"../evil\""},
		{"nested traversal", member{name: "a/../../evil", data: "x"}, "\"..\""},
		{"backslash traversal", member{name: `a\..\..\evil`, data: "x"}, "\"..\""},
		{"absolute", member{name: "/evil", data: "x"}, "absolute path"},
		{"windows drive", member{name: `C:\evil`, data: "x"}, "absolute path"},
		{"unc", member{name: `\\host\evil`, data: "x"}, "absolute path"},
		{"symlink", member{name: "link", typ: tar.TypeSymlink, link: "/etc/passwd"}, "symbolic links"},
	}
	for _, ar := range archives {
		for _, c := range cases {
			t.Run(ar.name+"/"+c.name, func(t *testing.T) {
				dir := t.TempDir()
				archive := filepath.Join(dir, ar.name)
				// The good file comes first: nothing of it may be written.
				ar.write(t, archive, []member{{name: "good.txt", data: "ok"}, c.bad})
				dest := filepath.Join(dir, "out")
				_, _, _, err := run(&extractAction{goos: runtime.GOOS}, archive, dest, nil)
				require.Error(t, err)
				assert.Contains(t, err.Error(), `extract(label="unpack"): `)
				assert.Contains(t, err.Error(), c.contain)
				assert.NoDirExists(t, dest)
				assert.NoFileExists(t, filepath.Join(dir, "evil"))
			})
		}
	}
}

func TestRefusedTarSpecials(t *testing.T) {
	for _, typ := range []byte{tar.TypeLink, tar.TypeChar, tar.TypeBlock, tar.TypeFifo} {
		dir := t.TempDir()
		archive := filepath.Join(dir, "a.tar.gz")
		writeTarGz(t, archive, []member{{name: "good.txt", data: "ok"}, {name: "special", typ: typ, link: "good.txt"}})
		dest := filepath.Join(dir, "out")
		_, _, _, err := run(&extractAction{goos: runtime.GOOS}, archive, dest, nil)
		require.Error(t, err, "type %c", typ)
		assert.Contains(t, err.Error(), `refusing entry "special"`)
		assert.NoDirExists(t, dest)
	}
}

func TestSymlinkInsideDest(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(dir, "outside")
	dest := filepath.Join(dir, "dest")
	require.NoError(t, os.MkdirAll(outside, 0o755))
	require.NoError(t, os.MkdirAll(dest, 0o755))
	if err := os.Symlink(outside, filepath.Join(dest, "pkg")); err != nil {
		t.Skipf("can't create symlinks: %v", err)
	}
	for _, ar := range archives {
		archive := filepath.Join(dir, ar.name)
		ar.write(t, archive, normal)
		_, _, _, err := run(&extractAction{goos: runtime.GOOS}, archive, dest, nil)
		require.Error(t, err, ar.name)
		entries, _ := os.ReadDir(outside)
		assert.Empty(t, entries, "wrote through the symlink")
	}
}

func TestSymlinkInPlaceOfFile(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(dir, "target.txt")
	require.NoError(t, os.WriteFile(outside, []byte("keep"), 0o644))
	dest := filepath.Join(dir, "dest")
	require.NoError(t, os.MkdirAll(dest, 0o755))
	if err := os.Symlink(outside, filepath.Join(dest, "LICENSE")); err != nil {
		t.Skipf("can't create symlinks: %v", err)
	}
	archive := filepath.Join(dir, "a.zip")
	writeZip(t, archive, normal)
	_, _, _, err := run(&extractAction{goos: runtime.GOOS}, archive, dest, nil)
	require.NoError(t, err)
	b, _ := os.ReadFile(outside)
	assert.Equal(t, "keep", string(b))
	info, err := os.Lstat(filepath.Join(dest, "LICENSE"))
	require.NoError(t, err)
	assert.True(t, info.Mode().IsRegular())
}

func TestUnknownFormat(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "a.rar")
	require.NoError(t, os.WriteFile(archive, []byte("x"), 0o644))
	dest := filepath.Join(dir, "out")
	_, _, _, err := run(&extractAction{goos: runtime.GOOS}, archive, dest, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown archive format")
	assert.NoDirExists(t, dest)
}

func TestStoreCheck(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "a.zip")
	writeZip(t, archive, normal)
	dest := filepath.Join(dir, "out")
	var checked string
	a := &extractAction{goos: runtime.GOOS, checkStored: func(p string) error {
		checked = p
		return errors.New("it was changed after it was stored")
	}}
	_, _, _, err := run(a, archive, dest, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `extract(label="unpack"): it was changed after it was stored`)
	assert.Equal(t, archive, checked)
	assert.NoDirExists(t, dest)
}

func TestPositionalArchive(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "a.zip")
	writeZip(t, archive, normal)
	dest := filepath.Join(dir, "out")
	a := &extractAction{goos: runtime.GOOS}
	r, err := call(a, starlark.Tuple{starlark.String(archive), starlark.String(dest)}, nil)
	require.NoError(t, err)
	assert.True(t, r.Changed)
}

func TestRelativeAndHomeDest(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "a.zip")
	writeZip(t, archive, normal)
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	_, _, paths, err := run(&extractAction{goos: runtime.GOOS}, archive, "~/home-out", nil)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "home-out", "pkg", "tool"), paths[0])

	_, _, _, err = run(&extractAction{goos: runtime.GOOS}, archive, "relative-out", nil)
	assert.ErrorContains(t, err, `extract(label="unpack"): dest: "relative-out" is not an absolute path`)
}
