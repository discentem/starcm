package aferohelpers

import (
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/afero"
)

// FileDefinition represents a file to be added to the in-memory filesystem
type FileDefinition struct {
	Path    string
	Content string
	Mode    int
	ModTime time.Time
	IsDir   bool
}

// NewMemMapFsWithFiles creates a new in-memory filesystem with predefined files
func NewMemFsWithFiles(files ...FileDefinition) afero.Fs {
	memFs := afero.NewMemMapFs()

	for _, file := range files {
		path := filepath.Clean(file.Path)

		// Create parent directories if needed
		dir := filepath.Dir(path)
		if dir != "." && dir != "/" {
			memFs.MkdirAll(dir, 0755)
		}

		// Default mode to regular file if not specified
		mode := file.Mode
		if mode == 0 {
			if file.IsDir {
				mode = 0755
			} else {
				mode = 0644
			}
		}

		if file.IsDir {
			// Create directory
			memFs.Mkdir(path, os.FileMode(mode))
		} else {
			// Create file
			afero.WriteFile(memFs, path, []byte(file.Content), os.FileMode(mode))
		}

		// Set custom modification time if specified
		if !file.ModTime.IsZero() {
			memFs.Chtimes(path, file.ModTime, file.ModTime)
		}
	}

	return memFs
}
