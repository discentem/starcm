//go:build windows

package localstore

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

// File access rights (winnt.h) that let someone change a file or directory.
const (
	fileWriteData       = 0x2 // FILE_ADD_FILE on directories
	fileAppendData      = 0x4 // FILE_ADD_SUBDIRECTORY on directories
	fileWriteEA         = 0x10
	fileDeleteChild     = 0x40
	fileWriteAttributes = 0x100
	accessDelete        = 0x10000
	writeDAC            = 0x40000
	writeOwner          = 0x80000

	// Rights that let someone replace, remove or re-permission the object.
	// These matter on every directory above the store too.
	replaceRights = accessDelete | fileDeleteChild | writeDAC | writeOwner | windows.GENERIC_ALL
	// Rights that let someone change the object's contents.
	modifyRights = replaceRights | fileWriteData | fileAppendData | fileWriteEA | fileWriteAttributes | windows.GENERIC_WRITE
)

// Allow-ACE types (winnt.h) besides ACCESS_ALLOWED_ACE_TYPE. Their layout
// differs, so Check refuses rather than misreading them.
const (
	accessAllowedObjectACEType         = 0x5
	accessAllowedCallbackACEType       = 0x9
	accessAllowedCallbackObjectACEType = 0xB
)

// trustedInstaller is NT SERVICE\TrustedInstaller, which owns some system
// directories such as C:\.
const trustedInstaller = "S-1-5-80-956008885-3418522649-1831038044-1853292631-2271478464"

// windowsPermissions trusts objects owned by SYSTEM, Administrators,
// TrustedInstaller or the current user whose DACL grants nobody else the
// right to change them. Directories above the store only need to be safe
// from replacement: anyone may add entries to them, but not remove or
// rename existing ones.
type windowsPermissions struct {
	trusted []*windows.SID
	// dacl is applied, protected from inheritance, to every directory
	// MkdirAll creates: full control for SYSTEM, Administrators and the
	// current user, read and execute for Users. Files in the store inherit it.
	dacl *windows.ACL
}

// NewPermissions returns the Permissions for this platform.
func NewPermissions() (Permissions, error) {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil, fmt.Errorf("reading the current user: %w", err)
	}
	system, err := windows.CreateWellKnownSid(windows.WinLocalSystemSid)
	if err != nil {
		return nil, err
	}
	admins, err := windows.CreateWellKnownSid(windows.WinBuiltinAdministratorsSid)
	if err != nil {
		return nil, err
	}
	installer, err := windows.StringToSid(trustedInstaller)
	if err != nil {
		return nil, err
	}
	// OWNER RIGHTS applies to whoever owns the object, and owners are
	// already required to be trusted.
	ownerRights, err := windows.StringToSid("S-1-3-4")
	if err != nil {
		return nil, err
	}
	sd, err := windows.SecurityDescriptorFromString(fmt.Sprintf(
		"D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;FA;;;%s)(A;OICI;0x1200a9;;;BU)",
		user.User.Sid.String(),
	))
	if err != nil {
		return nil, err
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		return nil, err
	}
	return windowsPermissions{
		trusted: []*windows.SID{system, admins, installer, ownerRights, user.User.Sid},
		dacl:    dacl,
	}, nil
}

func (p windowsPermissions) MkdirAll(dir string) error {
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
	return windows.SetNamedSecurityInfo(
		dir,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil, nil, p.dacl, nil,
	)
}

// LockFile does nothing: files in the store inherit the DACL of the
// directory MkdirAll created for them.
func (p windowsPermissions) LockFile(path string) error {
	return nil
}

func (p windowsPermissions) Check(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return err
	}
	for cur := real; ; {
		rights := windows.ACCESS_MASK(replaceRights)
		if cur == real {
			rights = modifyRights
		}
		if err := p.checkOne(cur, rights); err != nil {
			return err
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return nil
		}
		cur = parent
	}
}

// checkOne returns an error if path's owner isn't trusted or its DACL grants
// any of rights to an untrusted SID.
func (p windowsPermissions) checkOne(path string, rights windows.ACCESS_MASK) error {
	sd, err := windows.GetNamedSecurityInfo(
		path,
		windows.SE_FILE_OBJECT,
		windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION,
	)
	if err != nil {
		return fmt.Errorf("reading permissions of %s: %w", path, err)
	}
	owner, _, err := sd.Owner()
	if err != nil {
		return fmt.Errorf("reading owner of %s: %w", path, err)
	}
	if !p.isTrusted(owner) {
		return fmt.Errorf("%s is owned by %s; want SYSTEM, Administrators or the current user", path, sidName(owner))
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		return fmt.Errorf("reading permissions of %s: %w", path, err)
	}
	if dacl == nil {
		return fmt.Errorf("%s has no DACL, so everyone can change it", path)
	}
	for i := uint32(0); i < uint32(dacl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(dacl, i, &ace); err != nil {
			return fmt.Errorf("reading permissions of %s: %w", path, err)
		}
		if ace.Header.AceFlags&windows.INHERIT_ONLY_ACE != 0 {
			continue // applies only to children, which are checked themselves
		}
		switch ace.Header.AceType {
		case windows.ACCESS_ALLOWED_ACE_TYPE:
		case accessAllowedObjectACEType, accessAllowedCallbackACEType, accessAllowedCallbackObjectACEType:
			return fmt.Errorf("%s has an allow entry of type %d that starcm can't check", path, ace.Header.AceType)
		default:
			continue // deny and audit entries only take access away
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		if ace.Mask&rights != 0 && !p.isTrusted(sid) {
			return fmt.Errorf("%s lets %s change it (access mask %#x)", path, sidName(sid), uint32(ace.Mask))
		}
	}
	return nil
}

func (p windowsPermissions) isTrusted(sid *windows.SID) bool {
	for _, t := range p.trusted {
		if sid.Equals(t) {
			return true
		}
	}
	return false
}

func sidName(sid *windows.SID) string {
	account, domain, _, err := sid.LookupAccount("")
	if err != nil {
		return sid.String()
	}
	if domain != "" {
		return domain + `\` + account
	}
	return account
}
