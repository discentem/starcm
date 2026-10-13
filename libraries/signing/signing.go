// Package signing creates and checks detached signatures for files starcm runs
// or handles. A file's signature lives next to it in <file>.sig, as one line:
//
//	ed25519 key=2026-a sig=<base64>
//
// The algorithm selects a Verifier and key names a public key in a Keyring.
// Keys are PEM files as written by openssl:
//
//	openssl genpkey -algorithm ed25519 -out 2026-a.pem
//	openssl pkey -in 2026-a.pem -pubout -out 2026-a.pub.pem
package signing

import (
	"bytes"
	"crypto"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/afero"
)

// SignatureSuffix is appended to a file's path to find its signature file.
const SignatureSuffix = ".sig"

// Verifier checks signatures made with one algorithm.
type Verifier interface {
	// Verify returns nil if sig is a valid signature over data by pub.
	Verify(pub crypto.PublicKey, data, sig []byte) error
}

// Verifiers maps an algorithm name, as written in a signature file, to the
// Verifier for it. Supporting another algorithm means adding an entry here.
var Verifiers = map[string]Verifier{
	"ed25519": ed25519Verifier{},
}

type ed25519Verifier struct{}

func (ed25519Verifier) Verify(pub crypto.PublicKey, data, sig []byte) error {
	key, ok := pub.(ed25519.PublicKey)
	if !ok {
		return fmt.Errorf("key is %s, not ed25519", keyType(pub))
	}
	if !ed25519.Verify(key, data, sig) {
		return errors.New("signature does not match")
	}
	return nil
}

// Signature is the content of a signature file.
type Signature struct {
	Algorithm string
	KeyID     string
	Value     []byte
}

// ParseSignature parses a signature file.
func ParseSignature(b []byte) (Signature, error) {
	fields := strings.Fields(string(b))
	if len(fields) != 3 || !strings.HasPrefix(fields[1], "key=") || !strings.HasPrefix(fields[2], "sig=") {
		return Signature{}, errors.New(`malformed signature file, want "<algorithm> key=<id> sig=<base64>"`)
	}
	keyID := strings.TrimPrefix(fields[1], "key=")
	if keyID == "" {
		return Signature{}, errors.New("malformed signature file: empty key id")
	}
	value, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(fields[2], "sig="))
	if err != nil {
		return Signature{}, fmt.Errorf("malformed signature file: %w", err)
	}
	return Signature{Algorithm: fields[0], KeyID: keyID, Value: value}, nil
}

// Bytes formats s as a signature file.
func (s Signature) Bytes() []byte {
	return []byte(fmt.Sprintf("%s key=%s sig=%s\n", s.Algorithm, s.KeyID, base64.StdEncoding.EncodeToString(s.Value)))
}

// KeyIDFromPath derives a key ID from a key file's name: everything before
// the first dot, so keys/2026-a.pub.pem and 2026-a.pem are both "2026-a".
func KeyIDFromPath(path string) string {
	id, _, _ := strings.Cut(filepath.Base(path), ".")
	return id
}

// ParsePublicKey parses a PEM "PUBLIC KEY" block (PKIX).
func ParsePublicKey(pemBytes []byte) (crypto.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, errors.New(`not a PEM "PUBLIC KEY"`)
	}
	return x509.ParsePKIXPublicKey(block.Bytes)
}

// ParsePrivateKey parses a PEM "PRIVATE KEY" block (PKCS #8).
func ParsePrivateKey(pemBytes []byte) (crypto.Signer, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil || block.Type != "PRIVATE KEY" {
		return nil, errors.New(`not a PEM "PRIVATE KEY"`)
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	signer, ok := key.(crypto.Signer)
	if !ok {
		return nil, fmt.Errorf("unsupported private key type %T", key)
	}
	return signer, nil
}

// Sign signs data with key, recording keyID so verifiers can find the
// matching public key.
func Sign(key crypto.Signer, keyID string, data []byte) (Signature, error) {
	if keyID == "" || strings.ContainsAny(keyID, " \t\r\n") {
		return Signature{}, fmt.Errorf("invalid key id %q: must be non-empty with no spaces", keyID)
	}
	switch k := key.(type) {
	case ed25519.PrivateKey:
		return Signature{Algorithm: "ed25519", KeyID: keyID, Value: ed25519.Sign(k, data)}, nil
	default:
		return Signature{}, fmt.Errorf("signing with %s keys is not supported", keyType(key.Public()))
	}
}

// SignFile signs the file at path with the PEM private key at keyPath and
// writes path + SignatureSuffix. keyPath may start with ~. keyID defaults to
// KeyIDFromPath(keyPath).
// changed is false when the signature file already had this content (ed25519
// signatures are deterministic), in which case it isn't rewritten.
func SignFile(fsys afero.Fs, keyPath, keyID, path string) (sig Signature, changed bool, err error) {
	if keyPath, err = homedir.Expand(keyPath); err != nil {
		return Signature{}, false, err
	}
	keyPEM, err := afero.ReadFile(fsys, keyPath)
	if err != nil {
		return Signature{}, false, err
	}
	key, err := ParsePrivateKey(keyPEM)
	if err != nil {
		return Signature{}, false, fmt.Errorf("%s: %w", keyPath, err)
	}
	if keyID == "" {
		keyID = KeyIDFromPath(keyPath)
	}
	data, err := afero.ReadFile(fsys, path)
	if err != nil {
		return Signature{}, false, err
	}
	if sig, err = Sign(key, keyID, data); err != nil {
		return Signature{}, false, err
	}
	sigPath := path + SignatureSuffix
	if old, err := afero.ReadFile(fsys, sigPath); err == nil && bytes.Equal(old, sig.Bytes()) {
		return sig, false, nil
	}
	if err := afero.WriteFile(fsys, sigPath, sig.Bytes(), 0o644); err != nil {
		return Signature{}, false, err
	}
	return sig, true, nil
}

// Keyring holds the public keys a run trusts, by key ID.
type Keyring struct {
	keys map[string]crypto.PublicKey
}

// LoadKeyring reads every *.pem file in dir as a public key, named by
// KeyIDFromPath.
func LoadKeyring(fsys afero.Fs, dir string) (*Keyring, error) {
	entries, err := afero.ReadDir(fsys, dir)
	if err != nil {
		return nil, fmt.Errorf("reading trusted keys: %w", err)
	}
	keys := map[string]crypto.PublicKey{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".pem") {
			continue
		}
		id := KeyIDFromPath(e.Name())
		if _, dup := keys[id]; dup {
			return nil, fmt.Errorf("reading trusted keys: more than one key file for key id %q in %s", id, dir)
		}
		b, err := afero.ReadFile(fsys, filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("reading trusted keys: %w", err)
		}
		pub, err := ParsePublicKey(b)
		if err != nil {
			return nil, fmt.Errorf("reading trusted keys: %s: %w", e.Name(), err)
		}
		keys[id] = pub
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("reading trusted keys: no *.pem public keys in %s", dir)
	}
	return &Keyring{keys: keys}, nil
}

// KeyIDs returns the trusted key IDs, sorted.
func (k *Keyring) KeyIDs() []string {
	ids := make([]string, 0, len(k.keys))
	for id := range k.keys {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// Verify checks data against sigFile, the content of a signature file, and
// returns the parsed signature.
func (k *Keyring) Verify(data, sigFile []byte) (Signature, error) {
	sig, err := ParseSignature(sigFile)
	if err != nil {
		return Signature{}, err
	}
	v, ok := Verifiers[sig.Algorithm]
	if !ok {
		return sig, fmt.Errorf("unsupported signature algorithm %q", sig.Algorithm)
	}
	pub, ok := k.keys[sig.KeyID]
	if !ok {
		return sig, fmt.Errorf("key %q is not trusted", sig.KeyID)
	}
	if err := v.Verify(pub, data, sig.Value); err != nil {
		return sig, fmt.Errorf("key %q: %w", sig.KeyID, err)
	}
	return sig, nil
}

// CheckFile verifies data, the content of the file at path, against the
// signature file next to it (path + SignatureSuffix). Callers pass the bytes
// they are about to use so the check covers exactly those bytes.
func (k *Keyring) CheckFile(fsys afero.Fs, path string, data []byte) error {
	sigPath := path + SignatureSuffix
	sigFile, err := afero.ReadFile(fsys, sigPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("signature check failed for %s: no signature file %s", path, sigPath)
		}
		return fmt.Errorf("signature check failed for %s: %w", path, err)
	}
	if _, err := k.Verify(data, sigFile); err != nil {
		return fmt.Errorf("signature check failed for %s: %w", path, err)
	}
	return nil
}

func keyType(pub crypto.PublicKey) string {
	switch pub.(type) {
	case ed25519.PublicKey:
		return "ed25519"
	default:
		return fmt.Sprintf("%T", pub)
	}
}
