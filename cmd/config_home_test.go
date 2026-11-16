package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withoutHomeFolder makes the home lookup fail and points the system
// temporary folder at a folder the test controls, where it plants a config
// file as another user on a shared machine could.
func withoutHomeFolder(t *testing.T) (tempDir string) {
	t.Helper()
	tempDir = t.TempDir()
	for _, name := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(name, tempDir)
	}
	previous := configHomeDir
	configHomeDir = func() (string, error) { return "", errors.New("no home folder") }
	t.Cleanup(func() { configHomeDir = previous })
	planted := filepath.Join(tempDir, "labtether")
	if err := os.MkdirAll(planted, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"host":"https://hub.attacker.example","tls_ca_file":""}`
	if err := os.WriteFile(filepath.Join(planted, "config.json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	cfgHost, cfgAPIKey, cfgAPIKeyFile, cfgTLSCAFile = "", "", "", ""
	return tempDir
}

// Fault: with no home folder the CLI read its config from the shared
// temporary folder, so a config planted there by another user sent the API
// key from LABTETHER_API_KEY to that user's Hub address.
func TestNoHomeFolderIgnoresConfigInSharedTempFolder(t *testing.T) {
	withoutHomeFolder(t)
	t.Setenv("LABTETHER_HOST", "")
	t.Setenv("LABTETHER_API_KEY", "key-from-env")
	t.Setenv("LABTETHER_TLS_CA_FILE", "")
	c, err := newClient()
	if err == nil {
		t.Fatalf("the key would go to %s", c.BaseURL)
	}
	if strings.Contains(err.Error(), "attacker") {
		t.Fatalf("planted config was read: %v", err)
	}
}

// Fault: the same fallback let "config set-key" write the key into the
// shared temporary folder, where another user could have made the folder.
func TestNoHomeFolderRefusesToSaveSettings(t *testing.T) {
	tempDir := withoutHomeFolder(t)
	if err := os.RemoveAll(filepath.Join(tempDir, "labtether")); err != nil {
		t.Fatal(err)
	}
	err := saveConfig(config{Host: "https://hub.example.com", APIKey: "key-to-save"})
	if err == nil || !strings.Contains(err.Error(), "LABTETHER_HOST") {
		t.Fatalf("save without a home folder: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(tempDir, "labtether")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("settings folder was made in the temporary folder: %v", statErr)
	}
}

// A run with LABTETHER_HOST and LABTETHER_API_KEY set and no home folder must
// keep working: it needs no config file.
func TestNoHomeFolderWorksWithEnvironmentSettings(t *testing.T) {
	withoutHomeFolder(t)
	t.Setenv("LABTETHER_HOST", "https://hub.example.com")
	t.Setenv("LABTETHER_API_KEY", "key-from-env")
	t.Setenv("LABTETHER_TLS_CA_FILE", "")
	c, err := newClient()
	if err != nil || c.BaseURL != "https://hub.example.com" || c.APIKey != "key-from-env" {
		t.Fatalf("environment-only run: client=%+v err=%v", c, err)
	}
}
