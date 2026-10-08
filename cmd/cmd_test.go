package cmd

import (
	"bytes"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"strings"
	"testing"
)

func resetTestCommandState() {
	cfgHost = ""
	cfgAPIKey = ""
	cfgAPIKeyFile = ""
	cfgTLSCAFile = ""
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
	t.Setenv("LABTETHER_TLS_CA_FILE", "")

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
	t.Setenv("LABTETHER_TLS_CA_FILE", "")

	resetTestCommandState()

	var outBuf, errBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(&errBuf)
	rootCmd.SetArgs(args)

	err = rootCmd.Execute()
	return outBuf.String(), errBuf.String(), err
}

func mustLoadConfig(t *testing.T) config {
	t.Helper()
	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	return cfg
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

func TestVersionFlagReportsBuildVersion(t *testing.T) {
	stdout, stderr, err := runCmd(t, "--version")
	if err != nil {
		t.Fatalf("--version returned error: %v (stderr: %s)", err, stderr)
	}
	if got, want := strings.TrimSpace(stdout), "labtether-cli version dev"; got != want {
		t.Fatalf("--version output = %q, want %q", got, want)
	}
}
