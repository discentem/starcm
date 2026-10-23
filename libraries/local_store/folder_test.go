package localstore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	starcmsha256 "github.com/discentem/starcm/libraries/sha256"
	"github.com/discentem/starcm/libraries/signing"
	"github.com/stretchr/testify/require"
)

func hashOf(s string) string {
	h, err := starcmsha256.FromReader(strings.NewReader(s))
	if err != nil {
		panic(err)
	}
	return h
}

// putFolder stores files and assembles the folder for their manifest.
func putFolder(t *testing.T, s *Store, files map[string]string, sig []byte) (Folder, []byte, string) {
	t.Helper()
	m := signing.Manifest{Version: signing.ManifestVersion, Files: map[string]string{}}
	for p, content := range files {
		m.Files[p] = hashOf(content)
		_, err := s.Put(m.Files[p], FolderFileName(p), strings.NewReader(content))
		require.NoError(t, err)
	}
	manifest := m.Bytes()
	mh := hashOf(string(manifest))
	f, changed, err := s.PutFolder(mh, manifest, sig)
	require.NoError(t, err)
	require.True(t, changed)
	return f, manifest, mh
}

var folderFiles = map[string]string{
	"main.star":            "main\n",
	"templates/motd.jinja": "motd\n",
	"lib/same.txt":         "same\n",
	"other/same.txt":       "same\n",
}

func TestPutFolderThenGetFolder(t *testing.T) {
	s := openStore(t)
	f, manifest, mh := putFolder(t, s, folderFiles, []byte("sig v1\n"))
	require.Equal(t, filepath.Join(s.Dir(), "folders", mh), f.Dir)
	for p, content := range folderFiles {
		b, err := os.ReadFile(filepath.Join(f.Dir, filepath.FromSlash(p)))
		require.NoError(t, err)
		require.Equal(t, content, string(b))
	}
	b, err := os.ReadFile(filepath.Join(f.Dir, signing.ManifestName))
	require.NoError(t, err)
	require.Equal(t, manifest, b)

	got, ok, err := s.GetFolder(strings.ToUpper(mh))
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, f, got)

	// Again: nothing to do, except a new signature after re-signing.
	_, changed, err := s.PutFolder(mh, manifest, []byte("sig v1\n"))
	require.NoError(t, err)
	require.False(t, changed)
	_, changed, err = s.PutFolder(mh, manifest, []byte("sig v2\n"))
	require.NoError(t, err)
	require.True(t, changed)
	b, err = os.ReadFile(filepath.Join(f.Dir, signing.ManifestName+signing.SignatureSuffix))
	require.NoError(t, err)
	require.Equal(t, "sig v2\n", string(b))

	// Nothing left behind from assembling it.
	entries, err := os.ReadDir(filepath.Join(s.Dir(), "folders"))
	require.NoError(t, err)
	require.Len(t, entries, 1)
}

func TestGetFolderMissing(t *testing.T) {
	s := openStore(t)
	_, ok, err := s.GetFolder(contentHash)
	require.NoError(t, err)
	require.False(t, ok)
	_, _, err = s.GetFolder("nope")
	require.ErrorContains(t, err, "not 64 hex characters")
}

func TestPutFolderNeedsStoredFiles(t *testing.T) {
	s := openStore(t)
	m := signing.Manifest{Version: 1, Files: map[string]string{"a.txt": hashOf("a")}}
	manifest := m.Bytes()
	_, _, err := s.PutFolder(hashOf(string(manifest)), manifest, nil)
	require.ErrorContains(t, err, "a.txt (sha256 "+hashOf("a")+") is not in the store")
	_, _, err = s.PutFolder(contentHash, manifest, nil)
	require.ErrorContains(t, err, "expected manifest sha256 hash "+contentHash)
	_, err = os.Stat(filepath.Join(s.Dir(), "folders", hashOf(string(manifest))))
	require.True(t, os.IsNotExist(err))
}

// writable makes a stored file writable again, so a test can tamper with it.
func writable(t *testing.T, p string) {
	t.Helper()
	require.NoError(t, os.Chmod(p, 0o644))
}

func TestGetFolderRefusesChanges(t *testing.T) {
	for name, tamper := range map[string]func(t *testing.T, dir string){
		"changed file": func(t *testing.T, dir string) {
			p := filepath.Join(dir, "templates", "motd.jinja")
			writable(t, p)
			// A hard link changes the stored copy too; replace it instead.
			require.NoError(t, os.Remove(p))
			require.NoError(t, os.WriteFile(p, []byte("evil\n"), 0o644))
		},
		"extra file": func(t *testing.T, dir string) {
			require.NoError(t, os.WriteFile(filepath.Join(dir, "extra.star"), []byte("x"), 0o644))
		},
		"missing file": func(t *testing.T, dir string) {
			p := filepath.Join(dir, "main.star")
			writable(t, p)
			require.NoError(t, os.Remove(p))
		},
		"changed manifest": func(t *testing.T, dir string) {
			p := filepath.Join(dir, signing.ManifestName)
			writable(t, p)
			require.NoError(t, os.Remove(p))
			require.NoError(t, os.WriteFile(p, []byte(`{"version": 1, "files": {}}`), 0o644))
		},
	} {
		t.Run(name, func(t *testing.T) {
			s := openStore(t)
			f, _, mh := putFolder(t, s, folderFiles, nil)
			tamper(t, f.Dir)
			_, _, err := s.GetFolder(mh)
			require.Error(t, err)
			require.Contains(t, err.Error(), "changed after it was stored")
		})
	}
}

func TestInFolderAndCheckIfStored(t *testing.T) {
	s := openStore(t)
	f, _, mh := putFolder(t, s, folderFiles, nil)

	root, hash, rel, ok := InFolder(s.Dir(), filepath.Join(f.Dir, "templates", "motd.jinja"))
	require.True(t, ok)
	require.Equal(t, f.Dir, root)
	require.Equal(t, mh, hash)
	require.Equal(t, "templates/motd.jinja", rel)

	_, _, rel, ok = InFolder(s.Dir(), f.Dir)
	require.True(t, ok)
	require.Equal(t, ".", rel)

	_, _, _, ok = InFolder(s.Dir(), filepath.Join(s.Dir(), "sha256", contentHash, "x"))
	require.False(t, ok)

	check := CheckIfStored(s.Dir(), func() (*Store, error) { return s, nil })
	require.NoError(t, check(filepath.Join(f.Dir, "main.star")))
	require.NoError(t, check(filepath.Join(f.Dir, signing.ManifestName)))
	require.ErrorContains(t, check(filepath.Join(f.Dir, "nope.star")), "isn't a stored file")
	require.ErrorContains(t, check(filepath.Join(s.Dir(), "folders", contentHash, "main.star")), "is not in the store")
}
