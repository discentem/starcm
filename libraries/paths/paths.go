// Package paths implements starcm's rule for path arguments: a relative path
// only ever names a read-only resource that ships next to a .star file, and
// every other path must be absolute. A leading ~ is expanded everywhere. See
// docs/design/paths-and-arguments.md.
package paths

import (
	"fmt"
	"path/filepath"

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

// storeCheckKey is the thread-local key SetStoreCheck records under.
const storeCheckKey = "starcm.paths.inStore"

// SetStoreCheck records on thread how to tell whether a .star file runs from
// starcm's store, as modules loaded with load_remote do. The loader sets it
// on every thread it runs code on.
func SetStoreCheck(thread *starlark.Thread, inStore func(path string) bool) {
	thread.SetLocal(storeCheckKey, inStore)
}

// InStore reports whether file runs from starcm's store, by the check
// SetStoreCheck recorded on thread. Without one it reports false.
func InStore(thread *starlark.Thread, file string) bool {
	if thread == nil {
		return false
	}
	inStore, ok := thread.Local(storeCheckKey).(func(string) bool)
	return ok && inStore(file)
}
