package signing

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func newKey(t *testing.T) (pubPEM []byte, priv ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalPKIXPublicKey(pub)
	require.NoError(t, err)
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}), priv
}

func keyring(t *testing.T, files map[string][]byte) (*Keyring, error) {
	t.Helper()
	fsys := afero.NewMemMapFs()
	require.NoError(t, fsys.MkdirAll("keys", 0o755))
	for name, b := range files {
		require.NoError(t, afero.WriteFile(fsys, "keys/"+name, b, 0o644))
	}
	return LoadKeyring(fsys, "keys")
}

func TestSignAndVerify(t *testing.T) {
	pubPEM, priv := newKey(t)
	otherPEM, _ := newKey(t)
	k, err := keyring(t, map[string][]byte{"2026-a.pub.pem": pubPEM, "2026-b.pem": otherPEM, "README": []byte("ignored")})
	require.NoError(t, err)
	require.Equal(t, []string{"2026-a", "2026-b"}, k.KeyIDs())

	data := []byte("print('hi')\n")
	sig, err := Sign(priv, "2026-a", data)
	require.NoError(t, err)

	parsed, err := ParseSignature(sig.Bytes())
	require.NoError(t, err)
	require.Equal(t, sig, parsed)

	got, err := k.Verify(data, sig.Bytes())
	require.NoError(t, err)
	require.Equal(t, "2026-a", got.KeyID)

	t.Run("tampered data", func(t *testing.T) {
		_, err := k.Verify([]byte("print('pwned')\n"), sig.Bytes())
		require.EqualError(t, err, `key "2026-a": signature does not match`)
	})
	t.Run("signed by a different trusted key than claimed", func(t *testing.T) {
		lie := sig
		lie.KeyID = "2026-b"
		_, err := k.Verify(data, lie.Bytes())
		require.EqualError(t, err, `key "2026-b": signature does not match`)
	})
	t.Run("untrusted key", func(t *testing.T) {
		_, untrusted := newKey(t)
		s, err := Sign(untrusted, "someone-else", data)
		require.NoError(t, err)
		_, err = k.Verify(data, s.Bytes())
		require.EqualError(t, err, `key "someone-else" is not trusted`)
	})
	t.Run("unknown algorithm", func(t *testing.T) {
		s := sig
		s.Algorithm = "rot13"
		_, err := k.Verify(data, s.Bytes())
		require.EqualError(t, err, `unsupported signature algorithm "rot13"`)
	})
	t.Run("malformed", func(t *testing.T) {
		for _, bad := range []string{"", "ed25519", "ed25519 key= sig=AAAA", "ed25519 key=a sig=!!!", "ed25519 sig=AAAA key=a"} {
			_, err := k.Verify(data, []byte(bad))
			require.Error(t, err, bad)
			require.Contains(t, err.Error(), "malformed signature file", bad)
		}
	})
}

func TestCheckFile(t *testing.T) {
	pubPEM, priv := newKey(t)
	k, err := keyring(t, map[string][]byte{"k.pub.pem": pubPEM})
	require.NoError(t, err)

	fsys := afero.NewMemMapFs()
	data := []byte("x = 1\n")
	sig, err := Sign(priv, "k", data)
	require.NoError(t, err)
	require.NoError(t, afero.WriteFile(fsys, "main.star.sig", sig.Bytes(), 0o644))

	require.NoError(t, k.CheckFile(fsys, "main.star", data))
	require.EqualError(t, k.CheckFile(fsys, "main.star", []byte("x = 2\n")),
		`signature check failed for main.star: key "k": signature does not match`)
	require.EqualError(t, k.CheckFile(fsys, "other.star", data),
		"signature check failed for other.star: no signature file other.star.sig")
}

func TestLoadKeyringErrors(t *testing.T) {
	pubPEM, _ := newKey(t)

	_, err := keyring(t, nil)
	require.EqualError(t, err, "reading trusted keys: no *.pem public keys in keys")

	_, err = keyring(t, map[string][]byte{"a.pem": pubPEM, "a.pub.pem": pubPEM})
	require.EqualError(t, err, `reading trusted keys: more than one key file for key id "a" in keys`)

	_, err = keyring(t, map[string][]byte{"a.pem": []byte("not pem")})
	require.EqualError(t, err, `reading trusted keys: a.pem: not a PEM "PUBLIC KEY"`)

	_, err = LoadKeyring(afero.NewMemMapFs(), "missing")
	require.ErrorContains(t, err, "reading trusted keys:")
}

// A key that isn't ed25519 is accepted into the keyring (so other verifiers can
// use it later) but can't verify an ed25519 signature.
func TestVerifyWrongKeyType(t *testing.T) {
	ec, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalPKIXPublicKey(&ec.PublicKey)
	require.NoError(t, err)
	k, err := keyring(t, map[string][]byte{"ec.pem": pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})})
	require.NoError(t, err)

	_, priv := newKey(t)
	sig, err := Sign(priv, "ec", []byte("data"))
	require.NoError(t, err)
	_, err = k.Verify([]byte("data"), sig.Bytes())
	require.EqualError(t, err, `key "ec": key is *ecdsa.PublicKey, not ed25519`)

	_, err = Sign(ec, "ec", []byte("data"))
	require.EqualError(t, err, "signing with *ecdsa.PublicKey keys is not supported")
}

func TestSignRejectsBadKeyID(t *testing.T) {
	_, priv := newKey(t)
	for _, id := range []string{"", "has space", "tab\there"} {
		_, err := Sign(priv, id, []byte("x"))
		require.Error(t, err, id)
	}
}

// Keys and signatures made with openssl (see the package doc) work as-is: the
// public key file is used unchanged and the raw signature only needs wrapping.
func TestOpenSSLInterop(t *testing.T) {
	pubPEM, err := os.ReadFile("testdata/openssl.pub.pem")
	require.NoError(t, err)
	data, err := os.ReadFile("testdata/openssl-data.txt")
	require.NoError(t, err)
	raw, err := os.ReadFile("testdata/openssl-data.txt.raw-sig")
	require.NoError(t, err)

	k, err := keyring(t, map[string][]byte{"openssl.pub.pem": pubPEM})
	require.NoError(t, err)
	sig := Signature{Algorithm: "ed25519", KeyID: "openssl", Value: raw}
	_, err = k.Verify(data, sig.Bytes())
	require.NoError(t, err)
}

func TestKeyIDFromPath(t *testing.T) {
	require.Equal(t, "2026-a", KeyIDFromPath("/keys/2026-a.pub.pem"))
	require.Equal(t, "2026-a", KeyIDFromPath("2026-a.pem"))
	require.Equal(t, "plain", KeyIDFromPath("plain"))
}
