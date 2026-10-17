//go:build windows

package localstore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows"
)

const (
	fileAllAccess       = 0x1f01ff // FA
	fileGenericReadExec = 0x1200a9 // read and execute
	objectContainerInh  = windows.OBJECT_INHERIT_ACE | windows.CONTAINER_INHERIT_ACE
)

func currentUserSID(t *testing.T) string {
	t.Helper()
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	require.NoError(t, err)
	return user.User.Sid.String()
}

// setDACL replaces path's DACL with the one in sddl, protected from
// inheritance. A sddl of "D:NO_ACCESS_CONTROL" sets a null DACL.
func setDACL(t *testing.T, path, sddl string) {
	t.Helper()
	sd, err := windows.SecurityDescriptorFromString(sddl)
	require.NoError(t, err)
	dacl, _, err := sd.DACL()
	require.NoError(t, err)
	require.NoError(t, windows.SetNamedSecurityInfo(
		path,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil, nil, dacl, nil,
	))
}

// trustedDACL is SDDL granting full control to SYSTEM, Administrators and the
// current user, followed by extra.
func trustedDACL(t *testing.T, flags, extra string) string {
	return fmt.Sprintf("D:P(A;%[1]s;FA;;;SY)(A;%[1]s;FA;;;BA)(A;%[1]s;FA;;;%[2]s)%[3]s",
		flags, currentUserSID(t), extra)
}

type ace struct {
	sid   string
	mask  uint32
	flags uint8
	typ   uint8
}

func readACEs(t *testing.T, path string) (windows.SECURITY_DESCRIPTOR_CONTROL, []ace) {
	t.Helper()
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	require.NoError(t, err)
	control, _, err := sd.Control()
	require.NoError(t, err)
	dacl, _, err := sd.DACL()
	require.NoError(t, err)
	require.NotNil(t, dacl)
	var aces []ace
	for i := uint32(0); i < uint32(dacl.AceCount); i++ {
		var a *windows.ACCESS_ALLOWED_ACE
		require.NoError(t, windows.GetAce(dacl, i, &a))
		sid := (*windows.SID)(unsafe.Pointer(&a.SidStart))
		aces = append(aces, ace{sid.String(), uint32(a.Mask), a.Header.AceFlags, a.Header.AceType})
	}
	return control, aces
}

// realTempDir is t.TempDir() with symlinks and short (8.3) names resolved,
// matching the paths Check reports.
func realTempDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	return dir
}

func newPerms(t *testing.T) Permissions {
	t.Helper()
	perms, err := NewPermissions()
	require.NoError(t, err)
	return perms
}

func TestMkdirAllAppliesProtectedDACL(t *testing.T) {
	perms := newPerms(t)
	top := filepath.Join(realTempDir(t), "a")
	dir := filepath.Join(top, "b")
	require.NoError(t, perms.MkdirAll(dir))

	want := []ace{
		{"S-1-5-18", fileAllAccess, objectContainerInh, windows.ACCESS_ALLOWED_ACE_TYPE},
		{"S-1-5-32-544", fileAllAccess, objectContainerInh, windows.ACCESS_ALLOWED_ACE_TYPE},
		{currentUserSID(t), fileAllAccess, objectContainerInh, windows.ACCESS_ALLOWED_ACE_TYPE},
		{"S-1-5-32-545", fileGenericReadExec, objectContainerInh, windows.ACCESS_ALLOWED_ACE_TYPE},
	}
	// Both the directory and the parent MkdirAll created for it.
	for _, d := range []string{top, dir} {
		control, aces := readACEs(t, d)
		require.NotZero(t, control&windows.SE_DACL_PROTECTED, "%s: DACL is not protected", d)
		require.Equal(t, want, aces, d)
	}
	require.NoError(t, perms.Check(dir))
}

func TestCheckAcceptsFreshStore(t *testing.T) {
	perms := newPerms(t)
	s, err := Open(filepath.Join(realTempDir(t), "store"), perms)
	require.NoError(t, err)
	path, err := s.Put(contentHash, "payload.txt", strings.NewReader(content))
	require.NoError(t, err)
	got, ok, err := s.Get(contentHash, "payload.txt")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, path, got)

	for _, p := range []string{s.Dir(), filepath.Dir(path), path} {
		require.NoError(t, perms.Check(p), p)
	}
}

// storedFile returns a store and the path of a file Put into it.
func storedFile(t *testing.T) (*Store, string) {
	t.Helper()
	s, err := Open(filepath.Join(realTempDir(t), "store"), newPerms(t))
	require.NoError(t, err)
	path, err := s.Put(contentHash, "payload.txt", strings.NewReader(content))
	require.NoError(t, err)
	return s, path
}

func TestCheckRefusesFileWritableByOthers(t *testing.T) {
	for _, tc := range []struct{ name, sid string }{
		{"Everyone", "WD"},
		{"Authenticated Users", "AU"},
		{"Users", "BU"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, path := storedFile(t)
			// Same DACL with only read access for tc.sid is fine...
			setDACL(t, path, trustedDACL(t, "", fmt.Sprintf("(A;;0x1200a9;;;%s)", tc.sid)))
			_, ok, err := s.Get(contentHash, "payload.txt")
			require.NoError(t, err)
			require.True(t, ok)

			// ...but FILE_WRITE_DATA is not.
			setDACL(t, path, trustedDACL(t, "", fmt.Sprintf("(A;;0x2;;;%s)", tc.sid)))
			err = s.perms.Check(path)
			require.ErrorContains(t, err, "change it")
			require.ErrorContains(t, err, "access mask 0x2")
			_, _, err = s.Get(contentHash, "payload.txt")
			require.ErrorContains(t, err, "is not safe to use")
		})
	}
}

func TestCheckRefusesReplaceableParent(t *testing.T) {
	for _, tc := range []struct {
		name string
		mask uint32
	}{
		{"DELETE", accessDelete},
		{"WRITE_DAC", writeDAC},
		{"WRITE_OWNER", writeOwner},
		{"FILE_DELETE_CHILD", fileDeleteChild},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parent := filepath.Join(realTempDir(t), "parent")
			require.NoError(t, os.Mkdir(parent, 0o755))
			setDACL(t, parent, trustedDACL(t, "OICI", fmt.Sprintf("(A;;%#x;;;AU)", tc.mask)))

			_, err := Open(filepath.Join(parent, "store"), newPerms(t))
			require.ErrorContains(t, err, "is not safe to use")
			require.ErrorContains(t, err, parent+" lets ")
		})
	}
}

func TestCheckAllowsParentOthersCanAddTo(t *testing.T) {
	// Directories above the store only need to be safe from replacement:
	// others may add entries (FILE_ADD_FILE, FILE_ADD_SUBDIRECTORY).
	parent := filepath.Join(realTempDir(t), "parent")
	require.NoError(t, os.Mkdir(parent, 0o755))
	setDACL(t, parent, trustedDACL(t, "OICI", "(A;;0x1200af;;;AU)"))

	_, err := Open(filepath.Join(parent, "store"), newPerms(t))
	require.NoError(t, err)
}

func TestCheckIgnoresInheritOnlyACE(t *testing.T) {
	// An inherit-only entry applies only to children, which get it as a
	// normal entry and are checked themselves.
	parent := filepath.Join(realTempDir(t), "parent")
	require.NoError(t, os.Mkdir(parent, 0o755))
	setDACL(t, parent, trustedDACL(t, "OICI", "(A;OICIIO;FA;;;AU)"))
	perms := newPerms(t)
	require.NoError(t, perms.Check(parent))

	child := filepath.Join(parent, "child")
	require.NoError(t, os.WriteFile(child, nil, 0o644))
	require.ErrorContains(t, perms.Check(child), child+" lets ")
}

func TestCheckRefusesNullDACL(t *testing.T) {
	s, path := storedFile(t)
	perms := s.perms

	setDACL(t, path, "D:NO_ACCESS_CONTROL")
	require.ErrorContains(t, perms.Check(path), path+" has no DACL")

	setDACL(t, s.Dir(), "D:NO_ACCESS_CONTROL")
	require.ErrorContains(t, perms.Check(s.Dir()), s.Dir()+" has no DACL")
	_, err := Open(s.Dir(), perms)
	require.ErrorContains(t, err, "has no DACL")
}

func TestCheckRefusesCallbackACE(t *testing.T) {
	// A conditional (callback) allow entry: Check can't evaluate the
	// condition, so it refuses the object whoever the entry names.
	_, path := storedFile(t)
	sd, err := windows.SecurityDescriptorFromString(
		trustedDACL(t, "", "(XA;;FA;;;WD;(Member_of {SID(BA)}))"))
	if err != nil {
		t.Skipf("building a conditional ACE: %v", err)
	}
	dacl, _, err := sd.DACL()
	require.NoError(t, err)
	if err := windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil, nil, dacl, nil); err != nil {
		t.Skipf("this file system doesn't accept conditional ACEs: %v", err)
	}
	require.ErrorContains(t, newPerms(t).Check(path), "that starcm can't check")
}

// withRestorePrivilege runs f with SeRestorePrivilege enabled on an
// impersonation token for this thread only, which lets f set any owner. It
// skips the test if the process doesn't hold the privilege.
func withRestorePrivilege(t *testing.T, f func() error) {
	t.Helper()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	require.NoError(t, windows.ImpersonateSelf(windows.SecurityImpersonation))
	defer func() { require.NoError(t, windows.RevertToSelf()) }()

	var token windows.Token
	require.NoError(t, windows.OpenThreadToken(windows.CurrentThread(),
		windows.TOKEN_ADJUST_PRIVILEGES|windows.TOKEN_QUERY, false, &token))
	defer token.Close()

	var luid windows.LUID
	require.NoError(t, windows.LookupPrivilegeValue(nil, windows.StringToUTF16Ptr("SeRestorePrivilege"), &luid))
	privs := windows.Tokenprivileges{
		PrivilegeCount: 1,
		Privileges:     [1]windows.LUIDAndAttributes{{Luid: luid, Attributes: windows.SE_PRIVILEGE_ENABLED}},
	}
	// AdjustTokenPrivileges succeeds even when the privilege isn't held, so
	// a missing privilege shows up as f failing below.
	require.NoError(t, windows.AdjustTokenPrivileges(token, false, &privs, 0, nil, nil))

	err := f()
	if errors.Is(err, windows.ERROR_INVALID_OWNER) || errors.Is(err, windows.ERROR_ACCESS_DENIED) ||
		errors.Is(err, windows.ERROR_PRIVILEGE_NOT_HELD) {
		t.Skipf("setting a foreign owner needs SeRestorePrivilege, which this process doesn't hold (run elevated): %v", err)
	}
	require.NoError(t, err)
}

func TestCheckRefusesUntrustedOwner(t *testing.T) {
	_, path := storedFile(t)
	users, err := windows.CreateWellKnownSid(windows.WinBuiltinUsersSid)
	require.NoError(t, err)

	withRestorePrivilege(t, func() error {
		return windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT,
			windows.OWNER_SECURITY_INFORMATION, users, nil, nil, nil)
	})
	require.ErrorContains(t, newPerms(t).Check(path), path+" is owned by ")
}
