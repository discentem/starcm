// Package sha256 is starcm's one implementation of sha256 hashing, shared by
// download() and sha256().
package sha256

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"

	"github.com/spf13/afero"
)

// New returns a hash to stream data into, e.g. with io.MultiWriter while
// downloading. Hex formats its sum.
func New() hash.Hash { return sha256.New() }

// Hex returns h's sum as lowercase hex, the format download(sha256 = ...)
// and sha256() use.
func Hex(h hash.Hash) string { return hex.EncodeToString(h.Sum(nil)) }

// FromReader generates a sha256 hash from an io.Reader
func FromReader(r io.Reader) (string, error) {
	h := New()
	if _, err := io.Copy(h, r); err != nil {
		return "", fmt.Errorf("%v: %s", err, "could not generate sha256 due to io.Copy error")
	}
	return Hex(h), nil
}

// File returns the sha256 of the file at path.
func File(fsys afero.Fs, path string) (string, error) {
	f, err := fsys.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return Hex(h), nil
}
