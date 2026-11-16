//go:build windows

package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

// privateConfigACL grants access only to the invoking user, LocalSystem, and
// administrators.
func privateConfigACL() (*windows.ACL, error) {
	currentUser, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil, err
	}
	systemSID, err := windows.CreateWellKnownSid(windows.WinLocalSystemSid)
	if err != nil {
		return nil, err
	}
	adminSID, err := windows.CreateWellKnownSid(windows.WinBuiltinAdministratorsSid)
	if err != nil {
		return nil, err
	}

	entries := make([]windows.EXPLICIT_ACCESS, 0, 3)
	for _, sid := range []*windows.SID{currentUser.User.Sid, systemSID, adminSID} {
		entries = append(entries, windows.EXPLICIT_ACCESS{
			AccessPermissions: windows.ACCESS_MASK(windows.GENERIC_ALL),
			AccessMode:        windows.GRANT_ACCESS,
			Inheritance:       windows.NO_INHERITANCE,
			Trustee: windows.TRUSTEE{
				TrusteeForm:  windows.TRUSTEE_IS_SID,
				TrusteeType:  windows.TRUSTEE_IS_USER,
				TrusteeValue: windows.TrusteeValueFromSID(sid),
			},
		})
	}
	entries[2].Trustee.TrusteeType = windows.TRUSTEE_IS_GROUP
	return windows.ACLFromEntries(entries, nil)
}

// hardenConfigFile replaces inherited permissions with the protected private
// DACL.
func hardenConfigFile(path string) error {
	acl, err := privateConfigACL()
	if err != nil {
		return err
	}
	return windows.SetNamedSecurityInfo(
		path,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil,
		nil,
		acl,
		nil,
	)
}

// createConfigTemp makes the config's temporary copy with the protected
// private DACL set at create time, so it never has the folder's inherited
// access. No sharing is allowed, so no other handle can read it while the
// key is written.
func createConfigTemp(dir string) (*os.File, error) {
	acl, err := privateConfigACL()
	if err != nil {
		return nil, err
	}
	sd, err := windows.NewSecurityDescriptor()
	if err != nil {
		return nil, err
	}
	if err := sd.SetDACL(acl, true, false); err != nil {
		return nil, err
	}
	if err := sd.SetControl(windows.SE_DACL_PROTECTED, windows.SE_DACL_PROTECTED); err != nil {
		return nil, err
	}
	attributes := &windows.SecurityAttributes{SecurityDescriptor: sd}
	attributes.Length = uint32(unsafe.Sizeof(*attributes))
	for range 100 {
		var suffix [8]byte
		if _, err := rand.Read(suffix[:]); err != nil {
			return nil, err
		}
		path := filepath.Join(dir, ".config-"+hex.EncodeToString(suffix[:])+".tmp")
		name, err := windows.UTF16PtrFromString(path)
		if err != nil {
			return nil, err
		}
		handle, err := windows.CreateFile(name, windows.GENERIC_READ|windows.GENERIC_WRITE, 0,
			attributes, windows.CREATE_NEW, windows.FILE_ATTRIBUTE_NORMAL, 0)
		if errors.Is(err, windows.ERROR_FILE_EXISTS) {
			continue
		}
		if err != nil {
			return nil, &os.PathError{Op: "create", Path: path, Err: err}
		}
		return os.NewFile(uintptr(handle), path), nil
	}
	return nil, errors.New("could not create a temporary config file")
}

func replaceConfigFile(oldPath, newPath string) error {
	return windows.Rename(oldPath, newPath)
}
