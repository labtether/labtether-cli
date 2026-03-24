package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// runCmd executes the root cobra command with the given args and returns
// stdout, stderr, and the error. It isolates the test from the real
// config file and from leaked env vars.
func runCmd(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	// Wipe config env so tests don't accidentally inherit a real host/key
	t.Setenv("LABTETHER_HOST", "")
	t.Setenv("LABTETHER_API_KEY", "")

	// Reset flag state — cobra caches persistent-flag values between calls
	// within the same process; reset to defaults before each test.
	cfgHost = ""
	cfgAPIKey = ""
	jsonOutput = false

	var outBuf, errBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(&errBuf)
	rootCmd.SetArgs(args)

	err = rootCmd.Execute()
	return outBuf.String(), errBuf.String(), err
}

// ── newClient / config resolution ─────────────────────────────────────────

func TestNewClient_NotConfigured_NoHost(t *testing.T) {
	t.Setenv("LABTETHER_HOST", "")
	t.Setenv("LABTETHER_API_KEY", "")
	cfgHost = ""
	cfgAPIKey = ""

	_, err := newClient()
	if err == nil {
		t.Fatal("expected error when host is not configured")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestNewClient_NotConfigured_NoKey(t *testing.T) {
	t.Setenv("LABTETHER_HOST", "https://hub.local")
	t.Setenv("LABTETHER_API_KEY", "")
	cfgHost = ""
	cfgAPIKey = ""

	_, err := newClient()
	if err == nil {
		t.Fatal("expected error when API key is not configured")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestNewClient_FlagOverridesEnv(t *testing.T) {
	t.Setenv("LABTETHER_HOST", "https://from-env.local")
	t.Setenv("LABTETHER_API_KEY", "env-key")
	cfgHost = "https://from-flag.local"
	cfgAPIKey = "flag-key"

	c, err := newClient()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(c.BaseURL, "from-flag") {
		t.Errorf("flag host not used; BaseURL = %s", c.BaseURL)
	}
}

func TestNewClient_EnvOverridesConfig(t *testing.T) {
	// Put a dummy config so the file loader finds something
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	cfgHost = ""
	cfgAPIKey = ""
	t.Setenv("LABTETHER_HOST", "https://from-env.local")
	t.Setenv("LABTETHER_API_KEY", "env-key-123")

	c, err := newClient()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(c.BaseURL, "from-env") {
		t.Errorf("env host not used; BaseURL = %s", c.BaseURL)
	}
}

// ── Execute exit codes ────────────────────────────────────────────────────

func TestExecute_NoArgs_ReturnsZero(t *testing.T) {
	t.Setenv("LABTETHER_HOST", "")
	t.Setenv("LABTETHER_API_KEY", "")
	cfgHost = ""
	cfgAPIKey = ""
	rootCmd.SetArgs([]string{})

	code := Execute()
	if code != 0 {
		t.Errorf("Execute() with no args = %d, want 0 (shows help)", code)
	}
}

// ── exec argument validation ──────────────────────────────────────────────

func TestExecCmd_TooFewArgs_SingleTarget(t *testing.T) {
	t.Setenv("LABTETHER_HOST", "")
	t.Setenv("LABTETHER_API_KEY", "")
	cfgHost = ""
	cfgAPIKey = ""

	_, _, err := runCmd(t, "exec", "only-asset")
	if err == nil {
		t.Fatal("expected error: exec with one arg (no command) should fail before hitting API")
	}
	// Ensure it's a usage error, not a network error
	if strings.Contains(err.Error(), "connection refused") {
		t.Error("test unexpectedly reached the network")
	}
}

func TestExecCmd_MultiTarget_NoCommand(t *testing.T) {
	t.Setenv("LABTETHER_HOST", "")
	t.Setenv("LABTETHER_API_KEY", "")
	cfgHost = ""
	cfgAPIKey = ""

	_, _, err := runCmd(t, "exec", "--targets", "a,b")
	if err == nil {
		t.Fatal("expected error: --targets without a command should fail")
	}
}

// ── assets argument validation ────────────────────────────────────────────

func TestAssetsGetCmd_NoArgs(t *testing.T) {
	_, _, err := runCmd(t, "assets", "get")
	if err == nil {
		t.Fatal("expected error: 'assets get' requires exactly one arg")
	}
}

// ── config commands (no network) ─────────────────────────────────────────

func TestConfigShow_NoConfig(t *testing.T) {
	// Use a temp home so no real config file exists
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("LABTETHER_HOST", "")
	t.Setenv("LABTETHER_API_KEY", "")
	cfgHost = ""
	cfgAPIKey = ""

	_, _, err := runCmd(t, "config", "show")
	if err != nil {
		t.Fatalf("config show should not error: %v", err)
	}
}

func TestConfigSetHost_SavesAndLoads(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("LABTETHER_HOST", "")
	t.Setenv("LABTETHER_API_KEY", "")
	cfgHost = ""
	cfgAPIKey = ""

	_, _, err := runCmd(t, "config", "set-host", "https://myhub.local")
	if err != nil {
		t.Fatalf("config set-host error: %v", err)
	}

	// Load the saved config directly
	cfg := loadConfig()
	if cfg.Host != "https://myhub.local" {
		t.Errorf("Host = %q, want %q", cfg.Host, "https://myhub.local")
	}
}

func TestConfigSetKey_SavesAndLoads(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("LABTETHER_HOST", "")
	t.Setenv("LABTETHER_API_KEY", "")
	cfgHost = ""
	cfgAPIKey = ""

	_, _, err := runCmd(t, "config", "set-key", "lt_supersecretkey")
	if err != nil {
		t.Fatalf("config set-key error: %v", err)
	}

	cfg := loadConfig()
	if cfg.APIKey != "lt_supersecretkey" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "lt_supersecretkey")
	}
}

func TestConfigSetKey_NoArg(t *testing.T) {
	_, _, err := runCmd(t, "config", "set-key")
	if err == nil {
		t.Fatal("expected error: set-key requires exactly one arg")
	}
}

// ── saveConfig / loadConfig round-trip ───────────────────────────────────

func TestSaveLoadConfig_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	cfgHost = ""
	cfgAPIKey = ""

	want := config{Host: "https://round.trip", APIKey: "key-rt"}
	if err := saveConfig(want); err != nil {
		t.Fatalf("saveConfig: %v", err)
	}

	got := loadConfig()
	if got.Host != want.Host {
		t.Errorf("Host = %q, want %q", got.Host, want.Host)
	}
	if got.APIKey != want.APIKey {
		t.Errorf("APIKey = %q, want %q", got.APIKey, want.APIKey)
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	cfgHost = ""
	cfgAPIKey = ""

	cfg := loadConfig()
	if cfg.Host != "" || cfg.APIKey != "" {
		t.Errorf("expected empty config when file absent, got %+v", cfg)
	}
}

func TestSaveConfig_FilePermissions(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	if err := saveConfig(config{Host: "h", APIKey: "k"}); err != nil {
		t.Fatalf("saveConfig: %v", err)
	}

	info, err := os.Stat(configPath())
	if err != nil {
		t.Fatalf("stat config: %v", err)
	}
	// Config file should be owner-only (0600)
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("config file permissions = %04o, want 0600", perm)
	}
}
