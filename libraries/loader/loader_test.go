package loading

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	localstore "github.com/discentem/starcm/libraries/local_store"
	starcmsha256 "github.com/discentem/starcm/libraries/sha256"
	starcmshelllib "github.com/discentem/starcm/libraries/shell"
	"github.com/discentem/starcm/libraries/signing"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
)

func sumOf(b []byte) string {
	h, _ := starcmsha256.FromReader(strings.NewReader(string(b)))
	return h
}

// folder is a published folder: its files, and the manifest listing them.
type folder struct {
	files    map[string]string
	manifest []byte
}

func newFolder(files map[string]string) folder {
	m := signing.Manifest{Version: signing.ManifestVersion, Files: map[string]string{}}
	for p, c := range files {
		m.Files[p] = sumOf([]byte(c))
	}
	return folder{files: files, manifest: m.Bytes()}
}

// serve serves f under /cfg/, plus extra paths.
func serve(t *testing.T, f folder, extra map[string][]byte) *httptest.Server {
	t.Helper()
	content := map[string][]byte{"/cfg/" + signing.ManifestName: f.manifest}
	for p, c := range f.files {
		content["/cfg/"+p] = []byte(c)
	}
	for p, c := range extra {
		content[p] = c
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, ok := content[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Write(b)
	}))
	t.Cleanup(srv.Close)
	return srv
}

type env struct {
	dir      string // holds the local .star files
	storeDir string
	keyring  *signing.Keyring
	sign     func([]byte) []byte
}

func newEnv(t *testing.T) *env {
	t.Helper()
	tmp := t.TempDir()
	if real, err := filepath.EvalSymlinks(tmp); err == nil {
		tmp = real
	}
	e := &env{dir: filepath.Join(tmp, "repo"), storeDir: filepath.Join(tmp, "store")}
	require.NoError(t, os.MkdirAll(e.dir, 0o755))

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalPKIXPublicKey(pub)
	require.NoError(t, err)
	keys := filepath.Join(tmp, "trusted")
	require.NoError(t, os.MkdirAll(keys, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(keys, "release.pub.pem"), pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}), 0o644))
	e.keyring, err = signing.LoadKeyring(afero.NewOsFs(), keys)
	require.NoError(t, err)
	e.sign = func(data []byte) []byte {
		sig, err := signing.Sign(priv, "release", "", time.Now().Add(time.Hour), data)
		require.NoError(t, err)
		return sig.Bytes()
	}
	return e
}

// run writes src as main.star in e.dir and runs it the way main.go does,
// returning its globals.
func (e *env) run(t *testing.T, src string, trusted bool) (starlark.StringDict, error) {
	t.Helper()
	root := filepath.Join(e.dir, "main.star")
	require.NoError(t, os.WriteFile(root, []byte(src), 0o644))
	perms, err := localstore.NewPermissions()
	require.NoError(t, err)
	open := localstore.Lazy(e.storeDir, perms)
	opts := []LoaderOption{
		WithStore(open),
		WithStoreDir(e.storeDir),
		WithStoreCheck(localstore.CheckIfStored(e.storeDir, open)),
	}
	if trusted {
		require.NoError(t, os.WriteFile(root+".sig", e.sign([]byte(src)), 0o644))
		opts = append(opts, WithKeyring(e.keyring))
	}
	fsys := afero.NewOsFs()
	l := Default(context.Background(), fsys, &starcmshelllib.RealExecutor{}, e.dir, opts...)
	load := l.Sequential(context.Background())
	return load(&starlark.Thread{}, filepath.ToSlash(root))
}

var agent = newFolder(map[string]string{
	"main.star": `load("starcm", "here", "load_dynamic", "read_file")
load("lib/settings.star", "SETTINGS")
load("lib/settings.star", SAME = "SETTINGS")

CHANNEL = SETTINGS["channel"]
SAME_CHANNEL = SAME["channel"]
MOTD = read_file(here("templates", "motd.txt"), label = "motd").value
ROOT = here()
DYNAMIC = load_dynamic("lib/settings.star", label = "dynamic").value["SETTINGS"]["channel"]
`,
	"lib/settings.star": `SETTINGS = {"channel": "stable"}
`,
	"templates/motd.txt": "welcome\n",
	"escape.star": `load("../../outside.star", "X")
`,
	"escape_resource.star": `load("starcm", "load_dynamic")
load_dynamic("../../outside.star", label = "outside")
`,
	"escape_here.star": `load("starcm", "here")
P = here("..", "..", "x")
`,
})

// A local lib/settings.star, which the folder's relative load() must not
// pick up.
const localSettings = `SETTINGS = {"channel": "local"}`

func loadAgent(srvURL, entry, pin string) string {
	return `load("starcm", "load_remote")
r = load_remote("` + srvURL + `/cfg/", entry = "` + entry + `", ` + pin + `, label = "agent")
CHANGED = r.changed
G = r.value
`
}

func TestLoadRemoteFolder(t *testing.T) {
	e := newEnv(t)
	require.NoError(t, os.MkdirAll(filepath.Join(e.dir, "lib"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(e.dir, "lib", "settings.star"), []byte(localSettings), 0o644))
	srv := serve(t, agent, nil)
	pin := `sha256 = "` + sumOf(agent.manifest) + `"`

	g, err := e.run(t, loadAgent(srv.URL, "main.star", pin), false)
	require.NoError(t, err)
	require.Equal(t, starlark.True, g["CHANGED"])
	mod := g["G"].(*starlark.Dict)
	get := func(k string) starlark.Value {
		v, ok, err := mod.Get(starlark.String(k))
		require.NoError(t, err)
		require.True(t, ok, k)
		return v
	}
	require.Equal(t, starlark.String("welcome\n"), get("MOTD"))
	root := string(get("ROOT").(starlark.String))
	require.Equal(t, filepath.Join(e.storeDir, "folders", sumOf(agent.manifest)), root)
	require.Equal(t, starlark.String("stable"), get("CHANNEL"))
	require.Equal(t, starlark.String("stable"), get("SAME_CHANNEL"))

	// Relative resource paths resolve inside the folder.
	require.Equal(t, starlark.String("stable"), get("DYNAMIC"))

	// Again: everything is in the store.
	g, err = e.run(t, loadAgent(srv.URL, "main.star", pin), false)
	require.NoError(t, err)
	require.Equal(t, starlark.False, g["CHANGED"])
}

func TestLoadRemoteFolderStaysInside(t *testing.T) {
	e := newEnv(t)
	srv := serve(t, agent, nil)
	pin := `sha256 = "` + sumOf(agent.manifest) + `"`

	_, err := e.run(t, loadAgent(srv.URL, "escape.star", pin), false)
	require.ErrorContains(t, err, `load("../../outside.star")`)
	require.ErrorContains(t, err, "is outside the folder")

	_, err = e.run(t, loadAgent(srv.URL, "escape_resource.star", pin), false)
	require.ErrorContains(t, err, `load_dynamic(label="outside"): module: "../../outside.star" is outside the folder`)

	_, err = e.run(t, loadAgent(srv.URL, "escape_here.star", pin), false)
	require.ErrorContains(t, err, "here: \"../../x\" is outside the folder")

	_, err = e.run(t, loadAgent(srv.URL, "missing.star", pin), false)
	require.ErrorContains(t, err, `entry "missing.star" is not in`)

	_, err = e.run(t, loadAgent(srv.URL, "templates/motd.txt", pin), false)
	require.ErrorContains(t, err, `entry "templates/motd.txt" doesn't name a .star file`)
}

func TestLoadRemoteFolderTrustedKeys(t *testing.T) {
	e := newEnv(t)
	sigURL := "/cfg/" + signing.ManifestName + ".sig"
	srv := serve(t, agent, map[string][]byte{sigURL: e.sign(agent.manifest)})

	// Pinned by signature: the folder's .star files need no .sig of their own.
	_, err := e.run(t, loadAgent(srv.URL, "main.star", `signature = "`+srv.URL+sigURL+`"`), true)
	require.NoError(t, err)

	// Pinned by sha256 into an empty store, there's no signature to trust.
	require.NoError(t, os.RemoveAll(e.storeDir))
	_, err = e.run(t, loadAgent(srv.URL, "main.star", `sha256 = "`+sumOf(agent.manifest)+`"`), true)
	require.ErrorContains(t, err, "signature check failed for")
	require.ErrorContains(t, err, "fetch the folder pinned by signature")
}

// A file changed in the store after the folder was assembled stops the run.
func TestLoadRemoteFolderTampered(t *testing.T) {
	e := newEnv(t)
	srv := serve(t, agent, nil)
	pin := `sha256 = "` + sumOf(agent.manifest) + `"`
	_, err := e.run(t, loadAgent(srv.URL, "main.star", pin), false)
	require.NoError(t, err)

	p := filepath.Join(e.storeDir, "folders", sumOf(agent.manifest), "lib", "settings.star")
	require.NoError(t, os.Chmod(p, 0o644))
	require.NoError(t, os.Remove(p))
	require.NoError(t, os.WriteFile(p, []byte(`SETTINGS = {"channel": "evil"}`), 0o644))
	_, err = e.run(t, loadAgent(srv.URL, "main.star", pin), false)
	require.ErrorContains(t, err, "changed after it was stored")
}

// checkFolderModule refuses a module whose bytes don't match the manifest,
// even when the folder passed its last check.
func TestCheckFolderModule(t *testing.T) {
	e := newEnv(t)
	srv := serve(t, agent, nil)
	_, err := e.run(t, loadAgent(srv.URL, "main.star", `sha256 = "`+sumOf(agent.manifest)+`"`), false)
	require.NoError(t, err)

	l := Loader{Fsys: afero.NewOsFs(), StoreDir: e.storeDir}
	root := filepath.Join(e.storeDir, "folders", sumOf(agent.manifest))
	require.NoError(t, l.CheckSignature(filepath.Join(root, "main.star"), []byte(agent.files["main.star"])))
	require.ErrorContains(t, l.CheckSignature(filepath.Join(root, "main.star"), []byte("evil")), "main.star has sha256")
	require.ErrorContains(t, l.CheckSignature(filepath.Join(root, "other.star"), []byte("x")), "the manifest doesn't list other.star")
}
