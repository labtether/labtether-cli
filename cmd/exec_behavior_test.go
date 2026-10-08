package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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

func TestExecCmd_NonZeroRemoteExitFailsInHumanAndJSONModes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"stdout":    "partial output",
				"exit_code": 7,
			},
		})
	}))
	defer server.Close()

	for _, args := range [][]string{
		{"exec", "asset-1", "false"},
		{"--json", "exec", "asset-1", "false"},
	} {
		_, _, err := runConfiguredCmd(t, server.URL, args...)
		if err == nil || !strings.Contains(err.Error(), "exit code 7") {
			t.Fatalf("args=%v should fail with remote exit code, got %v", args, err)
		}
	}
}

func TestExecCmd_MultiTargetFailsWhenAnyRemoteCommandFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"results": map[string]any{
					"asset-ok":  map[string]any{"exit_code": 0, "stdout": "ok"},
					"asset-bad": map[string]any{"exit_code": 1, "stdout": "failed"},
				},
			},
		})
	}))
	defer server.Close()

	for _, args := range [][]string{
		{"exec", "--targets", "asset-ok,asset-bad", "check"},
		{"--json", "exec", "--targets", "asset-ok,asset-bad", "check"},
	} {
		_, _, err := runConfiguredCmd(t, server.URL, args...)
		if err == nil || !strings.Contains(err.Error(), "1 of 2 remote commands failed") {
			t.Fatalf("args=%v should fail for partial remote failure, got %v", args, err)
		}
	}
}

func TestExecCmd_MultiTargetRejectsMissingResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"summary": map[string]int{"total": 2}},
		})
	}))
	defer server.Close()

	_, _, err := runConfiguredCmd(t, server.URL, "exec", "--targets", "asset-a,asset-b", "check")
	if err == nil || !strings.Contains(err.Error(), "did not contain any results") {
		t.Fatalf("missing result map should fail closed, got %v", err)
	}
}

func TestExecCmd_RejectsMissingExitCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"stdout": "ambiguous"},
		})
	}))
	defer server.Close()

	_, _, err := runConfiguredCmd(t, server.URL, "exec", "asset-a", "check")
	if err == nil || !strings.Contains(err.Error(), "missing exit_code") {
		t.Fatalf("missing exit_code should fail closed, got %v", err)
	}
}
