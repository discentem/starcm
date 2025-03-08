package fileutils

import (
	"github.com/spf13/afero"
)

func IsDir(fsys afero.Fs, path string) (bool, error) {
	fsinfo, err := fsys.Stat(path)
	if err != nil {
		return false, err
	}
	return fsinfo.IsDir(), nil
}
