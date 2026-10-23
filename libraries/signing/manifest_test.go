package signing

import (
	"crypto/x509"
	"encoding/pem"
	"path/filepath"
	"strings"
	"testing"

	starcmsha256 "github.com/discentem/starcm/libraries/sha256"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func sum(s string) string {
	h, _ := starcmsha256.FromReader(strings.NewReader(s))
	return h
}

func writeFiles(t *testing.T, fsys afero.Fs, dir string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		require.NoError(t, afero.WriteFile(fsys, filepath.Join(dir, filepath.FromSlash(name)), []byte(content), 0o644))
	}
}

func TestManifestOf(t *testing.T) {
	fsys := afero.NewMemMapFs()
	writeFiles(t, fsys, "cfg", map[string]string{
		"main.star":                "main",
		"templates/motd.jinja":     "motd",
		".hidden":                  "skipped",
		".git/config":              "skipped",
		"templates/.DS_Store":      "skipped",
		ManifestName:               "old manifest",
		ManifestName + ".sig":      "old sig",
		"sub/" + ManifestName:      "nested manifests are files",
		"templates/motd.jinja.sig": "a per-file signature is just a file",
	})
	m, err := ManifestOf(fsys, "cfg")
	require.NoError(t, err)
	require.Equal(t, map[string]string{
		"main.star":                sum("main"),
		"templates/motd.jinja":     sum("motd"),
		"sub/" + ManifestName:      sum("nested manifests are files"),
		"templates/motd.jinja.sig": sum("a per-file signature is just a file"),
	}, m.Files)
	require.NoError(t, m.Check(fsys, "cfg"))

	parsed, err := ParseManifest(m.Bytes())
	require.NoError(t, err)
	require.Equal(t, m, parsed)
	require.True(t, strings.HasPrefix(string(m.Bytes()), "{\n  \"version\": 1,\n  \"files\": {\n"), string(m.Bytes()))

	writeFiles(t, fsys, "cfg", map[string]string{"main.star": "changed"})
	require.ErrorContains(t, m.Check(fsys, "cfg"), "main.star has sha256 "+sum("changed")+", but the manifest lists "+sum("main"))
}

func TestParseManifestRefuses(t *testing.T) {
	h := sum("x")
	for _, c := range []struct{ json, want string }{
		{`{"version": 2, "files": {}}`, "manifest version 2 is not supported"},
		{`{"version": 1}`, `no "files"`},
		{`{"version": 1, "files": {}, "extra": 1}`, "unknown field"},
		{`{"version": 1, "files": {}} {}`, "data after the JSON object"},
		{`{"version": 1, "files": {"a": "ABC"}}`, "not 64 lowercase hex"},
		{`{"version": 1, "files": {"../a": "` + h + `"}}`, "not a clean path"},
		{`{"version": 1, "files": {"a/./b": "` + h + `"}}`, "not a clean path"},
		{`{"version": 1, "files": {"a//b": "` + h + `"}}`, "not a clean path"},
		{`{"version": 1, "files": {"/etc/passwd": "` + h + `"}}`, "is absolute"},
		{`{"version": 1, "files": {"a\\b": "` + h + `"}}`, `can't contain \`},
		{`{"version": 1, "files": {"C:x": "` + h + `"}}`, `can't contain \`},
		{`{"version": 1, "files": {"` + ManifestName + `": "` + h + `"}}`, "can't list itself"},
		{`{"version": 1, "files": {"A.txt": "` + h + `", "a.txt": "` + h + `"}}`, "differ only in case"},
		{`{"version": 1, "files": {"a": "` + h + `", "a/b": "` + h + `"}}`, "as a file and as the directory of"},
	} {
		_, err := ParseManifest([]byte(c.json))
		require.ErrorContains(t, err, c.want, c.json)
	}
}

func TestManifestOfRefusesCaseCollisions(t *testing.T) {
	fsys := afero.NewMemMapFs() // case sensitive
	writeFiles(t, fsys, "cfg", map[string]string{"A.txt": "a", "a.txt": "b"})
	_, err := ManifestOf(fsys, "cfg")
	require.ErrorContains(t, err, "differ only in case")
}

func TestSignDir(t *testing.T) {
	pubPEM, priv := newKey(t)
	fsys := afero.NewMemMapFs()
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	require.NoError(t, err)
	require.NoError(t, afero.WriteFile(fsys, "keys/release.pem", pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0o600))
	writeFiles(t, fsys, "cfg", map[string]string{"main.star": "main", "lib/x.star": "x"})

	m, sig, changed, err := SignDir(fsys, "keys/release.pem", "", "", "cfg", later)
	require.NoError(t, err)
	require.True(t, changed)
	require.Len(t, m.Files, 2)
	require.Equal(t, "release", sig.KeyID)

	manifest, err := afero.ReadFile(fsys, "cfg/"+ManifestName)
	require.NoError(t, err)
	require.Equal(t, m.Bytes(), manifest)
	sigFile, err := afero.ReadFile(fsys, "cfg/"+ManifestName+SignatureSuffix)
	require.NoError(t, err)

	k, err := keyring(t, map[string][]byte{"release.pub.pem": pubPEM})
	require.NoError(t, err)
	_, err = k.Verify(manifest, sigFile)
	require.NoError(t, err)

	// Signing again with nothing changed writes nothing; the manifest and its
	// signature aren't listed in the new manifest.
	m2, _, changed, err := SignDir(fsys, "keys/release.pem", "", "", "cfg", later)
	require.NoError(t, err)
	require.False(t, changed)
	require.Equal(t, m, m2)
}
