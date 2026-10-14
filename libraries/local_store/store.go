// Package localstore keeps downloaded files on disk under their sha256, at
// <dir>/sha256/<hash>/<name>, in a directory that only the current user (and
// root or administrators) can change.
package localstore

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	starcmsha256 "github.com/discentem/starcm/libraries/sha256"
)

// Permissions creates and checks store directories. Each platform has its own
// implementation; NewPermissions returns the one for this platform.
type Permissions interface {
	// MkdirAll creates dir and any missing parents. Each directory it creates
	// can be changed only by the current user, root and administrators.
	// Directories that already exist are left alone; Check decides whether
	// they can be trusted.
	MkdirAll(dir string) error
	// LockFile restricts a file just written into the store so that only the
	// current user, root and administrators can change it.
	LockFile(path string) error
	// Check returns an error if path, or any directory above it, could be
	// changed by anyone other than the current user, root or administrators.
	Check(path string) error
}

// Store is an opened, checked store directory.
type Store struct {
	dir   string
	perms Permissions
}

// Open creates dir if needed and checks that it can be trusted. dir is
// resolved through any symlinks, so later operations can't be redirected by
// changing a symlink after the check.
func Open(dir string, perms Permissions) (*Store, error) {
	if dir == "" {
		return nil, errors.New("store directory must not be empty")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if err := perms.MkdirAll(abs); err != nil {
		return nil, fmt.Errorf("creating store %s: %w", abs, err)
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, err
	}
	if err := perms.Check(real); err != nil {
		return nil, fmt.Errorf("store %s is not safe to use: %w", real, err)
	}
	return &Store{dir: real, perms: perms}, nil
}

// Lazy returns a function that opens the store the first time it's called
// and returns the same result after that, so runs that never use the store
// never create it.
func Lazy(dir string, perms Permissions) func() (*Store, error) {
	return sync.OnceValues(func() (*Store, error) {
		return Open(dir, perms)
	})
}

// Dir is the store's directory, with symlinks resolved.
func (s *Store) Dir() string { return s.dir }

var hashPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Path is where the file with this hash and name lives in the store.
func (s *Store) Path(hash, name string) (string, error) {
	hash = strings.ToLower(hash)
	if !hashPattern.MatchString(hash) {
		return "", fmt.Errorf("sha256 %q is not 64 hex characters", hash)
	}
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
		return "", fmt.Errorf("file name %q must be a plain file name", name)
	}
	return filepath.Join(s.dir, "sha256", hash, name), nil
}

// Get returns the path of the stored file with this hash and name, and
// whether it's there. A stored file is re-checked and re-hashed on every Get;
// one that fails either is an error, not a miss, since files only reach the
// store after their hash matched.
func (s *Store) Get(hash, name string) (string, bool, error) {
	path, err := s.Path(hash, name)
	if err != nil {
		return "", false, err
	}
	fi, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return path, false, nil
	}
	if err != nil {
		return "", false, err
	}
	if !fi.Mode().IsRegular() {
		return "", false, fmt.Errorf("%s in the store is not a regular file", path)
	}
	if err := s.perms.Check(path); err != nil {
		return "", false, fmt.Errorf("%s is not safe to use: %w", path, err)
	}
	got, err := hashFile(path)
	if err != nil {
		return "", false, err
	}
	if got != strings.ToLower(hash) {
		return "", false, fmt.Errorf("%s in the store has sha256 %s; it was changed after it was stored", path, got)
	}
	return path, true, nil
}

// Put writes r into the store under hash and name. The data goes to a
// temporary file next to its final path and is renamed into place only once
// its sha256 matches, so the final path never holds unverified data.
func (s *Store) Put(hash, name string, r io.Reader) (string, error) {
	path, err := s.Path(hash, name)
	if err != nil {
		return "", err
	}
	hash = strings.ToLower(hash)
	dir := filepath.Dir(path)
	if err := s.perms.MkdirAll(dir); err != nil {
		return "", err
	}
	if err := s.perms.Check(dir); err != nil {
		return "", fmt.Errorf("%s is not safe to use: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, "."+name+".tmp-")
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	// Every path below closes tmp before this runs, and before the rename:
	// Windows can't delete or rename a file that is still open.
	defer os.Remove(tmpName) // no-op once renamed

	hasher := starcmsha256.New()
	if _, err := io.Copy(io.MultiWriter(tmp, hasher), r); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	if got := starcmsha256.Hex(hasher); got != hash {
		return "", fmt.Errorf("expected sha256 hash %s, got %s", hash, got)
	}
	if err := s.perms.LockFile(tmpName); err != nil {
		return "", err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return "", err
	}
	return path, nil
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return starcmsha256.FromReader(f)
}
