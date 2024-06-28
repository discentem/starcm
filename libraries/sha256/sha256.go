package sha256

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
)

// FromReader generates a sha256 hash from an io.Reader
func FromReader(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", fmt.Errorf("%v: %s", err, "could not generate sha256 due to io.Copy error")
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
