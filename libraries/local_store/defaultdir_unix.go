//go:build !windows

package localstore

import (
	"os"
	"path/filepath"
	"runtime"
)

// DefaultDir is the store directory used when --store-dir isn't given: a
// system directory when running as root, otherwise the user's cache directory.
func DefaultDir() (string, error) {
	if os.Geteuid() == 0 {
		if runtime.GOOS == "darwin" {
			return "/Library/Application Support/starcm/store", nil
		}
		return "/var/lib/starcm/store", nil
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cache, "starcm", "store"), nil
}
