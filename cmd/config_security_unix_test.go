//go:build !windows

package cmd

import (
	"os"
	"testing"
)

func assertConfigProtection(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat config: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("config file permissions = %04o, want 0600", perm)
	}
}
