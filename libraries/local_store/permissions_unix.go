//go:build !windows

package localstore

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
)

// unixPermissions trusts files owned by root or the current user that no
// group or other user can write. A world-writable directory above the store
// is allowed only if it's sticky (like /tmp), since then nobody else can
// rename or remove what's in it.
//
// POSIX and macOS ACLs aren't checked.
type unixPermissions struct {
	uid uint32
}

// NewPermissions returns the Permissions for this platform.
func NewPermissions() (Permissions, error) {
	return unixPermissions{uid: uint32(os.Geteuid())}, nil
}

func (p unixPermissions) MkdirAll(dir string) error {
	dir = filepath.Clean(dir)
	if fi, err := os.Stat(dir); err == nil {
		if !fi.IsDir() {
			return fmt.Errorf("%s exists and is not a directory", dir)
		}
		return nil
	}
	if parent := filepath.Dir(dir); parent != dir {
		if err := p.MkdirAll(parent); err != nil {
			return err
		}
	}
	if err := os.Mkdir(dir, 0o755); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return nil // created concurrently; Check decides whether to trust it
		}
		return err
	}
	// Mkdir's mode is filtered by the umask; set it exactly.
	return os.Chmod(dir, 0o755)
}

func (p unixPermissions) LockFile(path string) error {
	return os.Chmod(path, 0o555)
}

func (p unixPermissions) Check(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	// Check the real path. Symlinks along the way are fine as long as what
	// they point to passes; callers use the resolved path afterwards.
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return err
	}
	for cur := real; ; {
		fi, err := os.Lstat(cur)
		if err != nil {
			return err
		}
		st, ok := fi.Sys().(*syscall.Stat_t)
		if !ok {
			return fmt.Errorf("can't read the owner of %s on %s", cur, runtime.GOOS)
		}
		if st.Uid != 0 && st.Uid != p.uid {
			return fmt.Errorf("%s is owned by uid %d; want root or uid %d", cur, st.Uid, p.uid)
		}
		if fi.Mode().Perm()&0o022 != 0 {
			stickyParent := cur != real && fi.IsDir() && fi.Mode()&fs.ModeSticky != 0
			if !stickyParent {
				return fmt.Errorf("%s is writable by group or others (mode %v)", cur, fi.Mode())
			}
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return nil
		}
		cur = parent
	}
}
