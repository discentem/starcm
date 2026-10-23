package download

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	localstore "github.com/discentem/starcm/libraries/local_store"
	starcmsha256 "github.com/discentem/starcm/libraries/sha256"
	"github.com/discentem/starcm/libraries/signing"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func sumOf(b []byte) string {
	h, _ := starcmsha256.FromReader(strings.NewReader(string(b)))
	return h
}

// folderServer serves files under /cfg/ with a manifest listing them, and
// counts requests per path.
type folderServer struct {
	*httptest.Server
	mu       sync.Mutex
	content  map[string][]byte
	requests map[string]int
	manifest []byte
}

func newFolderServer(t *testing.T, files map[string]string) *folderServer {
	t.Helper()
	fs := &folderServer{content: map[string][]byte{}, requests: map[string]int{}}
	m := signing.Manifest{Version: signing.ManifestVersion, Files: map[string]string{}}
	for p, c := range files {
		fs.content["/cfg/"+p] = []byte(c)
		m.Files[p] = sumOf([]byte(c))
	}
	fs.manifest = m.Bytes()
	fs.content["/cfg/"+signing.ManifestName] = fs.manifest
	fs.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fs.mu.Lock()
		defer fs.mu.Unlock()
		fs.requests[r.URL.Path]++
		b, ok := fs.content[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Write(b)
	}))
	t.Cleanup(fs.Close)
	return fs
}

func (fs *folderServer) set(path string, b []byte) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.content[path] = b
}

func (fs *folderServer) count(path string) int {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return fs.requests[path]
}

var testFolder = map[string]string{
	"main.star":            "print('main')\n",
	"templates/motd.jinja": "hello {{ name }}\n",
	"a b/c%d.txt":          "escaped\n",
}

func storeFetcher(t *testing.T, opts ...Option) *Fetcher {
	t.Helper()
	perms, err := localstore.NewPermissions()
	require.NoError(t, err)
	opts = append([]Option{WithStore(localstore.Lazy(filepath.Join(t.TempDir(), "store"), perms))}, opts...)
	return NewFetcher(afero.NewOsFs(), nil, opts...)
}

func TestFetchFolderIntoStore(t *testing.T) {
	srv := newFolderServer(t, testFolder)
	srv.content["/cfg/unlisted.star"] = []byte("never fetched")
	f := storeFetcher(t)
	mh := sumOf(srv.manifest)
	req := Request{Label: "cfg", SHA256: mh}

	got, err := f.FetchURL(context.Background(), srv.URL+"/cfg/", req)
	require.NoError(t, err)
	require.True(t, got.Changed)
	require.True(t, strings.HasSuffix(got.Path, filepath.Join("folders", mh)), got.Path)
	require.Contains(t, got.Message, "downloaded 3 of 3 files to ")
	require.Len(t, got.Manifest.Files, 3)
	for p, c := range testFolder {
		b, err := os.ReadFile(filepath.Join(got.Path, filepath.FromSlash(p)))
		require.NoError(t, err)
		require.Equal(t, c, string(b))
	}
	require.Equal(t, 1, srv.count("/cfg/a b/c%d.txt"))
	require.Equal(t, 0, srv.count("/cfg/unlisted.star"))

	// The second fetch re-checks the stored folder and fetches nothing.
	again, err := f.FetchURL(context.Background(), srv.URL+"/cfg/", req)
	require.NoError(t, err)
	require.False(t, again.Changed)
	require.Equal(t, got.Path, again.Path)
	require.Contains(t, again.Message, "already in the store and sha256 verified")
	require.Equal(t, 1, srv.count("/cfg/"+signing.ManifestName))
}

// A new release only fetches the files whose hashes changed.
func TestFetchFolderNewRelease(t *testing.T) {
	srv := newFolderServer(t, testFolder)
	f := storeFetcher(t)
	_, err := f.FetchURL(context.Background(), srv.URL+"/cfg/", Request{SHA256: sumOf(srv.manifest)})
	require.NoError(t, err)

	files := map[string]string{}
	for k, v := range testFolder {
		files[k] = v
	}
	files["main.star"] = "print('v2')\n"
	v2 := newFolderServer(t, files)
	got, err := f.FetchURL(context.Background(), v2.URL+"/cfg/", Request{SHA256: sumOf(v2.manifest)})
	require.NoError(t, err)
	require.Contains(t, got.Message, "downloaded 1 of 3 files")
	require.Equal(t, 0, v2.count("/cfg/templates/motd.jinja"))
}

func TestFetchFolderRefuses(t *testing.T) {
	srv := newFolderServer(t, testFolder)
	f := storeFetcher(t)
	mh := sumOf(srv.manifest)

	_, err := f.FetchURL(context.Background(), srv.URL+"/cfg/", Request{SHA256: strings.Repeat("0", 64)})
	require.ErrorContains(t, err, "starcm-manifest.json: expected sha256 hash "+strings.Repeat("0", 64))

	_, err = f.FetchURL(context.Background(), srv.URL+"/cfg/", Request{Label: "x", SHA256: mh, FileName: "a"})
	require.ErrorContains(t, err, "file_name doesn't apply to a folder")

	srv.set("/cfg/templates/motd.jinja", []byte("tampered\n"))
	_, err = f.FetchURL(context.Background(), srv.URL+"/cfg/", Request{SHA256: mh})
	require.ErrorContains(t, err, "templates/motd.jinja: expected sha256 hash")

	delete(srv.content, "/cfg/templates/motd.jinja")
	_, err = f.FetchURL(context.Background(), srv.URL+"/cfg/", Request{SHA256: mh})
	require.ErrorContains(t, err, "404")

	_, err = f.FetchFolder(context.Background(), Source{URL: srv.URL + "/cfg"}, Request{SHA256: mh})
	require.ErrorContains(t, err, "must end in /")

	bad := []byte(`{"version": 1, "files": {"../escape": "` + strings.Repeat("a", 64) + `"}}`)
	srv.set("/cfg/"+signing.ManifestName, bad)
	_, err = f.FetchURL(context.Background(), srv.URL+"/cfg/", Request{SHA256: sumOf(bad)})
	require.ErrorContains(t, err, "not a clean path")
}

func TestFetchFolderSignature(t *testing.T) {
	keyring, sign := testKey(t)
	srv := newFolderServer(t, testFolder)
	srv.content["/cfg/"+signing.ManifestName+".sig"] = sign(srv.manifest)
	f := storeFetcher(t, WithKeyring(keyring))

	got, err := f.FetchURL(context.Background(), srv.URL+"/cfg/", Request{Signature: srv.URL + "/cfg/" + signing.ManifestName + ".sig"})
	require.NoError(t, err)
	sig, err := os.ReadFile(filepath.Join(got.Path, signing.ManifestName+".sig"))
	require.NoError(t, err)
	require.Equal(t, srv.content["/cfg/"+signing.ManifestName+".sig"], sig)

	// A signature for different content is refused before anything else is
	// fetched.
	srv.set("/cfg/"+signing.ManifestName+".sig", sign([]byte("other")))
	srv.set("/cfg/"+signing.ManifestName, []byte("other"))
	_, err = f.FetchURL(context.Background(), srv.URL+"/cfg/", Request{Signature: srv.URL + "/cfg/" + signing.ManifestName + ".sig"})
	require.ErrorContains(t, err, "malformed manifest")
}

func TestFetchFolderSaveTo(t *testing.T) {
	srv := newFolderServer(t, testFolder)
	fsys := afero.NewMemMapFs()
	f := NewFetcher(fsys, nil)
	mh := sumOf(srv.manifest)
	dest := abs("managed/cfg")
	require.NoError(t, afero.WriteFile(fsys, filepath.Join(dest, "local.txt"), []byte("mine"), 0o644))

	got, err := f.FetchURL(context.Background(), srv.URL+"/cfg/", Request{SHA256: mh, SaveTo: dest})
	require.NoError(t, err)
	require.True(t, got.Changed)
	require.Equal(t, dest, got.Path)
	require.Equal(t, "downloaded 3 of 3 files to "+dest, got.Message)
	for p, c := range testFolder {
		b, err := afero.ReadFile(fsys, filepath.Join(dest, filepath.FromSlash(p)))
		require.NoError(t, err)
		require.Equal(t, c, string(b))
	}
	b, err := afero.ReadFile(fsys, filepath.Join(dest, signing.ManifestName))
	require.NoError(t, err)
	require.Equal(t, srv.manifest, b)
	b, err = afero.ReadFile(fsys, filepath.Join(dest, "local.txt"))
	require.NoError(t, err)
	require.Equal(t, "mine", string(b), "files the manifest doesn't list are left alone")

	again, err := f.FetchURL(context.Background(), srv.URL+"/cfg/", Request{SHA256: mh, SaveTo: dest})
	require.NoError(t, err)
	require.False(t, again.Changed)
	require.Equal(t, 1, srv.count("/cfg/main.star"))

	// A changed file is fetched again; the others aren't.
	require.NoError(t, afero.WriteFile(fsys, filepath.Join(dest, "main.star"), []byte("edited"), 0o644))
	fixed, err := f.FetchURL(context.Background(), srv.URL+"/cfg/", Request{SHA256: mh, SaveTo: dest})
	require.NoError(t, err)
	require.True(t, fixed.Changed)
	require.Equal(t, "downloaded 1 of 3 files to "+dest, fixed.Message)

	_, err = f.FetchURL(context.Background(), srv.URL+"/cfg/", Request{SHA256: mh, SaveTo: "relative"})
	require.ErrorContains(t, err, "not an absolute path")
	_, err = NewFetcher(fsys, nil, StoreOnly(true)).FetchURL(context.Background(), srv.URL+"/cfg/", Request{SHA256: mh, SaveTo: dest})
	require.ErrorContains(t, err, "--store-only")
}

func TestFetchFolderS3(t *testing.T) {
	awsEnv(t)
	keyring, sign := testKey(t)
	m := signing.Manifest{Version: signing.ManifestVersion, Files: map[string]string{}}
	objects := map[string]string{}
	for p, c := range testFolder {
		m.Files[p] = sumOf([]byte(c))
		objects["/config/agent/"+p] = c
	}
	manifest := m.Bytes()
	objects["/config/agent/"+signing.ManifestName] = string(manifest)
	objects["/config/agent/"+signing.ManifestName+".sig"] = string(sign(manifest))
	objects["/config/agent/unlisted.star"] = "never fetched"
	s3 := &fakeS3{objects: objects}
	srv := httptest.NewServer(s3)
	t.Cleanup(srv.Close)

	f := storeFetcher(t, WithKeyring(keyring))
	obj := S3Object{Bucket: "config", Key: "agent/", Endpoint: srv.URL}
	got, err := f.FetchS3(context.Background(), obj, Request{Signature: "agent/" + signing.ManifestName + ".sig"})
	require.NoError(t, err)
	require.True(t, got.Changed)
	b, err := os.ReadFile(filepath.Join(got.Path, "templates", "motd.jinja"))
	require.NoError(t, err)
	require.Equal(t, testFolder["templates/motd.jinja"], string(b))
	for _, p := range s3.paths {
		require.NotEqual(t, "/config/agent/unlisted.star", p)
	}
}
