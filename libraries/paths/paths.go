// Package paths implements starcm's rule for path arguments: a relative path
// only ever names a read-only resource that ships next to a .star file, and
// every other path must be absolute. A leading ~ is expanded everywhere. See
// docs/design/paths-and-arguments.md.
package paths

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
	"go.starlark.net/starlark"
)

// Expand expands a leading ~ to the user's home directory.
func Expand(p string) (string, error) {
	return homedir.Expand(p)
}

// Abs returns p cleaned, with a leading ~ expanded. A relative path is an
// error: outputs and inputs that aren't bundled resources must say exactly
// where they are.
func Abs(p string) (string, error) {
	expanded, err := Expand(p)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(expanded) {
		return "", fmt.Errorf("%q is not an absolute path: write the full path, or pass the result of the step that made the file", p)
	}
	return filepath.Clean(expanded), nil
}

// Resource resolves p as a read-only resource bundled with the code: a
// relative p is joined onto dir, the directory of the .star file the call is
// written in. Absolute paths and ~ paths are used as given.
func Resource(dir, p string) (string, error) {
	expanded, err := Expand(p)
	if err != nil {
		return "", err
	}
	if filepath.IsAbs(expanded) {
		return filepath.Clean(expanded), nil
	}
	if dir == "" {
		return "", fmt.Errorf("can't resolve %q: no .star file to resolve it next to", p)
	}
	return filepath.Join(dir, expanded), nil
}

// CallerFile returns the absolute path of the .star file containing the call
// to the builtin that is running on thread. It's the file the call is
// written in, so a call inside a function from another module reports that
// module's file, the way load() resolves paths.
func CallerFile(thread *starlark.Thread) (string, bool) {
	if thread == nil {
		return "", false
	}
	stack := thread.CallStack()
	if len(stack) < 2 {
		return "", false
	}
	name := stack.At(1).Pos.Filename()
	if name == "" {
		return "", false
	}
	abs, err := filepath.Abs(name)
	if err != nil {
		return "", false
	}
	return abs, true
}

// CallerDir returns the directory of CallerFile.
func CallerDir(thread *starlark.Thread) (string, bool) {
	file, ok := CallerFile(thread)
	if !ok {
		return "", false
	}
	return filepath.Dir(file), true
}

// Origin says where a .star file runs from, for the rule that relative paths
// name files that ship next to it.
type Origin struct {
	// Store is set for a file that runs from starcm's store, as modules
	// loaded with load_remote do.
	Store bool
	// Root, for a file in a folder load_remote fetched (a store folder whose
	// manifest was checked), is that folder's directory. Files ship next to
	// such a file, so relative paths work, but only inside Root. Empty for a
	// single file in the store, next to which nothing ships.
	Root string
}

// Shipped reports whether files can ship next to a .star file from o: it's
// not in the store, or it's in a folder fetched from a manifest.
func (o Origin) Shipped() bool { return !o.Store || o.Root != "" }

// Resolve joins rel onto dir, the directory of a .star file from o, and
// returns the result cleaned. For a file in a fetched folder the result must
// stay inside the folder: ".." past its root is an error.
func (o Origin) Resolve(dir, rel string) (string, error) {
	p := filepath.Join(dir, rel)
	if o.Root != "" && !Within(o.Root, p) {
		return "", fmt.Errorf("%q is outside the folder %s that this .star file was fetched in with load_remote", rel, o.Root)
	}
	return p, nil
}

// Within reports whether p is root or inside it, comparing the cleaned paths.
func Within(root, p string) bool {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

// storeCheckKey is the thread-local key SetStoreCheck records under.
const storeCheckKey = "starcm.paths.origin"

// StoreCheck tells where a .star file runs from.
type StoreCheck func(path string) Origin

// SetStoreCheck records on thread how to tell where a .star file runs from.
// The loader sets it on every thread it runs code on.
func SetStoreCheck(thread *starlark.Thread, check StoreCheck) {
	thread.SetLocal(storeCheckKey, check)
}

// OriginOf reports where file runs from, by the check SetStoreCheck recorded
// on thread. Without one it reports a file outside the store.
func OriginOf(thread *starlark.Thread, file string) Origin {
	if thread == nil {
		return Origin{}
	}
	check, ok := thread.Local(storeCheckKey).(StoreCheck)
	if !ok || check == nil {
		return Origin{}
	}
	return check(file)
}
