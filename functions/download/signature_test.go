package download

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
	"testing"
	"time"

	"github.com/discentem/starcm/functions/base"
	localstore "github.com/discentem/starcm/libraries/local_store"
	"github.com/discentem/starcm/libraries/signing"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
)

// testKey returns a keyring trusting a new ed25519 key with id "release",
// and a function that signs data with it.
func testKey(t *testing.T) (*signing.Keyring, func(data []byte) []byte) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalPKIXPublicKey(pub)
	require.NoError(t, err)
	fsys := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fsys, "trusted/release.pub.pem", pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}), 0o644))
	k, err := signing.LoadKeyring(fsys, "trusted")
	require.NoError(t, err)
	return k, func(data []byte) []byte {
		sig, err := signing.Sign(priv, "release", "", time.Now().Add(time.Hour), data)
		require.NoError(t, err)
		return sig.Bytes()
	}
}

// files serves a fixed set of paths.
func files(t *testing.T, content map[string][]byte) *httptest.Server {
	t.Helper()
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

func kw(pairs ...string) []starlark.Tuple {
	var out []starlark.Tuple
	for i := 0; i < len(pairs); i += 2 {
		out = append(out, starlark.Tuple{starlark.String(pairs[i]), starlark.String(pairs[i+1])})
	}
	return out
}

func TestDownloadPinnedBySignature(t *testing.T) {
	keyring, sign := testKey(t)
	payload := []byte("agent v2\n")
	sig := sign(payload)
	srv := files(t, map[string][]byte{"/agent.txt": payload, "/agent.txt.sig": sig})
	fsys := afero.NewMemMapFs()
	a := &downloadAction{httpClient: http.DefaultClient, fsys: fsys, fn: "download", keyring: keyring}
	run := func(pairs ...string) (*base.Result, error) {
		return a.Run(context.Background(), "", "fetch", nil, nil, kw(pairs...))
	}

	r, err := run("url", srv.URL+"/agent.txt", "signature", srv.URL+"/agent.txt.sig", "save_to", "agent.txt")
	require.NoError(t, err)
	assert.True(t, r.Changed)
	got, err := afero.ReadFile(fsys, "agent.txt")
	require.NoError(t, err)
	assert.Equal(t, payload, got)
	// The signature is kept next to the file, where verify() and
	// --trusted-keys look for it.
	gotSig, err := afero.ReadFile(fsys, "agent.txt.sig")
	require.NoError(t, err)
	assert.Equal(t, sig, gotSig)

	// Present with the signed hash: nothing to do.
	r, err = run("url", srv.URL+"/agent.txt", "signature", srv.URL+"/agent.txt.sig", "save_to", "agent.txt")
	require.NoError(t, err)
	assert.False(t, r.Changed)
}

func TestDownloadSignatureRefusals(t *testing.T) {
	keyring, sign := testKey(t)
	_, otherSign := testKey(t)
	payload := []byte("agent v2\n")
	srv := files(t, map[string][]byte{
		"/agent.txt":       payload,
		"/agent.txt.sig":   sign(payload),
		"/other-file.sig":  sign([]byte("a different file\n")),
		"/untrusted.sig":   otherSign(payload),
		"/not-a-signature": []byte("hello\n"),
	})
	a := &downloadAction{httpClient: http.DefaultClient, fsys: afero.NewMemMapFs(), fn: "download", keyring: keyring}
	run := func(pairs ...string) error {
		_, err := a.Run(context.Background(), "", "fetch", nil, nil, kw(pairs...))
		return err
	}
	url := srv.URL + "/agent.txt"

	assert.EqualError(t, run("url", url, "save_to", "x"), `download(label="fetch"): sha256 or signature must be provided`)
	assert.EqualError(t, run("url", url, "save_to", "x", "sha256", helloSHA256, "signature", url+".sig"), `download(label="fetch"): pass sha256 or signature, not both`)
	// Signed by another key that claims the trusted key's id.
	assert.ErrorContains(t, run("url", url, "save_to", "x", "signature", srv.URL+"/untrusted.sig"), `key "release": signature does not match`)
	assert.ErrorContains(t, run("url", url, "save_to", "x", "signature", srv.URL+"/not-a-signature"), "malformed signature file")
	assert.ErrorContains(t, run("url", url, "save_to", "x", "signature", srv.URL+"/missing.sig"), "fetching signature: failed to download file")
	// A valid signature for some other file: the download doesn't match the
	// sha256 it vouches for.
	assert.ErrorContains(t, run("url", url, "save_to", "x", "signature", srv.URL+"/other-file.sig"), "expected sha256 hash")

	a.keyring = nil
	assert.EqualError(t, run("url", url, "save_to", "x", "signature", url+".sig"), `download(label="fetch"): signature needs trusted keys: run starcm with --trusted-keys`)
}

func TestDownloadSignatureIntoStore(t *testing.T) {
	keyring, sign := testKey(t)
	payload := []byte("x = 1\n")
	sig := sign(payload)
	srv := files(t, map[string][]byte{"/main.star": payload, "/main.star.sig": sig})
	perms, err := localstore.NewPermissions()
	require.NoError(t, err)
	open := localstore.Lazy(filepath.Join(t.TempDir(), "store"), perms)
	a := &downloadAction{httpClient: http.DefaultClient, fsys: afero.NewOsFs(), fn: "download", keyring: keyring, store: open}

	r, err := a.Run(context.Background(), "", "fetch", nil, nil, kw("url", srv.URL+"/main.star", "signature", srv.URL+"/main.star.sig"))
	require.NoError(t, err)
	stored := string(r.Return.(starlark.String))
	got, err := os.ReadFile(stored + ".sig")
	require.NoError(t, err)
	assert.Equal(t, sig, got)
	// So the loader's check (the .sig next to the file) passes for it.
	data, err := os.ReadFile(stored)
	require.NoError(t, err)
	assert.NoError(t, keyring.CheckFile(afero.NewOsFs(), stored, data))
}

func TestS3DownloadPinnedBySignature(t *testing.T) {
	awsEnv(t)
	keyring, sign := testKey(t)
	fake := &fakeS3{objects: map[string]string{
		"/fleet/roles/web.star":     "hello world",
		"/fleet/roles/web.star.sig": string(sign([]byte("hello world"))),
	}}
	srv := httptest.NewServer(fake)
	defer srv.Close()
	fsys := afero.NewMemMapFs()
	a := &s3DownloadAction{&downloadAction{httpClient: http.DefaultClient, fsys: fsys, fn: "s3_download", keyring: keyring}}

	_, err := a.Run(context.Background(), "", "fetch", nil, nil, s3Kwargs(
		"bucket", "fleet", "key", "roles/web.star", "signature", "roles/web.star.sig", "save_to", "web.star", "endpoint", srv.URL))
	require.NoError(t, err)
	got, err := afero.ReadFile(fsys, "web.star.sig")
	require.NoError(t, err)
	assert.Equal(t, fake.objects["/fleet/roles/web.star.sig"], string(got))
	// The signature came from the same bucket, then the file.
	assert.Equal(t, []string{"/fleet/roles/web.star.sig", "/fleet/roles/web.star"}, fake.paths)
}
