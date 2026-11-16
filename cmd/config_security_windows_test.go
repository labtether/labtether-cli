//go:build windows

package cmd

import (
	"io"
	"os"
	"regexp"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

func assertConfigProtection(t *testing.T, path string) {
	t.Helper()
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil || sd == nil {
		t.Fatalf("read config ACL: descriptor=%v error=%v", sd, err)
	}
	control, _, err := sd.Control()
	if err != nil || control&windows.SE_DACL_PROTECTED == 0 {
		t.Fatalf("config ACL still inherits access: control=%v error=%v", control, err)
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatalf("read current user: %v", err)
	}
	allowed := map[string]bool{user.User.Sid.String(): true, "SY": true, "BA": true}
	entries := regexp.MustCompile(`\(([^()]*)\)`).FindAllStringSubmatch(sd.String(), -1)
	if len(entries) != len(allowed) {
		t.Fatalf("config ACL contains %d entries, want %d", len(entries), len(allowed))
	}
	for _, entry := range entries {
		fields := strings.Split(entry[1], ";")
		if len(fields) != 6 || fields[0] != "A" || fields[1] != "" ||
			(fields[2] != "FA" && fields[2] != "GA") || !allowed[fields[5]] {
			t.Fatalf("unexpected config ACL entry: %q", entry[1])
		}
		delete(allowed, fields[5])
	}
}

// Fault: the config's temporary copy was created with the folder's inherited
// access and protected only afterwards. A handle opened in that moment keeps
// the access it was opened with, so it could read the API key written next.
func TestSaveConfigTempFileIsPrivateFromCreation(t *testing.T) {
	isolateConfig(t)
	var early *os.File
	checked := false
	previous := configTempCreated
	configTempCreated = func(path string) {
		checked = true
		assertConfigProtection(t, path)
		if file, err := os.Open(path); err == nil {
			early = file
		}
	}
	t.Cleanup(func() {
		configTempCreated = previous
		if early != nil {
			_ = early.Close()
		}
	})
	if err := saveConfig(config{Host: "https://hub.example.com", APIKey: "key-in-temp-copy"}); err != nil {
		t.Fatalf("save config: %v", err)
	}
	if !checked {
		t.Fatal("the temporary copy was never checked")
	}
	if early != nil {
		data, _ := io.ReadAll(early)
		t.Fatalf("a second handle was opened on the temporary copy and read %d bytes", len(data))
	}
	assertConfigProtection(t, configPath())
}
