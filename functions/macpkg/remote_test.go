package macpkg

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/discentem/starcm/functions/base"
	"github.com/discentem/starcm/functions/download"
	localstore "github.com/discentem/starcm/libraries/local_store"
	"github.com/discentem/starcm/libraries/signing"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
)

// pkgServer serves fixed paths and counts requests for each.
type pkgServer struct {
	*httptest.Server
	mu   sync.Mutex
	hits map[string]int
}

func servePkgs(t *testing.T, content map[string][]byte) *pkgServer {
	t.Helper()
	s := &pkgServer{hits: map[string]int{}}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.hits[r.URL.Path]++
		s.mu.Unlock()
		b, ok := content[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Write(b)
	}))
	t.Cleanup(s.Close)
	return s
}

func (s *pkgServer) hit(path string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hits[path]
}

// trustedKey returns a keyring trusting a new key, and a function that signs
// data with it.
func trustedKey(t *testing.T) (*signing.Keyring, func(data []byte) []byte) {
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

// remoteSetup is install_remote_mac_pkg() with a scratch store, as the
// loader configures it.
type remoteSetup struct {
	mac     *fakeMac
	module  *base.Module
	storeAt string
}

func newRemote(t *testing.T, goos string, keyring *signing.Keyring) *remoteSetup {
	t.Helper()
	perms, err := localstore.NewPermissions()
	require.NoError(t, err)
	dir := filepath.Join(t.TempDir(), "store")
	open := localstore.Lazy(dir, perms)
	f := download.NewFetcher(afero.NewOsFs(), io.Discard, download.WithStore(open), download.WithKeyring(keyring))
	mac := &fakeMac{checkOutput: readTestdata(t, "developer-id-notarized.txt")}
	return &remoteSetup{
		mac:     mac,
		storeAt: dir,
		module:  NewRemote(context.Background(), f, mac, WithGOOS(goos), WithStoreCheck(localstore.CheckIfStored(dir, open))),
	}
}

func (s *remoteSetup) call(kwargs ...any) (*base.Result, error) {
	kv := append(kw(kwargs...), starlark.Tuple{starlark.String("label"), starlark.String("agent")})
	return s.module.Call(nil, nil, kv)
}

func TestRemote(t *testing.T) {
	pkg := []byte("pkg bytes")

	t.Run("fetches into the store by sha256 and installs", func(t *testing.T) {
		srv := servePkgs(t, map[string][]byte{"/agent-1.2.0.pkg": pkg})
		s := newRemote(t, "darwin", nil)
		url := srv.URL + "/agent-1.2.0.pkg"
		r, err := s.call("url", url, "sha256", pkgBytesSHA256, "signer", tailscaleSigner)
		require.NoError(t, err)
		assert.True(t, r.Changed)
		assert.Equal(t, `installed `+url+`, signed by "`+tailscaleSigner+`"`, *r.Message)
		assert.Equal(t, []string{"pkgutil --check-signature", "installer -pkg"}, s.mac.commands())
		assert.Equal(t, "pkg bytes", s.mac.seen)
		team, _ := r.Return.(interface {
			Attr(string) (starlark.Value, error)
		}).Attr("team_id")
		assert.Equal(t, starlark.String("W5364U7YZB"), team)
		assert.FileExists(t, filepath.Join(s.storeAt, "sha256", pkgBytesSHA256, "agent-1.2.0.pkg"))

		// A second run installs from the store without fetching again.
		_, err = s.call("url", url, "sha256", strings.ToUpper(pkgBytesSHA256), "signer", tailscaleSigner)
		require.NoError(t, err)
		assert.Equal(t, 1, srv.hit("/agent-1.2.0.pkg"))
	})

	t.Run("fetches by signature under --trusted-keys", func(t *testing.T) {
		keyring, sign := trustedKey(t)
		srv := servePkgs(t, map[string][]byte{"/agent.pkg": pkg, "/agent.pkg.sig": sign(pkg)})
		s := newRemote(t, "darwin", keyring)
		r, err := s.call("url", srv.URL+"/agent.pkg", "signature", srv.URL+"/agent.pkg.sig", "signer", tailscaleSigner)
		require.NoError(t, err)
		assert.True(t, r.Changed)
		assert.Equal(t, "pkg bytes", s.mac.seen)
		assert.FileExists(t, filepath.Join(s.storeAt, "sha256", pkgBytesSHA256, "agent.pkg.sig"))

		s = newRemote(t, "darwin", nil)
		_, err = s.call("url", srv.URL+"/agent.pkg", "signature", srv.URL+"/agent.pkg.sig", "signer", tailscaleSigner)
		assert.EqualError(t, err, `install_remote_mac_pkg(label="agent"): signature needs trusted keys: run starcm with --trusted-keys`)
	})

	t.Run("an installed version isn't fetched", func(t *testing.T) {
		srv := servePkgs(t, map[string][]byte{"/agent.pkg": pkg})
		s := newRemote(t, "darwin", nil)
		s.mac.pkgInfo = "package-id: com.example.agent\nversion: 1.2.0\n"
		r, err := s.call("url", srv.URL+"/agent.pkg", "sha256", pkgBytesSHA256, "signer", tailscaleSigner, "id", "com.example.agent", "version", "1.2.0")
		require.NoError(t, err)
		assert.False(t, r.Changed)
		assert.Equal(t, "com.example.agent 1.2.0 is already installed", *r.Message)
		assert.Equal(t, []string{"pkgutil --pkg-info"}, s.mac.commands())
		assert.Zero(t, srv.hit("/agent.pkg"))
	})

	t.Run("what_if fetches and checks but doesn't install", func(t *testing.T) {
		srv := servePkgs(t, map[string][]byte{"/agent.pkg": pkg})
		s := newRemote(t, "darwin", nil)
		r, err := s.call("url", srv.URL+"/agent.pkg", "sha256", pkgBytesSHA256, "signer", tailscaleSigner, "what_if", true)
		require.NoError(t, err)
		assert.True(t, r.Changed)
		assert.Equal(t, `would install `+srv.URL+`/agent.pkg, signed by "`+tailscaleSigner+`"`, *r.Message)
		assert.Equal(t, []string{"pkgutil --check-signature"}, s.mac.commands())
	})

	t.Run("a package that doesn't match its pin is never checked or installed", func(t *testing.T) {
		srv := servePkgs(t, map[string][]byte{"/agent.pkg": []byte("something else")})
		s := newRemote(t, "darwin", nil)
		_, err := s.call("url", srv.URL+"/agent.pkg", "sha256", pkgBytesSHA256, "signer", tailscaleSigner)
		assert.ErrorContains(t, err, "expected sha256 hash "+pkgBytesSHA256)
		assert.Empty(t, s.mac.calls)
	})

	t.Run("a refused signature stops before installer", func(t *testing.T) {
		srv := servePkgs(t, map[string][]byte{"/agent.pkg": pkg})
		s := newRemote(t, "darwin", nil)
		_, err := s.call("url", srv.URL+"/agent.pkg", "sha256", pkgBytesSHA256, "signer", "Developer ID Installer: Example Corp (ABCDE12345)")
		assert.ErrorContains(t, err, `install_remote_mac_pkg(label="agent"): `+srv.URL+`/agent.pkg: signed by "`+tailscaleSigner+`"`)
		assert.Equal(t, []string{"pkgutil --check-signature"}, s.mac.commands())
	})

	t.Run("argument errors come before the platform check, and nothing is fetched", func(t *testing.T) {
		srv := servePkgs(t, map[string][]byte{"/agent.pkg": pkg})
		s := newRemote(t, "linux", nil)
		url := srv.URL + "/agent.pkg"
		for _, tc := range []struct {
			args []any
			want string
		}{
			{[]any{"url", url, "signer", tailscaleSigner}, "pass exactly one of sha256 or signature"},
			{[]any{"url", url, "signer", tailscaleSigner, "sha256", pkgBytesSHA256, "signature", url + ".sig"}, "pass exactly one of sha256 or signature"},
			{[]any{"url", url, "signer", " ", "sha256", pkgBytesSHA256}, "signer must name the certificate"},
			{[]any{"url", url, "signer", tailscaleSigner, "sha256", "abc"}, `sha256 "abc" is not 64 hex characters`},
			{[]any{"url", srv.URL + "/agent.dmg", "signer", tailscaleSigner, "sha256", pkgBytesSHA256}, srv.URL + "/agent.dmg doesn't name a .pkg file"},
			{[]any{"url", "ftp://example.com/agent.pkg", "signer", tailscaleSigner, "sha256", pkgBytesSHA256}, "url must start with http://, https:// or s3://"},
			{[]any{"url", url, "signer", tailscaleSigner, "sha256", pkgBytesSHA256, "region", "us-west-2"}, "endpoint and region are only for s3:// URLs"},
			{[]any{"url", "s3://releases/agent.pkg", "signer", tailscaleSigner, "signature", "s3://elsewhere/agent.pkg.sig"}, `signature s3://elsewhere/agent.pkg.sig must be an s3:// URL in bucket "releases"`},
			{[]any{"url", url, "signer", tailscaleSigner, "sha256", pkgBytesSHA256}, "install_remote_mac_pkg() only works on macOS"},
		} {
			_, err := s.call(tc.args...)
			assert.ErrorContains(t, err, `install_remote_mac_pkg(label="agent"): `+tc.want)
		}
		assert.Zero(t, srv.hit("/agent.pkg"))
		assert.Empty(t, s.mac.calls)
	})
}
