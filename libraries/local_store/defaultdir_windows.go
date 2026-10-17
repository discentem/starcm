//go:build windows

package localstore

import (
	"errors"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// DefaultDir is the store directory used when --store-dir isn't given:
// %ProgramData% when running elevated (including as SYSTEM, as in WinPE),
// otherwise %LocalAppData%.
func DefaultDir() (string, error) {
	if windows.GetCurrentProcessToken().IsElevated() {
		programData := os.Getenv("ProgramData")
		if programData == "" {
			return "", errors.New("%ProgramData% is not set; pass --store-dir")
		}
		return filepath.Join(programData, "starcm", "store"), nil
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cache, "starcm", "store"), nil
}
