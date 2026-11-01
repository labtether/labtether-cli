package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Lab QA saw usage text after every failure, which buried Hub errors. Usage
// belongs only to mistakes in the command line itself.
func TestUsageTextOnlyForCommandLineMistakes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"agent disconnected"}`))
	}))
	defer server.Close()
	for _, tc := range []struct {
		name  string
		args  []string
		usage bool
	}{
		{"hub failure", []string{"ps", "list", "node-1"}, false},
		{"missing argument", []string{"ps", "list"}, true},
		{"unknown flag", []string{"ps", "list", "node-1", "--typo"}, true},
		{"invalid pid", []string{"ps", "kill", "node-1", "abc"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("LABTETHER_HOST", server.URL)
			t.Setenv("LABTETHER_API_KEY", "test-key")
			t.Setenv("LABTETHER_TLS_CA_FILE", "")
			resetTestCommandState()
			var stdout, stderr bytes.Buffer
			rootCmd.SetOut(&stdout)
			rootCmd.SetErr(&stderr)
			if code := executeArgs(tc.args); code == 0 {
				t.Fatal("command succeeded, want failure")
			}
			output := stdout.String() + stderr.String()
			if !strings.Contains(output, "Error:") || strings.Contains(output, "Usage:") != tc.usage {
				t.Fatalf("usage shown = %v, want %v; output=%q", !tc.usage, tc.usage, output)
			}
		})
	}
}
