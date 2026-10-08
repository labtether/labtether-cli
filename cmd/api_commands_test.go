package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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

func TestPsKillCmd_AssetPathSegmentIsEscaped(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{},
		})
	}))
	defer server.Close()

	_, _, err := runConfiguredCmd(t, server.URL, "ps", "kill", "asset/one?x=1", "123")
	if err != nil {
		t.Fatalf("ps kill command failed: %v", err)
	}
	if gotPath != "/api/v2/assets/asset%2Fone%3Fx=1/processes/kill" {
		t.Fatalf("unexpected escaped path %q", gotPath)
	}
}

func TestHumanOutputCommandsFailOnMalformedResponseData(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		data any
	}{
		{
			name: "docker hosts expects list",
			args: []string{"docker", "hosts"},
			data: map[string]any{"wrong": true},
		},
		{
			name: "files cat expects object",
			args: []string{"files", "cat", "asset-1", "/tmp/test.txt"},
			data: []any{"wrong"},
		},
		{
			name: "ps list expects list",
			args: []string{"ps", "list", "asset-1"},
			data: map[string]any{"wrong": true},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"data": tc.data,
				})
			}))
			defer server.Close()

			_, _, err := runConfiguredCmd(t, server.URL, tc.args...)
			if err == nil {
				t.Fatal("expected malformed response data to fail")
			}
			if !strings.Contains(err.Error(), "decode response data") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
