// Package localstore keeps downloaded files on disk under their sha256, at
// <dir>/sha256/<hash>/<name>, in a directory that only the current user (and
// root or administrators) can change.
package localstore

import (
	"bytes"
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

// PutSignature writes sig as the signature file of the stored file with
// this hash and name, <name>.sig next to it, so checks that look for a
// file's signature beside it (--trusted-keys loading a stored .star file)
// find it. A different signature already there is replaced, as happens
// after re-signing; changed is false when it was already there.
func (s *Store) PutSignature(hash, name string, sig []byte) (path string, changed bool, err error) {
	stored, err := s.Path(hash, name)
	if err != nil {
		return "", false, err
	}
	path = stored + ".sig"
	if old, err := os.ReadFile(path); err == nil && bytes.Equal(old, sig) {
		return path, false, nil
	}
	dir := filepath.Dir(path)
	if err := s.perms.Check(dir); err != nil {
		return "", false, fmt.Errorf("%s is not safe to use: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, "."+name+".sig.tmp-")
	if err != nil {
		return "", false, err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once renamed
	if _, err := tmp.Write(sig); err != nil {
		tmp.Close()
		return "", false, err
	}
	if err := tmp.Close(); err != nil {
		return "", false, err
	}
	if err := s.perms.LockFile(tmpName); err != nil {
		return "", false, err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return "", false, err
	}
	return path, true, nil
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return starcmsha256.FromReader(f)
}

// CheckIfStored returns a function that verifies path if it names something
// inside the store at dir, and does nothing otherwise. A path inside the
// store must be a stored file, <dir>/sha256/<hash>/<name>; it's verified
// with Get, so its permissions are re-checked and it's re-hashed against the
// hash in its own path. open is only called for paths inside the store.
//
// Paths are compared against dir both as given and with symlinks resolved,
// so a symlink into the store is caught too.
func CheckIfStored(dir string, open func() (*Store, error)) func(path string) error {
	return func(path string) error {
		if path == "" {
			return nil
		}
		rel, inside, err := relToStore(dir, path)
		if err != nil || !inside {
			return err
		}
		if rel == "." {
			return nil // the store directory itself, not a file in it
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) != 3 || parts[0] != "sha256" {
			return fmt.Errorf("%s is inside the store at %s but isn't a stored file", path, dir)
		}
		store, err := open()
		if err != nil {
			return err
		}
		_, ok, err := store.Get(parts[1], parts[2])
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%s is not in the store", path)
		}
		return nil
	}
}

// relToStore returns path relative to the store at dir, if it's inside it.
func relToStore(dir, path string) (string, bool, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", false, err
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", false, err
	}
	dirs := []string{absDir}
	if real, err := filepath.EvalSymlinks(absDir); err == nil && real != absDir {
		dirs = append(dirs, real)
	}
	paths := []string{absPath}
	if real, err := filepath.EvalSymlinks(absPath); err == nil && real != absPath {
		paths = append(paths, real)
	}
	for _, d := range dirs {
		for _, p := range paths {
			rel, err := filepath.Rel(d, p)
			if err != nil {
				continue // e.g. a different drive on Windows
			}
			if rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return rel, true, nil
			}
		}
	}
	return "", false, nil
}
