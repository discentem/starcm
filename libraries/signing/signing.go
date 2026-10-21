// Package signing creates and checks detached signatures for files starcm runs
// or handles. A file's signature lives next to it in <file>.sig, as one line:
//
//	ed25519 key=2026-a expires=2026-11-14T00:00:00Z release=2026.10.21 sha256=<hex> sig=<base64>
//
// The algorithm selects a Verifier and key names a public key in a Keyring.
// Every signature expires: what's signed is Message, which covers the
// algorithm, key id, expiry, release and the file's sha256, so none of them
// can be changed without breaking the signature, and Keyring refuses
// signatures past their expiry.
//
// release is optional. It names the set of files signed together, so a
// Keyring that requires one (SetRelease) refuses files from an older release,
// or from none, even while their signatures are unexpired.
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
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	starcmsha256 "github.com/discentem/starcm/libraries/sha256"
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
	// Expires is when the signature stops being valid, in UTC to the second.
	Expires time.Time
	// Release is the release the file was signed for, or "" for none.
	Release string
	// SHA256 is the signed file's sha256, lowercase hex. It's part of what's
	// signed, so a verified signature is also a trusted sha256 pin.
	SHA256 string
	Value  []byte
}

const signatureFormat = `"<algorithm> key=<id> expires=<RFC 3339 time> [release=<id>] sha256=<hex> sig=<base64>"`

var sha256Hex = regexp.MustCompile(`^[0-9a-f]{64}$`)

// ParseSignature parses a signature file.
func ParseSignature(b []byte) (Signature, error) {
	fields := strings.Fields(string(b))
	var release string
	if len(fields) == 6 && strings.HasPrefix(fields[3], "release=") {
		release = strings.TrimPrefix(fields[3], "release=")
		if release == "" {
			return Signature{}, errors.New("malformed signature file: empty release")
		}
		fields = append(fields[:3], fields[4:]...)
	}
	if len(fields) != 5 ||
		!strings.HasPrefix(fields[1], "key=") ||
		!strings.HasPrefix(fields[2], "expires=") ||
		!strings.HasPrefix(fields[3], "sha256=") ||
		!strings.HasPrefix(fields[4], "sig=") {
		return Signature{}, errors.New("malformed signature file, want " + signatureFormat)
	}
	keyID := strings.TrimPrefix(fields[1], "key=")
	if keyID == "" {
		return Signature{}, errors.New("malformed signature file: empty key id")
	}
	expires, err := time.Parse(time.RFC3339, strings.TrimPrefix(fields[2], "expires="))
	if err != nil {
		return Signature{}, fmt.Errorf("malformed signature file: expires: %w", err)
	}
	sum := strings.TrimPrefix(fields[3], "sha256=")
	if !sha256Hex.MatchString(sum) {
		return Signature{}, fmt.Errorf("malformed signature file: sha256 %q is not 64 lowercase hex characters", sum)
	}
	value, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(fields[4], "sig="))
	if err != nil {
		return Signature{}, fmt.Errorf("malformed signature file: %w", err)
	}
	return Signature{Algorithm: fields[0], KeyID: keyID, Expires: expires.UTC(), Release: release, SHA256: sum, Value: value}, nil
}

// Bytes formats s as a signature file.
func (s Signature) Bytes() []byte {
	var release string
	if s.Release != "" {
		release = " release=" + s.Release
	}
	return []byte(fmt.Sprintf("%s key=%s expires=%s%s sha256=%s sig=%s\n",
		s.Algorithm, s.KeyID, formatTime(s.Expires), release, s.SHA256, base64.StdEncoding.EncodeToString(s.Value)))
}

// Message is what a signature by keyID with this algorithm, expiry and
// release ("" for none) signs for data. Signing the file's sha256 rather than
// the file keeps verifying large files cheap.
func Message(algorithm, keyID string, expires time.Time, release string, data []byte) []byte {
	sum, _ := starcmsha256.FromReader(bytes.NewReader(data)) // reading memory can't fail
	return digestMessage(algorithm, keyID, expires, release, sum)
}

// digestMessage is Message for a file whose sha256 is sum. Without a release
// the message has no release line, so signatures made before releases existed
// still verify, and one can't be turned into the other.
func digestMessage(algorithm, keyID string, expires time.Time, release, sum string) []byte {
	var releaseLine string
	if release != "" {
		releaseLine = "release=" + release + "\n"
	}
	return []byte(fmt.Sprintf("starcm-signature\nalgorithm=%s\nkey=%s\nexpires=%s\n%ssha256=%s\n",
		algorithm, keyID, formatTime(expires), releaseLine, sum))
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// ParseDuration parses a duration such as "30d", "36h" or "90m". "d" means 24
// hours; anything else is as time.ParseDuration. It must be positive.
func ParseDuration(s string) (time.Duration, error) {
	var d time.Duration
	if days, ok := strings.CutSuffix(s, "d"); ok {
		n, err := strconv.Atoi(days)
		if err != nil {
			return 0, fmt.Errorf("invalid duration %q", s)
		}
		d = time.Duration(n) * 24 * time.Hour
	} else {
		var err error
		if d, err = time.ParseDuration(s); err != nil {
			return 0, fmt.Errorf("invalid duration %q", s)
		}
	}
	if d <= 0 {
		return 0, fmt.Errorf("duration %q must be positive", s)
	}
	return d, nil
}

// ParseExpiry parses when a signature should expire: a duration from now (see
// ParseDuration), a date ("2026-11-14", midnight UTC) or an RFC 3339 time.
func ParseExpiry(s string, now time.Time) (time.Time, error) {
	if d, err := ParseDuration(s); err == nil {
		return now.Add(d), nil
	}
	if t, err := time.Parse(time.DateOnly, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("invalid expiry %q: want a duration like 30d, a date like 2026-11-14, or an RFC 3339 time", s)
}

// formatDuration prints whole days as "30d" and anything else as Go does.
func formatDuration(d time.Duration) string {
	if day := 24 * time.Hour; d%day == 0 {
		return fmt.Sprintf("%dd", d/day)
	}
	return d.String()
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

// Sign signs data with key until expires, for release ("" for none),
// recording keyID so verifiers can find the matching public key.
func Sign(key crypto.Signer, keyID, release string, expires time.Time, data []byte) (Signature, error) {
	if keyID == "" || strings.ContainsAny(keyID, " \t\r\n") {
		return Signature{}, fmt.Errorf("invalid key id %q: must be non-empty with no spaces", keyID)
	}
	if strings.ContainsAny(release, " \t\r\n") {
		return Signature{}, fmt.Errorf("invalid release %q: must have no spaces", release)
	}
	if expires.IsZero() {
		return Signature{}, errors.New("signatures must have an expiry")
	}
	expires = expires.UTC().Truncate(time.Second)
	switch k := key.(type) {
	case ed25519.PrivateKey:
		sum, _ := starcmsha256.FromReader(bytes.NewReader(data)) // reading memory can't fail
		msg := digestMessage("ed25519", keyID, expires, release, sum)
		return Signature{Algorithm: "ed25519", KeyID: keyID, Expires: expires, Release: release, SHA256: sum, Value: ed25519.Sign(k, msg)}, nil
	default:
		return Signature{}, fmt.Errorf("signing with %s keys is not supported", keyType(key.Public()))
	}
}

// SignFile signs the file at path with the PEM private key at keyPath, valid
// until expires, for release ("" for none), and writes path + SignatureSuffix. keyPath may start with ~.
// keyID defaults to KeyIDFromPath(keyPath).
// changed is false when the signature file already had this content (ed25519
// signatures are deterministic, so only for the same content, key and
// expiry and release), in which case it isn't rewritten.
func SignFile(fsys afero.Fs, keyPath, keyID, release, path string, expires time.Time) (sig Signature, changed bool, err error) {
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
	if sig, err = Sign(key, keyID, release, expires, data); err != nil {
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

// Describe says what SignFile did for path, as `starcm sign` and sign() print
// it.
func Describe(path string, sig Signature) string {
	var release string
	if sig.Release != "" {
		release = fmt.Sprintf(" for release %q", sig.Release)
	}
	return fmt.Sprintf("signed %s with key %q%s until %s -> %s",
		path, sig.KeyID, release, formatTime(sig.Expires), path+SignatureSuffix)
}

// Keyring holds the public keys a run trusts, by key ID.
type Keyring struct {
	keys map[string]crypto.PublicKey
	// maxExpiry, when set, refuses signatures that expire further than this
	// from now.
	maxExpiry time.Duration
	// release, when set, refuses signatures for any other release, or none.
	release string
	now     func() time.Time
}

// SetRelease makes the keyring refuse signatures that weren't made for
// release. "" accepts any release, or none.
func (k *Keyring) SetRelease(release string) {
	k.release = release
}

// SetMaxExpiry makes the keyring refuse signatures that expire more than d
// from now. Zero allows any expiry.
func (k *Keyring) SetMaxExpiry(d time.Duration) {
	k.maxExpiry = d
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
	return &Keyring{keys: keys, now: time.Now}, nil
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
// returns the parsed signature. A signature past its expiry, or one expiring
// further out than SetMaxExpiry allows, is refused.
func (k *Keyring) Verify(data, sigFile []byte) (Signature, error) {
	sig, err := ParseSignature(sigFile)
	if err != nil {
		return Signature{}, err
	}
	sum, _ := starcmsha256.FromReader(bytes.NewReader(data)) // reading memory can't fail
	return sig, k.verify(sig, sum)
}

// VerifyDigest checks sigFile against the sha256 it records, without the
// file, and returns the parsed signature. Its SHA256 is then a sha256 a
// trusted key vouches for, to pin a download to before fetching it.
func (k *Keyring) VerifyDigest(sigFile []byte) (Signature, error) {
	sig, err := ParseSignature(sigFile)
	if err != nil {
		return Signature{}, err
	}
	return sig, k.verify(sig, sig.SHA256)
}

// verify checks that sig is a valid, unexpired signature by a trusted key
// over a file whose sha256 is sum.
func (k *Keyring) verify(sig Signature, sum string) error {
	v, ok := Verifiers[sig.Algorithm]
	if !ok {
		return fmt.Errorf("unsupported signature algorithm %q", sig.Algorithm)
	}
	pub, ok := k.keys[sig.KeyID]
	if !ok {
		return fmt.Errorf("key %q is not trusted", sig.KeyID)
	}
	if err := v.Verify(pub, digestMessage(sig.Algorithm, sig.KeyID, sig.Expires, sig.Release, sum), sig.Value); err != nil {
		return fmt.Errorf("key %q: %w", sig.KeyID, err)
	}
	// The signature covers sum, not the sha256 written in the file, so a
	// mismatch here means only that field was edited.
	if sig.SHA256 != sum {
		return fmt.Errorf("key %q: signature file records sha256 %s, but the signature is for %s", sig.KeyID, sig.SHA256, sum)
	}
	// Checked after the signature, so the release and expiry are known to be
	// the signer's.
	if k.release != "" && sig.Release != k.release {
		if sig.Release == "" {
			return fmt.Errorf("key %q: signature has no release, want release %q", sig.KeyID, k.release)
		}
		return fmt.Errorf("key %q: signature is for release %q, want release %q", sig.KeyID, sig.Release, k.release)
	}
	now := k.now().UTC().Truncate(time.Second)
	if !now.Before(sig.Expires) {
		return fmt.Errorf("key %q: signature expired at %s (now %s)", sig.KeyID, formatTime(sig.Expires), formatTime(now))
	}
	if k.maxExpiry > 0 && sig.Expires.After(now.Add(k.maxExpiry)) {
		return fmt.Errorf("key %q: signature expires at %s, more than %s from now (%s)",
			sig.KeyID, formatTime(sig.Expires), formatDuration(k.maxExpiry), formatTime(now))
	}
	return nil
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
