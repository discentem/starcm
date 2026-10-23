package localstore

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	starcmsha256 "github.com/discentem/starcm/libraries/sha256"
	"github.com/discentem/starcm/libraries/signing"
)

// foldersDir holds assembled folders, <store>/folders/<manifest hash>/, next
// to <store>/sha256/.
const foldersDir = "folders"

// Folder is a folder assembled in the store from a manifest: each file it
// lists, laid out under Dir by its path in the manifest.
type Folder struct {
	// Dir is <store>/folders/<manifest hash>.
	Dir      string
	Manifest signing.Manifest
}

// FolderPath is where the folder whose manifest has this sha256 lives in
// the store.
func (s *Store) FolderPath(manifestHash string) (string, error) {
	manifestHash = strings.ToLower(manifestHash)
	if !hashPattern.MatchString(manifestHash) {
		return "", fmt.Errorf("sha256 %q is not 64 hex characters", manifestHash)
	}
	return filepath.Join(s.dir, foldersDir, manifestHash), nil
}

// GetFolder returns the folder assembled from the manifest with this sha256,
// and whether it's there. Like Get, it re-checks what's stored every time:
// the folder's permissions, its manifest's sha256, and every file's sha256
// against the manifest. A folder holding anything the manifest doesn't list
// (other than the manifest and its signature) or missing a file it lists is
// an error, not a miss, since folders are only assembled from checked files.
func (s *Store) GetFolder(manifestHash string) (Folder, bool, error) {
	dir, err := s.FolderPath(manifestHash)
	if err != nil {
		return Folder{}, false, err
	}
	fi, err := os.Lstat(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return Folder{}, false, nil
	}
	if err != nil {
		return Folder{}, false, err
	}
	if !fi.IsDir() {
		return Folder{}, false, fmt.Errorf("%s in the store is not a directory", dir)
	}
	if err := s.perms.Check(dir); err != nil {
		return Folder{}, false, fmt.Errorf("%s is not safe to use: %w", dir, err)
	}
	m, err := readFolderManifest(dir, manifestHash)
	if err != nil {
		return Folder{}, false, err
	}
	changed := func(format string, args ...any) error {
		return fmt.Errorf("folder %s in the store was changed after it was stored: %s", dir, fmt.Sprintf(format, args...))
	}
	seen := map[string]bool{}
	err = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return changed("%s is not a regular file", rel)
		}
		if rel == signing.ManifestName || rel == signing.ManifestName+signing.SignatureSuffix {
			return nil
		}
		want, listed := m.Files[rel]
		if !listed {
			return changed("%s isn't in its manifest", rel)
		}
		got, err := hashFile(p)
		if err != nil {
			return err
		}
		if got != want {
			return changed("%s has sha256 %s, but the manifest lists %s", rel, got, want)
		}
		seen[rel] = true
		return nil
	})
	if err != nil {
		return Folder{}, false, err
	}
	for _, p := range m.Paths() {
		if !seen[p] {
			return Folder{}, false, changed("%s is missing", p)
		}
	}
	return Folder{Dir: dir, Manifest: m}, true, nil
}

// readFolderManifest reads and parses dir's manifest, which must have
// sha256 manifestHash.
func readFolderManifest(dir, manifestHash string) (signing.Manifest, error) {
	b, err := readLimited(filepath.Join(dir, signing.ManifestName), signing.MaxManifestSize)
	if err != nil {
		return signing.Manifest{}, err
	}
	got, _ := starcmsha256.FromReader(bytes.NewReader(b)) // reading memory can't fail
	if got != strings.ToLower(manifestHash) {
		return signing.Manifest{}, fmt.Errorf("%s in the store has sha256 %s; it was changed after it was stored", filepath.Join(dir, signing.ManifestName), got)
	}
	return signing.ParseManifest(b)
}

func readLimited(path string, limit int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("%s is larger than %d bytes", path, limit)
	}
	return b, nil
}

// FolderFileName is the name a file listed in a manifest is stored under in
// <store>/sha256/<hash>/: the last part of its path.
func FolderFileName(manifestPath string) string {
	return path.Base(manifestPath)
}

// PutFolder assembles the folder for manifest, whose sha256 must be
// manifestHash, under FolderPath(manifestHash). Every file it lists must
// already be stored (Put) under its hash and FolderFileName; each is hard
// linked into the folder, or copied where links don't work. sig, when not
// nil, is the manifest's signature, kept next to it in the folder as
// <manifest>.sig so --trusted-keys can trust the folder's .star files.
//
// The folder is built in a temporary directory and renamed into place, so
// FolderPath never holds a partial folder. A folder already there is
// re-checked with GetFolder; only its signature is replaced, when it
// differs. changed reports whether anything was written.
func (s *Store) PutFolder(manifestHash string, manifest, sig []byte) (f Folder, changed bool, err error) {
	manifestHash = strings.ToLower(manifestHash)
	if got, _ := starcmsha256.FromReader(bytes.NewReader(manifest)); got != manifestHash {
		return Folder{}, false, fmt.Errorf("expected manifest sha256 hash %s, got %s", manifestHash, got)
	}
	m, err := signing.ParseManifest(manifest)
	if err != nil {
		return Folder{}, false, err
	}
	if f, ok, err := s.GetFolder(manifestHash); err != nil {
		return Folder{}, false, err
	} else if ok {
		changed, err := s.putFolderSignature(f.Dir, sig)
		return f, changed, err
	}

	final, err := s.FolderPath(manifestHash)
	if err != nil {
		return Folder{}, false, err
	}
	parent := filepath.Dir(final)
	if err := s.perms.MkdirAll(parent); err != nil {
		return Folder{}, false, err
	}
	if err := s.perms.Check(parent); err != nil {
		return Folder{}, false, fmt.Errorf("%s is not safe to use: %w", parent, err)
	}
	tmp, err := os.MkdirTemp(parent, "."+manifestHash+".tmp-")
	if err != nil {
		return Folder{}, false, err
	}
	defer removeAll(tmp) // no-op once renamed
	// MkdirTemp makes the directory 0700; give it the store's usual mode.
	if err := os.Chmod(tmp, 0o755); err != nil {
		return Folder{}, false, err
	}
	for _, p := range m.Paths() {
		src, ok, err := s.Get(m.Files[p], FolderFileName(p))
		if err != nil {
			return Folder{}, false, err
		}
		if !ok {
			return Folder{}, false, fmt.Errorf("can't assemble folder %s: %s (sha256 %s) is not in the store", final, p, m.Files[p])
		}
		dst := filepath.Join(tmp, filepath.FromSlash(p))
		if err := s.perms.MkdirAll(filepath.Dir(dst)); err != nil {
			return Folder{}, false, err
		}
		if err := s.linkOrCopy(src, dst); err != nil {
			return Folder{}, false, err
		}
	}
	if err := s.writeLocked(filepath.Join(tmp, signing.ManifestName), manifest); err != nil {
		return Folder{}, false, err
	}
	if sig != nil {
		if err := s.writeLocked(filepath.Join(tmp, signing.ManifestName+signing.SignatureSuffix), sig); err != nil {
			return Folder{}, false, err
		}
	}
	if err := os.Rename(tmp, final); err != nil {
		// Another run may have assembled it first.
		if f, ok, gerr := s.GetFolder(manifestHash); gerr == nil && ok {
			changed, err := s.putFolderSignature(f.Dir, sig)
			return f, changed, err
		}
		return Folder{}, false, err
	}
	f, ok, err := s.GetFolder(manifestHash)
	if err != nil {
		return Folder{}, false, err
	}
	if !ok {
		return Folder{}, false, fmt.Errorf("folder %s vanished after it was assembled", final)
	}
	return f, true, nil
}

// PutFolderSignature keeps sig, when it's set, as the signature of f's
// manifest, replacing a different one, as happens after re-signing. changed
// reports whether it was written.
func (s *Store) PutFolderSignature(f Folder, sig []byte) (changed bool, err error) {
	return s.putFolderSignature(f.Dir, sig)
}

// putFolderSignature replaces the signature of the folder at dir's manifest
// with sig, when sig is set and differs, as happens after re-signing.
func (s *Store) putFolderSignature(dir string, sig []byte) (bool, error) {
	if sig == nil {
		return false, nil
	}
	p := filepath.Join(dir, signing.ManifestName+signing.SignatureSuffix)
	if old, err := os.ReadFile(p); err == nil && bytes.Equal(old, sig) {
		return false, nil
	}
	if err := s.perms.Check(dir); err != nil {
		return false, fmt.Errorf("%s is not safe to use: %w", dir, err)
	}
	if err := s.writeLocked(p, sig); err != nil {
		return false, err
	}
	return true, nil
}

// writeLocked writes data to path through a temporary file next to it,
// locks it and renames it into place, replacing what was there.
func (s *Store) writeLocked(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once renamed
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := s.perms.LockFile(tmpName); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// linkOrCopy hard links the stored file src to dst, or copies it and locks
// the copy where hard links aren't supported.
func (s *Store) linkOrCopy(src, dst string) error {
	if err := os.Link(src, dst); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return s.perms.LockFile(dst)
}

// removeAll removes a partly assembled folder. Stored files are read-only,
// which Windows won't delete, so it makes them writable first.
func removeAll(dir string) {
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			_ = os.Chmod(p, 0o644)
		}
		return nil
	})
	_ = os.RemoveAll(dir)
}

// InFolder reports whether path is inside a folder assembled in the store at
// dir, without opening or creating the store. It returns the folder's root
// (<dir>/folders/<manifest hash>), its manifest's sha256, and path relative
// to the root, slash separated ("." for the root itself).
func InFolder(dir, path string) (root, manifestHash, rel string, ok bool) {
	storeRel, base, inside, err := relToStoreDir(dir, path)
	if err != nil || !inside {
		return "", "", "", false
	}
	parts := strings.Split(filepath.ToSlash(storeRel), "/")
	if len(parts) < 2 || parts[0] != foldersDir || !hashPattern.MatchString(parts[1]) {
		return "", "", "", false
	}
	rel = "."
	if len(parts) > 2 {
		rel = strings.Join(parts[2:], "/")
	}
	return filepath.Join(base, foldersDir, parts[1]), parts[1], rel, true
}

// checkFolderFile verifies path, rel inside the store folder for
// manifestHash: the folder is re-checked with GetFolder, and path must be one
// of the files its manifest lists, or the manifest.
func checkFolderFile(store *Store, path, manifestHash, rel string) error {
	f, ok, err := store.GetFolder(manifestHash)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%s is not in the store", path)
	}
	if _, listed := f.Manifest.Files[rel]; !listed && rel != signing.ManifestName {
		return fmt.Errorf("%s is inside the store at %s but isn't a stored file", path, store.Dir())
	}
	return nil
}
