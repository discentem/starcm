package signing

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	starcmsha256 "github.com/discentem/starcm/libraries/sha256"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
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

// later is an expiry comfortably in the future.
var later = time.Now().Add(time.Hour)

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
	sig, err := Sign(priv, "2026-a", "", later, data)
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
		s, err := Sign(untrusted, "someone-else", "", later, data)
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
		for _, bad := range []string{"", "ed25519", "ed25519 key= expires=2026-01-01T00:00:00Z sig=AAAA",
			"ed25519 key=a expires=2026-01-01T00:00:00Z sig=!!!",
			"ed25519 key=a expires=tomorrow sig=AAAA",
			"ed25519 key=a sig=AAAA expires=2026-01-01T00:00:00Z",
			// The old format, without an expiry, is no longer accepted.
			"ed25519 key=a sig=AAAA",
		} {
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
	sig, err := Sign(priv, "k", "", later, data)
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
	sig, err := Sign(priv, "ec", "", later, []byte("data"))
	require.NoError(t, err)
	_, err = k.Verify([]byte("data"), sig.Bytes())
	require.EqualError(t, err, `key "ec": key is *ecdsa.PublicKey, not ed25519`)

	_, err = Sign(ec, "ec", "", later, []byte("data"))
	require.EqualError(t, err, "signing with *ecdsa.PublicKey keys is not supported")
}

func TestSignRejectsBadKeyID(t *testing.T) {
	_, priv := newKey(t)
	for _, id := range []string{"", "has space", "tab\there"} {
		_, err := Sign(priv, id, "", later, []byte("x"))
		require.Error(t, err, id)
	}
}

// Keys and signatures made with openssl (see the package doc) work as-is: the
// public key file is used unchanged, openssl signs Message (written out in
// testdata/openssl-data.txt.message) and the raw signature only needs
// wrapping.
//
//	openssl pkeyutl -sign -rawin -inkey key.pem -in openssl-data.txt.message -out openssl-data.txt.raw-sig
func TestOpenSSLInterop(t *testing.T) {
	pubPEM, err := os.ReadFile("testdata/openssl.pub.pem")
	require.NoError(t, err)
	data, err := os.ReadFile("testdata/openssl-data.txt")
	require.NoError(t, err)
	message, err := os.ReadFile("testdata/openssl-data.txt.message")
	require.NoError(t, err)
	raw, err := os.ReadFile("testdata/openssl-data.txt.raw-sig")
	require.NoError(t, err)

	expires := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	require.Equal(t, string(message), string(Message("ed25519", "openssl", expires, "", data)))

	k, err := keyring(t, map[string][]byte{"openssl.pub.pem": pubPEM})
	require.NoError(t, err)
	sum, err := starcmsha256.FromReader(bytes.NewReader(data))
	require.NoError(t, err)
	sig := Signature{Algorithm: "ed25519", KeyID: "openssl", Expires: expires, SHA256: sum, Value: raw}
	_, err = k.Verify(data, sig.Bytes())
	require.NoError(t, err)
}

func TestExpiry(t *testing.T) {
	pubPEM, priv := newKey(t)
	k, err := keyring(t, map[string][]byte{"k.pub.pem": pubPEM})
	require.NoError(t, err)
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	k.now = func() time.Time { return now }
	data := []byte("x = 1\n")

	sign := func(expires time.Time) Signature {
		t.Helper()
		sig, err := Sign(priv, "k", "", expires, data)
		require.NoError(t, err)
		return sig
	}

	t.Run("valid until expiry", func(t *testing.T) {
		_, err := k.Verify(data, sign(now.Add(time.Second)).Bytes())
		require.NoError(t, err)
	})
	t.Run("expired", func(t *testing.T) {
		_, err := k.Verify(data, sign(now).Bytes())
		require.EqualError(t, err, `key "k": signature expired at 2026-10-15T12:00:00Z (now 2026-10-15T12:00:00Z)`)
		_, err = k.Verify(data, sign(now.Add(-24*time.Hour)).Bytes())
		require.EqualError(t, err, `key "k": signature expired at 2026-10-14T12:00:00Z (now 2026-10-15T12:00:00Z)`)
	})
	t.Run("expiry is signed", func(t *testing.T) {
		extended := sign(now.Add(-time.Hour))
		extended.Expires = now.Add(time.Hour)
		_, err := k.Verify(data, extended.Bytes())
		require.EqualError(t, err, `key "k": signature does not match`)
	})
	t.Run("max expiry", func(t *testing.T) {
		k.SetMaxExpiry(30 * 24 * time.Hour)
		defer k.SetMaxExpiry(0)

		_, err := k.Verify(data, sign(now.Add(30*24*time.Hour)).Bytes())
		require.NoError(t, err)
		_, err = k.Verify(data, sign(now.Add(30*24*time.Hour+time.Second)).Bytes())
		require.EqualError(t, err, `key "k": signature expires at 2026-11-14T12:00:01Z, more than 30d from now (2026-10-15T12:00:00Z)`)
	})
	t.Run("expiry is required", func(t *testing.T) {
		_, err := Sign(priv, "k", "", time.Time{}, data)
		require.EqualError(t, err, "signatures must have an expiry")
	})
}

func TestParseExpiry(t *testing.T) {
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	for in, want := range map[string]time.Time{
		"30d":                  now.Add(30 * 24 * time.Hour),
		"36h":                  now.Add(36 * time.Hour),
		"2026-11-14":           time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC),
		"2026-11-14T08:00:00Z": time.Date(2026, 11, 14, 8, 0, 0, 0, time.UTC),
	} {
		got, err := ParseExpiry(in, now)
		require.NoError(t, err, in)
		require.True(t, want.Equal(got), "%s: got %s, want %s", in, got, want)
	}
	for _, bad := range []string{"", "0d", "-1d", "soon", "30 days", "14/11/2026"} {
		_, err := ParseExpiry(bad, now)
		require.Error(t, err, bad)
	}
}

func TestParseDuration(t *testing.T) {
	d, err := ParseDuration("30d")
	require.NoError(t, err)
	require.Equal(t, 30*24*time.Hour, d)
	d, err = ParseDuration("90m")
	require.NoError(t, err)
	require.Equal(t, 90*time.Minute, d)
	for _, bad := range []string{"", "d", "0h", "-3d", "1.5d", "week"} {
		_, err := ParseDuration(bad)
		require.Error(t, err, bad)
	}
	require.Equal(t, "30d", formatDuration(30*24*time.Hour))
	require.Equal(t, "36h0m0s", formatDuration(36*time.Hour))
}

func TestKeyIDFromPath(t *testing.T) {
	require.Equal(t, "2026-a", KeyIDFromPath("/keys/2026-a.pub.pem"))
	require.Equal(t, "2026-a", KeyIDFromPath("2026-a.pem"))
	require.Equal(t, "plain", KeyIDFromPath("plain"))
}

func TestVerifyDigest(t *testing.T) {
	pubPEM, priv := newKey(t)
	k, err := keyring(t, map[string][]byte{"k.pub.pem": pubPEM})
	require.NoError(t, err)
	data := []byte("agent payload\n")
	sig, err := Sign(priv, "k", "", time.Now().Add(time.Hour), data)
	require.NoError(t, err)
	want, err := starcmsha256.FromReader(bytes.NewReader(data))
	require.NoError(t, err)
	assert.Equal(t, want, sig.SHA256)
	assert.Contains(t, string(sig.Bytes()), " sha256="+want+" ")

	// Without the file, the signature vouches for the sha256 it records.
	got, err := k.VerifyDigest(sig.Bytes())
	require.NoError(t, err)
	assert.Equal(t, want, got.SHA256)

	// Editing the recorded sha256 breaks the signature...
	edited := sig
	edited.SHA256 = strings.Repeat("0", 64)
	_, err = k.VerifyDigest(edited.Bytes())
	assert.EqualError(t, err, `key "k": signature does not match`)
	// ...and Verify, which hashes the file itself, says which field is off.
	_, err = k.Verify(data, edited.Bytes())
	assert.EqualError(t, err, `key "k": signature file records sha256 `+strings.Repeat("0", 64)+`, but the signature is for `+want)

	// Untrusted keys are refused either way.
	otherPub, _ := newKey(t)
	other, err := keyring(t, map[string][]byte{"other.pub.pem": otherPub})
	require.NoError(t, err)
	_, err = other.VerifyDigest(sig.Bytes())
	assert.EqualError(t, err, `key "k" is not trusted`)
}

func TestParseSignatureRequiresSHA256(t *testing.T) {
	_, err := ParseSignature([]byte("ed25519 key=k expires=2099-01-01T00:00:00Z sig=AAAA\n"))
	assert.EqualError(t, err, "malformed signature file, want "+signatureFormat)
	_, err = ParseSignature([]byte("ed25519 key=k expires=2099-01-01T00:00:00Z sha256=ABC sig=AAAA\n"))
	assert.ErrorContains(t, err, "is not 64 lowercase hex characters")
}

func TestRelease(t *testing.T) {
	pubPEM, priv := newKey(t)
	k, err := keyring(t, map[string][]byte{"k.pub.pem": pubPEM})
	require.NoError(t, err)
	data := []byte("print('hi')\n")

	oct, err := Sign(priv, "k", "2026.10", later, data)
	require.NoError(t, err)
	assert.Contains(t, string(oct.Bytes()), " release=2026.10 sha256=")
	parsed, err := ParseSignature(oct.Bytes())
	require.NoError(t, err)
	assert.Equal(t, oct, parsed)
	sep, err := Sign(priv, "k", "2026.09", later, data)
	require.NoError(t, err)
	none, err := Sign(priv, "k", "", later, data)
	require.NoError(t, err)
	assert.NotContains(t, string(none.Bytes()), "release=")

	// Without SetRelease, any release, or none, is accepted.
	for _, s := range []Signature{oct, sep, none} {
		_, err := k.Verify(data, s.Bytes())
		assert.NoError(t, err, string(s.Bytes()))
	}

	k.SetRelease("2026.10")
	got, err := k.Verify(data, oct.Bytes())
	require.NoError(t, err)
	assert.Equal(t, "2026.10", got.Release)
	_, err = k.Verify(data, sep.Bytes())
	assert.EqualError(t, err, `key "k": signature is for release "2026.09", want release "2026.10"`)
	_, err = k.Verify(data, none.Bytes())
	assert.EqualError(t, err, `key "k": signature has no release, want release "2026.10"`)
	_, err = k.VerifyDigest(sep.Bytes())
	assert.EqualError(t, err, `key "k": signature is for release "2026.09", want release "2026.10"`)

	// The release is signed: relabelling an old signature breaks it, and so
	// does adding a release to a signature made without one.
	for _, s := range []Signature{sep, none} {
		relabelled := s
		relabelled.Release = "2026.10"
		_, err = k.Verify(data, relabelled.Bytes())
		assert.EqualError(t, err, `key "k": signature does not match`)
	}

	_, err = Sign(priv, "k", "has space", later, data)
	assert.EqualError(t, err, `invalid release "has space": must have no spaces`)
	_, err = ParseSignature([]byte("ed25519 key=k expires=2099-01-01T00:00:00Z release= sha256=" + strings.Repeat("0", 64) + " sig=AAAA\n"))
	assert.EqualError(t, err, "malformed signature file: empty release")
}
