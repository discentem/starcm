package fileutils

import (
	"io"
	"os"

	starcmsha256 "github.com/discentem/starcm/libraries/sha256"
	"github.com/spf13/afero"
)

func IsDir(fsys afero.Fs, path string) (bool, error) {
	fsinfo, err := fsys.Stat(path)
	if err != nil {
		return false, err
	}
	return fsinfo.IsDir(), nil
}

// CopyNew copies src to dst, which must not exist yet, creating it with
// perm, and returns the sha256 of what it wrote. Copying into a private
// directory gives a copy that can be checked and then used without anyone
// swapping it in between.
func CopyNew(src, dst string, perm os.FileMode) (string, error) {
	in, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return "", err
	}
	h := starcmsha256.New()
	if _, err := io.Copy(io.MultiWriter(out, h), in); err != nil {
		out.Close()
		return "", err
	}
	if err := out.Close(); err != nil {
		return "", err
	}
	return starcmsha256.Hex(h), nil
}
