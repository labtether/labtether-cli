package cmd

import (
	"fmt"
	"os"
	"testing"
)

// No command test may read or change the caller's real CLI credentials.
// Replacing the directory lookup also works on Windows without changing HOME
// or USERPROFILE for unrelated subprocesses.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "labtether-cli-tests-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	configHomeDir = func() (string, error) { return dir, nil }
	exitCode := m.Run()
	if err := os.RemoveAll(dir); err != nil {
		fmt.Fprintln(os.Stderr, "remove test config:", err)
		exitCode = 1
	}
	os.Exit(exitCode)
}

func isolateConfig(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	previous := configHomeDir
	configHomeDir = func() (string, error) { return dir, nil }
	t.Cleanup(func() { configHomeDir = previous })
}
