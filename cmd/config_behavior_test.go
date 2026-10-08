package cmd

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

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

func TestNewClient_RefusesAPIKeyCommandLineArgument(t *testing.T) {
	t.Setenv("LABTETHER_HOST", "https://from-env.local")
	t.Setenv("LABTETHER_API_KEY", "env-key")
	cfgHost = "https://from-flag.local"
	cfgAPIKey = "flag-key"

	_, err := newClient()
	if err == nil || !strings.Contains(err.Error(), "command-line arguments") {
		t.Fatalf("expected command-line secret to be rejected, got %v", err)
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
	cfg := mustLoadConfig(t)
	if cfg.Host != "https://myhub.local" {
		t.Errorf("Host = %q, want %q", cfg.Host, "https://myhub.local")
	}
}

func TestConfigSetHost_NormalizesWhitespaceAndTrailingSlash(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("LABTETHER_HOST", "")
	t.Setenv("LABTETHER_API_KEY", "")
	cfgHost = ""
	cfgAPIKey = ""

	_, _, err := runCmd(t, "config", "set-host", " https://myhub.local/ ")
	if err != nil {
		t.Fatalf("config set-host error: %v", err)
	}

	cfg := mustLoadConfig(t)
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

	rootCmd.SetIn(strings.NewReader("lt_supersecretkey\n"))
	_, _, err := runCmd(t, "config", "set-key")
	if err != nil {
		t.Fatalf("config set-key error: %v", err)
	}

	cfg := mustLoadConfig(t)
	if cfg.APIKey != "lt_supersecretkey" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "lt_supersecretkey")
	}
}

func TestConfigSetKey_NoArg(t *testing.T) {
	rootCmd.SetIn(strings.NewReader(""))
	_, _, err := runCmd(t, "config", "set-key")
	if err == nil {
		t.Fatal("expected error for empty standard input")
	}
}

func TestConfigSetKey_RefusesArgumentThatWouldLeakToProcessList(t *testing.T) {
	_, _, err := runCmd(t, "config", "set-key", "lt_supersecretkey")
	if err == nil {
		t.Fatal("expected command-line API key to be rejected")
	}
}

func TestAPIKeyStatusDoesNotExposeSecretMaterial(t *testing.T) {
	secret := "lt_supersecretkey"
	status := apiKeyStatus(secret != "")
	if strings.Contains(status, "super") || strings.Contains(status, "key") || strings.Contains(status, "lt_") {
		t.Fatalf("status leaked secret material: %q", status)
	}
	if status != "(set)" {
		t.Fatalf("status = %q, want (set)", status)
	}
}

func TestRedactSensitiveJSON(t *testing.T) {
	input := json.RawMessage(`{
		"host": "hub.local",
		"api_key": "lt_supersecretkey",
		"api_key_status": "(set)",
		"nested": {
			"access_token": "token-value",
			"name": "visible"
		}
	}`)

	data, err := json.Marshal(redactSensitiveJSON(input))
	if err != nil {
		t.Fatalf("marshal redacted json: %v", err)
	}
	out := string(data)
	for _, leaked := range []string{"lt_supersecretkey", "token-value"} {
		if strings.Contains(out, leaked) {
			t.Fatalf("redacted JSON leaked %q in %s", leaked, out)
		}
	}
	if !strings.Contains(out, "visible") {
		t.Fatalf("redacted JSON removed non-sensitive value: %s", out)
	}
	if !strings.Contains(out, "(set)") {
		t.Fatalf("redacted JSON removed safe secret status: %s", out)
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

	got := mustLoadConfig(t)
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

	cfg := mustLoadConfig(t)
	if cfg.Host != "" || cfg.APIKey != "" {
		t.Errorf("expected empty config when file absent, got %+v", cfg)
	}
}

func TestLoadConfig_InvalidJSONReturnsError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	if err := os.MkdirAll(configDir(), 0700); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(configPath(), []byte(`{"host":`), 0600); err != nil {
		t.Fatalf("write invalid config: %v", err)
	}

	if _, err := loadConfig(); err == nil {
		t.Fatal("expected invalid config JSON to return an error")
	}
}

func TestConfigSetKeyRefusesInvalidExistingConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("LABTETHER_HOST", "")
	t.Setenv("LABTETHER_API_KEY", "")
	if err := os.MkdirAll(configDir(), 0700); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	original := []byte(`{"host":`)
	if err := os.WriteFile(configPath(), original, 0600); err != nil {
		t.Fatalf("write invalid config: %v", err)
	}

	rootCmd.SetIn(strings.NewReader("lt_newkey\n"))
	_, _, err := runCmd(t, "config", "set-key")
	if err == nil {
		t.Fatal("expected set-key to fail for invalid existing config")
	}
	got, readErr := os.ReadFile(configPath())
	if readErr != nil {
		t.Fatalf("read config: %v", readErr)
	}
	if string(got) != string(original) {
		t.Fatalf("invalid config was overwritten: %q", string(got))
	}
}

func TestNewClientAllowsEnvToBypassInvalidConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("LABTETHER_HOST", "https://env-hub.local")
	t.Setenv("LABTETHER_API_KEY", "env-key")
	cfgHost = ""
	cfgAPIKey = ""
	if err := os.MkdirAll(configDir(), 0700); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(configPath(), []byte(`{"host":`), 0600); err != nil {
		t.Fatalf("write invalid config: %v", err)
	}

	c, err := newClient()
	if err != nil {
		t.Fatalf("newClient should use complete env config despite invalid config file: %v", err)
	}
	if !strings.Contains(c.BaseURL, "env-hub") {
		t.Fatalf("env host not used; BaseURL = %s", c.BaseURL)
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
