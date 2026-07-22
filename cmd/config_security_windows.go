//go:build windows

package cmd

import "golang.org/x/sys/windows"

// hardenConfigFile replaces inherited permissions with a protected DACL that
// grants access only to the invoking user, LocalSystem, and administrators.
func hardenConfigFile(path string) error {
	currentUser, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return err
	}
	systemSID, err := windows.CreateWellKnownSid(windows.WinLocalSystemSid)
	if err != nil {
		return err
	}
	adminSID, err := windows.CreateWellKnownSid(windows.WinBuiltinAdministratorsSid)
	if err != nil {
		return err
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

	acl, err := windows.ACLFromEntries(entries, nil)
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

func replaceConfigFile(oldPath, newPath string) error {
	return windows.Rename(oldPath, newPath)
}
