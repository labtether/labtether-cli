package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func resetTestCommandState() {
	cfgHost = ""
	cfgAPIKey = ""
	jsonOutput = false
	resetCobraCommandFlags(rootCmd)
}

func resetCobraCommandFlags(cmd *cobra.Command) {
	resetFlagSet(cmd.PersistentFlags())
	resetFlagSet(cmd.Flags())
	for _, child := range cmd.Commands() {
		resetCobraCommandFlags(child)
	}
}

func resetFlagSet(flags *pflag.FlagSet) {
	flags.VisitAll(func(flag *pflag.Flag) {
		_ = flag.Value.Set(flag.DefValue)
		flag.Changed = false
	})
}

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
	resetTestCommandState()

	var outBuf, errBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(&errBuf)
	rootCmd.SetArgs(args)

	err = rootCmd.Execute()
	return outBuf.String(), errBuf.String(), err
}

func runConfiguredCmd(t *testing.T, host string, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	t.Setenv("LABTETHER_HOST", host)
	t.Setenv("LABTETHER_API_KEY", "test-key")

	resetTestCommandState()

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

func TestExecCmd_InvalidTimeoutsFailBeforeClientConfig(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{
			name: "zero",
			args: []string{"exec", "--timeout", "0", "asset-1", "echo ok"},
		},
		{
			name: "negative",
			args: []string{"exec", "--timeout", "-1", "asset-1", "echo ok"},
		},
		{
			name: "too high",
			args: []string{"exec", "--timeout", "301", "asset-1", "echo ok"},
		},
		{
			name: "signed",
			args: []string{"exec", "--timeout", "+30", "asset-1", "echo ok"},
		},
		{
			name: "malformed",
			args: []string{"exec", "--timeout", "30abc", "asset-1", "echo ok"},
		},
		{
			name: "multi target too high",
			args: []string{"exec", "--targets", "asset-1,asset-2", "--timeout", "301", "uptime"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := runCmd(t, tc.args...)
			if err == nil {
				t.Fatal("expected timeout validation error")
			}
			if !strings.Contains(err.Error(), "timeout must be between 1 and 300 seconds") {
				t.Fatalf("unexpected error: %v", err)
			}
			if strings.Contains(err.Error(), "not configured") || strings.Contains(err.Error(), "request failed") {
				t.Fatalf("timeout validation should fail before client setup/network, got: %v", err)
			}
		})
	}
}

func TestDockerLogsCmd_InvalidTailFailsBeforeClientConfig(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{
			name: "zero",
			args: []string{"docker", "logs", "--tail", "0", "container-1"},
		},
		{
			name: "negative",
			args: []string{"docker", "logs", "--tail", "-1", "container-1"},
		},
		{
			name: "too high",
			args: []string{"docker", "logs", "--tail", "10001", "container-1"},
		},
		{
			name: "signed",
			args: []string{"docker", "logs", "--tail", "+100", "container-1"},
		},
		{
			name: "malformed",
			args: []string{"docker", "logs", "--tail", "100abc", "container-1"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := runCmd(t, tc.args...)
			if err == nil {
				t.Fatal("expected tail validation error")
			}
			if !strings.Contains(err.Error(), "tail must be between 1 and 10000") {
				t.Fatalf("unexpected error: %v", err)
			}
			if strings.Contains(err.Error(), "not configured") || strings.Contains(err.Error(), "request failed") {
				t.Fatalf("tail validation should fail before client setup/network, got: %v", err)
			}
		})
	}
}

// ── assets argument validation ────────────────────────────────────────────

func TestAssetsGetCmd_NoArgs(t *testing.T) {
	_, _, err := runCmd(t, "assets", "get")
	if err == nil {
		t.Fatal("expected error: 'assets get' requires exactly one arg")
	}
}

func TestCLIPathSegmentsAreEscaped(t *testing.T) {
	var gotPath, gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		gotQuery = r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]string{"logs": "ok"},
		})
	}))
	defer server.Close()

	_, _, err := runConfiguredCmd(t, server.URL, "docker", "logs", "--tail", "7", "ct/one?x=1")
	if err != nil {
		t.Fatalf("docker logs command failed: %v", err)
	}
	if gotPath != "/api/v2/docker/containers/ct%2Fone%3Fx=1/logs" {
		t.Fatalf("unexpected escaped path %q", gotPath)
	}
	if gotQuery != "tail=7" {
		t.Fatalf("unexpected query %q", gotQuery)
	}
}

func TestExecCmd_SingleTargetPathSegmentIsEscaped(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"stdout":    "ok",
				"exit_code": 0,
			},
		})
	}))
	defer server.Close()

	_, _, err := runConfiguredCmd(t, server.URL, "exec", "asset/one?x=1", "echo", "ok")
	if err != nil {
		t.Fatalf("exec command failed: %v", err)
	}
	if gotPath != "/api/v2/assets/asset%2Fone%3Fx=1/exec" {
		t.Fatalf("unexpected escaped path %q", gotPath)
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
