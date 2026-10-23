package signing

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	starcmsha256 "github.com/discentem/starcm/libraries/sha256"
	"github.com/spf13/afero"
)

// ManifestName is the file `starcm sign --dir` writes into a folder, listing
// every file in it with its sha256. Its signature is ManifestName +
// SignatureSuffix, next to it.
const ManifestName = "starcm-manifest.json"

// ManifestVersion is the only manifest version starcm reads and writes.
const ManifestVersion = 1

// MaxManifestSize bounds a manifest starcm will read.
const MaxManifestSize = 16 << 20

// Manifest lists the files of a folder: each file's path relative to the
// folder, slash separated, and its sha256 in lowercase hex. One signature on
// the manifest covers every file it lists.
//
//	{
//	  "version": 1,
//	  "files": {
//	    "main.star": "4aad939f…",
//	    "templates/motd.jinja": "5e5d7344…"
//	  }
//	}
type Manifest struct {
	Version int               `json:"version"`
	Files   map[string]string `json:"files"`
}

// Paths returns the manifest's file paths, sorted.
func (m Manifest) Paths() []string {
	ps := make([]string, 0, len(m.Files))
	for p := range m.Files {
		ps = append(ps, p)
	}
	sort.Strings(ps)
	return ps
}

// Bytes formats m as a manifest file. The output is stable: the same files
// always give the same bytes, and so the same sha256.
func (m Manifest) Bytes() []byte {
	// encoding/json sorts map keys.
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		panic(err) // a struct of strings and an int always marshals
	}
	return append(b, '\n')
}

// ParseManifest parses and checks a manifest file: the version must be
// ManifestVersion, every path must be a plain relative path inside the folder
// (see CheckManifestPath) and every sha256 64 lowercase hex characters.
func ParseManifest(b []byte) (Manifest, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	var m Manifest
	if err := dec.Decode(&m); err != nil {
		return Manifest{}, fmt.Errorf("malformed manifest: %w", err)
	}
	if dec.More() {
		return Manifest{}, errors.New("malformed manifest: data after the JSON object")
	}
	if m.Version != ManifestVersion {
		return Manifest{}, fmt.Errorf("manifest version %d is not supported, want %d", m.Version, ManifestVersion)
	}
	if m.Files == nil {
		return Manifest{}, errors.New(`malformed manifest: no "files"`)
	}
	// Paths that differ only in case would be one file on macOS and Windows,
	// and a path that is also another's directory can't be laid out at all.
	folded := map[string]string{}
	for _, p := range m.Paths() {
		if err := CheckManifestPath(p); err != nil {
			return Manifest{}, err
		}
		if !sha256Hex.MatchString(m.Files[p]) {
			return Manifest{}, fmt.Errorf("manifest: %s: sha256 %q is not 64 lowercase hex characters", p, m.Files[p])
		}
		key := strings.ToLower(p)
		if other, ok := folded[key]; ok {
			return Manifest{}, fmt.Errorf("manifest lists both %s and %s, which differ only in case", other, p)
		}
		folded[key] = p
	}
	for key, p := range folded {
		for dir := path.Dir(key); dir != "."; dir = path.Dir(dir) {
			if other, ok := folded[dir]; ok {
				return Manifest{}, fmt.Errorf("manifest lists %s as a file and as the directory of %s", other, p)
			}
		}
	}
	return m, nil
}

// CheckManifestPath returns an error unless p is a path a manifest may list:
// relative, slash separated, without empty, "." or ".." parts, backslashes,
// colons or NUL, and not the manifest or its signature.
func CheckManifestPath(p string) error {
	if p == "" {
		return errors.New("manifest: empty path")
	}
	if strings.ContainsAny(p, "\\:\x00") {
		return fmt.Errorf("manifest: %q: paths are slash separated and can't contain \\, : or NUL", p)
	}
	if strings.HasPrefix(p, "/") {
		return fmt.Errorf("manifest: %q is absolute; paths are relative to the folder", p)
	}
	for _, part := range strings.Split(p, "/") {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("manifest: %q is not a clean path inside the folder", p)
		}
	}
	if p == ManifestName || p == ManifestName+SignatureSuffix {
		return fmt.Errorf("manifest: %q can't list itself or its signature", p)
	}
	return nil
}

// ManifestOf builds the manifest of the folder dir: every regular file in
// it, recursively. Files and directories whose names start with "." are
// left out, as are the manifest and its signature. Anything that isn't a
// regular file or directory, such as a symlink, is an error.
func ManifestOf(fsys afero.Fs, dir string) (Manifest, error) {
	m := Manifest{Version: ManifestVersion, Files: map[string]string{}}
	err := afero.Walk(fsys, dir, func(p string, fi fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(fi.Name(), ".") {
			if fi.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if fi.IsDir() {
			return nil
		}
		if rel == ManifestName || rel == ManifestName+SignatureSuffix {
			return nil
		}
		if !fi.Mode().IsRegular() {
			return fmt.Errorf("%s is not a regular file", p)
		}
		if err := CheckManifestPath(rel); err != nil {
			return err
		}
		sum, err := starcmsha256.File(fsys, p)
		if err != nil {
			return err
		}
		m.Files[rel] = sum
		return nil
	})
	if err != nil {
		return Manifest{}, err
	}
	// Re-parse, so a folder that can't be described (two files differing only
	// in case on a case-sensitive file system) fails here, not on machines.
	if _, err := ParseManifest(m.Bytes()); err != nil {
		return Manifest{}, fmt.Errorf("%s: %w", dir, err)
	}
	return m, nil
}

// Check returns an error unless every file m lists is in dir with its
// sha256.
func (m Manifest) Check(fsys afero.Fs, dir string) error {
	for _, p := range m.Paths() {
		got, err := starcmsha256.File(fsys, filepath.Join(dir, filepath.FromSlash(p)))
		if err != nil {
			return err
		}
		if got != m.Files[p] {
			return fmt.Errorf("%s has sha256 %s, but the manifest lists %s", p, got, m.Files[p])
		}
	}
	return nil
}

// SignDir writes dir's manifest (see ManifestOf) to dir/ManifestName and
// signs it as SignFile does. The manifest is only rewritten when it
// changed; changed reports whether either file was written.
func SignDir(fsys afero.Fs, keyPath, keyID, release, dir string, expires time.Time) (m Manifest, sig Signature, changed bool, err error) {
	if m, err = ManifestOf(fsys, dir); err != nil {
		return Manifest{}, Signature{}, false, err
	}
	manifestPath := filepath.Join(dir, ManifestName)
	if old, err := afero.ReadFile(fsys, manifestPath); err != nil || !bytes.Equal(old, m.Bytes()) {
		if err := afero.WriteFile(fsys, manifestPath, m.Bytes(), 0o644); err != nil {
			return Manifest{}, Signature{}, false, err
		}
		changed = true
	}
	sig, sigChanged, err := SignFile(fsys, keyPath, keyID, release, manifestPath, expires)
	if err != nil {
		return Manifest{}, Signature{}, false, err
	}
	return m, sig, changed || sigChanged, nil
}
