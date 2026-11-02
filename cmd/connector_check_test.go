package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The Hub rejects a connector test without connection settings, so
// `connectors test` failed for every saved connection before it sent them.
func TestConnectorTestSendsSavedConnectionSettings(t *testing.T) {
	collectors := []map[string]any{
		{"id": "col-ha", "collector_type": "homeassistant", "config": map[string]any{
			"base_url": "https://ha.example.com", "credential_id": "cred-ha", "token": "[redacted]",
		}},
		{"id": "col-pve-1", "collector_type": "proxmox", "config": map[string]any{"base_url": "https://pve1.example.com"}},
		{"id": "col-pve-2", "collector_type": "proxmox", "config": map[string]any{"base_url": "https://pve2.example.com"}},
	}
	var testPath string
	var testBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v2/collectors" {
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"collectors": collectors}})
			return
		}
		testPath, testBody = r.URL.Path, nil
		if err := json.NewDecoder(r.Body).Decode(&testBody); err != nil {
			t.Errorf("test request has no JSON body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"success": true}})
	}))
	defer server.Close()

	if _, _, err := runConfiguredCmd(t, server.URL, "connectors", "test", "home-assistant"); err != nil {
		t.Fatalf("test home-assistant: %v", err)
	}
	if testPath != "/api/v2/connectors/home-assistant/test" ||
		testBody["base_url"] != "https://ha.example.com" || testBody["credential_id"] != "cred-ha" {
		t.Fatalf("path=%q body=%v", testPath, testBody)
	}
	if _, found := testBody["token"]; found {
		t.Fatalf("test body carries a secret field: %v", testBody)
	}

	_, _, err := runConfiguredCmd(t, server.URL, "connectors", "test", "proxmox")
	if err == nil || !strings.Contains(err.Error(), "col-pve-1, col-pve-2") {
		t.Fatalf("two saved connections error=%v", err)
	}
	if _, _, err := runConfiguredCmd(t, server.URL, "connectors", "test", "col-pve-2"); err != nil {
		t.Fatalf("test by collector ID: %v", err)
	}
	if testPath != "/api/v2/connectors/proxmox/test" || testBody["base_url"] != "https://pve2.example.com" {
		t.Fatalf("path=%q body=%v", testPath, testBody)
	}
}
