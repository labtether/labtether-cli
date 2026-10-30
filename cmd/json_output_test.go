package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMutationCommandsHonorJSONOutput(t *testing.T) {
	// These families previously discarded the response and printed plain text,
	// so a successful action broke scripts that requested machine-readable data.
	commands := [][]string{
		{"alerts", "ack", "alert-1"}, {"incidents", "resolve", "incident-1"},
		{"agents", "approve", "asset-1"}, {"services", "restart", "asset-1", "nginx"},
		{"packages", "install", "asset-1", "curl"}, {"packages", "update", "asset-1"},
		{"ha", "call", "switch.test", "switch.turn_on"}, {"reboot", "asset-1"},
		{"discovery", "run"}, {"discovery", "accept", "proposal-1"},
		{"updates", "plans", "execute", "plan-1"}, {"web-services", "sync"},
		{"ps", "kill", "asset-1", "123"}, {"docker", "restart", "container-1"},
		{"proxmox", "start", "vm-1"},
	}
	result := map[string]any{"status": "succeeded", "marker": "hub-result", "result": map[string]string{"status": "succeeded"}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"asset": map[string]string{"source": "proxmox", "type": "vm"}}})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": result})
	}))
	defer server.Close()
	for _, args := range commands {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			stdout, _, err := runConfiguredCmd(t, server.URL, append([]string{"--json"}, args...)...)
			var got map[string]any
			if err != nil || json.Unmarshal([]byte(stdout), &got) != nil || got["marker"] != "hub-result" {
				t.Fatalf("action lost its JSON result: stdout=%q err=%v", stdout, err)
			}
			expected, _ := json.Marshal(result)
			actual, _ := json.Marshal(got)
			if !bytes.Equal(expected, actual) {
				t.Fatalf("action changed Hub payload: %s", stdout)
			}
		})
	}
	stdout, _, err := runConfiguredCmd(t, server.URL, "services", "restart", "asset-1", "nginx")
	if err != nil || stdout != "Service nginx restarted on asset-1\n" {
		t.Fatalf("human result changed: stdout=%q err=%v", stdout, err)
	}
}

func TestConfigMutationJSONDoesNotRevealKey(t *testing.T) {
	isolateConfig(t)
	rootCmd.SetIn(strings.NewReader("lt_private_test_key\n"))
	t.Cleanup(func() { rootCmd.SetIn(nil) })
	stdout, _, err := runCmd(t, "--json", "config", "set-key")
	var got map[string]bool
	if err != nil || json.Unmarshal([]byte(stdout), &got) != nil || !got["saved"] {
		t.Fatalf("key save did not return safe JSON: stdout=%q err=%v", stdout, err)
	}
	if strings.Contains(stdout, "lt_private_test_key") {
		t.Fatal("key save exposed the secret")
	}
	if mustLoadConfig(t).APIKey != "lt_private_test_key" {
		t.Fatal("key save reported success without storing the key")
	}
}

func TestRawCommandsPreserveBytesWithJSONFlag(t *testing.T) {
	const contents = "raw output\n\x00last byte"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/files/read") {
			_, _ = w.Write([]byte(contents))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]string{"logs": contents}})
	}))
	defer server.Close()
	for _, args := range [][]string{
		{"--json", "files", "cat", "asset-1", "/tmp/data"},
		{"--json", "docker", "logs", "container-1"},
	} {
		stdout, _, err := runConfiguredCmd(t, server.URL, args...)
		if err != nil || stdout != contents {
			t.Fatalf("raw output changed: args=%v stdout=%q err=%v", args, stdout, err)
		}
	}
}

func TestExecuteJSONFailuresEmitOneResultAndKeepExitCode(t *testing.T) {
	for _, tc := range []struct {
		name   string
		args   []string
		status int
		body   string
		code   int
		field  string
	}{
		{"auth", []string{"alerts", "ack", "alert-1"}, 403, `{"error":"forbidden","message":"scope denied"}`, 2, "error"},
		{"not found", []string{"services", "restart", "asset-1", "nginx"}, 404, `{"error":"not_found"}`, 3, "error"},
		{"action failed", []string{"docker", "restart", "container-1"}, 200, `{"data":{"result":{"status":"failed","message":"engine denied"}}}`, 1, "error"},
		{"remote exec failed", []string{"exec", "asset-1", "false"}, 200, `{"data":{"exit_code":7,"stdout":"partial output"}}`, 7, "exit_code"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			t.Setenv("LABTETHER_HOST", server.URL)
			t.Setenv("LABTETHER_API_KEY", "test-key")
			t.Setenv("LABTETHER_TLS_CA_FILE", "")
			resetTestCommandState()
			var stdout, stderr bytes.Buffer
			rootCmd.SetOut(&stdout)
			rootCmd.SetErr(&stderr)
			code := executeArgs(append([]string{"--json"}, tc.args...))
			var got map[string]any
			if code != tc.code || json.Unmarshal(stdout.Bytes(), &got) != nil || got[tc.field] == nil {
				t.Fatalf("invalid JSON failure: code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
		})
	}
}

func TestJSONArgumentErrorsRespectGlobalModeAndArgumentBoundary(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		json bool
	}{
		{"unknown command", []string{"--json", "typo"}, true},
		{"unknown flag", []string{"assets", "list", "--typo", "--json"}, true},
		{"invalid value first", []string{"exec", "--timeout", "wrong", "--json"}, true},
		{"explicit false", []string{"--json", "--json=false", "typo"}, false},
		{"argument boundary", []string{"typo", "--", "--json"}, false},
		{"global flag value", []string{"--host", "--json", "typo"}, false},
		{"command flag value", []string{"exec", "--targets", "--json"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetTestCommandState()
			var stdout, stderr bytes.Buffer
			rootCmd.SetOut(&stdout)
			rootCmd.SetErr(&stderr)
			code := executeArgs(tc.args)
			var result map[string]string
			isJSON := json.Unmarshal(stdout.Bytes(), &result) == nil && result["error"] != ""
			if code == 0 || isJSON != tc.json || (tc.json && stderr.Len() != 0) {
				t.Fatalf("argument error lost output choice: code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
		})
	}
}
