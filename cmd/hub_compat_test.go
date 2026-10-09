package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCommandsUseCurrentHubRoutes(t *testing.T) {
	checks := []struct {
		name   string
		args   []string
		method string
		path   string
		data   any
	}{
		{"checks list", []string{"checks", "list"}, "GET", "/api/v2/synthetic-checks", map[string]any{}},
		{"checks get", []string{"checks", "get", "check-1"}, "GET", "/api/v2/synthetic-checks/check-1", map[string]any{}},
		{"credentials list", []string{"credentials", "list"}, "GET", "/api/v2/credentials/profiles", map[string]any{}},
		{"credentials get", []string{"credentials", "get", "profile-1"}, "GET", "/api/v2/credentials/profiles/profile-1", map[string]any{}},
		{"web services list", []string{"web-services", "list"}, "GET", "/api/v2/web-services", map[string]any{}},
		{"web services sync", []string{"web-services", "sync"}, "POST", "/api/v2/web-services/sync", map[string]any{}},
		{"tls status", []string{"tls", "status"}, "GET", "/api/v2/hub/tls", map[string]any{}},
		{"failover list", []string{"failover", "list"}, "GET", "/api/v2/failover-pairs", map[string]any{}},
		{"failover get", []string{"failover", "get", "pair-1"}, "GET", "/api/v2/failover-pairs/pair-1", map[string]any{}},
		{"failover readiness", []string{"failover", "check-readiness", "pair-1"}, "POST", "/api/v2/failover-pairs/pair-1/check-readiness", map[string]any{}},
		{"dependencies", []string{"topology", "dependencies", "asset-1"}, "GET", "/api/v2/dependencies?asset_id=asset-1", map[string]any{}},
		{"downstream", []string{"topology", "blast-radius", "asset-1"}, "GET", "/api/v2/edges/tree?root=asset-1", map[string]any{}},
		{"upstream", []string{"topology", "upstream", "asset-1"}, "GET", "/api/v2/edges/ancestors?id=asset-1", map[string]any{}},
		{"edges", []string{"topology", "edges", "asset-1"}, "GET", "/api/v2/edges?asset_id=asset-1", map[string]any{}},
		{"discovery dismiss", []string{"discovery", "dismiss", "proposal-1"}, "POST", "/api/v2/discovery/proposals/proposal-1/dismiss", map[string]any{}},
		{"truenas overview", []string{"truenas", "get", "nas-1"}, "GET", "/api/v2/truenas/assets/nas-1/overview", map[string]any{}},
		{"pbs details", []string{"pbs", "get", "pbs-1"}, "GET", "/api/v2/pbs/assets/pbs-1/details", map[string]any{}},
		{"portainer overview", []string{"portainer", "get", "env-1"}, "GET", "/api/v2/portainer/assets/env-1/overview", map[string]any{}},
		{"proxmox resources", []string{"proxmox", "resources"}, "GET", "/api/v2/proxmox/cluster/resources", map[string]any{}},
		{"proxmox nodes", []string{"proxmox", "nodes"}, "GET", "/api/v2/proxmox/cluster/resources", map[string]any{"resources": []any{}}},
		{"proxmox details", []string{"proxmox", "get", "vm-1"}, "GET", "/api/v2/proxmox/assets/vm-1/details", map[string]any{}},
		{"agent asset", []string{"agents", "get", "agent-1"}, "GET", "/api/v2/assets/agent-1", map[string]any{}},
		{"alert resolve", []string{"alerts", "resolve", "alert-1"}, "POST", "/api/v2/alerts/alert-1/resolve", map[string]any{}},
		{"package upgrade", []string{"packages", "update", "agent-1"}, "POST", "/api/v2/assets/agent-1/packages/upgrade", map[string]any{}},
	}
	for _, tc := range checks {
		t.Run(tc.name, func(t *testing.T) {
			var gotMethod, gotPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod, gotPath = r.Method, r.URL.RequestURI()
				_ = json.NewEncoder(w).Encode(map[string]any{"data": tc.data})
			}))
			defer server.Close()
			_, _, err := runConfiguredCmd(t, server.URL, tc.args...)
			if err != nil {
				t.Fatalf("command failed: %v", err)
			}
			if gotMethod != tc.method || gotPath != tc.path {
				t.Fatalf("request %s %s, want %s %s", gotMethod, gotPath, tc.method, tc.path)
			}
		})
	}
}

func TestHumanOutputCommandsAcceptHubListEnvelopes(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		data any
	}{
		{"docker hosts", []string{"docker", "hosts"}, map[string]any{"hosts": []any{}}},
		{"docker hosts null", []string{"docker", "hosts"}, map[string]any{"hosts": nil}},
		{"docker containers", []string{"docker", "ps", "host-1"}, map[string]any{"containers": []any{}}},
		{"docker containers null", []string{"docker", "ps", "host-1"}, map[string]any{"containers": nil}},
		{"file listing", []string{"files", "ls", "agent-1", "/tmp"}, map[string]any{"entries": []any{}}},
		{"file listing null", []string{"files", "ls", "agent-1", "/tmp"}, map[string]any{"entries": nil}},
		{"process listing", []string{"ps", "list", "agent-1"}, map[string]any{"processes": []any{}}},
		{"process listing null", []string{"ps", "list", "agent-1"}, map[string]any{"processes": nil, "request_id": "agent-request"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(map[string]any{"data": tc.data})
			}))
			defer server.Close()
			if _, _, err := runConfiguredCmd(t, server.URL, tc.args...); err != nil {
				t.Fatalf("valid Hub response rejected: %v", err)
			}
		})
	}
}

func TestPackageUpdateSendsJSONBody(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Hub requires a JSON body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}})
	}))
	defer server.Close()
	if _, _, err := runConfiguredCmd(t, server.URL, "packages", "update", "agent-1"); err != nil {
		t.Fatalf("package update: %v", err)
	}
	if body == nil {
		t.Fatal("package update omitted JSON object")
	}
}

func TestConnectorGetSelectsDescriptorFromList(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
			"connectors": []map[string]any{{"id": "proxmox", "display_name": "Proxmox", "capabilities": map[string]any{}}},
		}})
	}))
	defer server.Close()
	_, _, err := runConfiguredCmd(t, server.URL, "connectors", "get", "proxmox")
	if err != nil || gotPath != "/api/v2/connectors" {
		t.Fatalf("path=%q err=%v", gotPath, err)
	}
	_, _, err = runConfiguredCmd(t, server.URL, "connectors", "get", "missing")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("missing connector error=%v", err)
	}
}

func TestIncidentChangesUsePatchStatus(t *testing.T) {
	for _, tc := range []struct{ command, status string }{
		{"investigate", "investigating"},
		{"resolve", "resolved"},
	} {
		t.Run(tc.command, func(t *testing.T) {
			var method, path, status string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				method, path = r.Method, r.URL.Path
				var body struct {
					Status string `json:"status"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Errorf("decode body: %v", err)
				}
				status = body.Status
				_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}})
			}))
			defer server.Close()
			_, _, err := runConfiguredCmd(t, server.URL, "incidents", tc.command, "incident-1")
			if err != nil || method != "PATCH" || path != "/api/v2/incidents/incident-1" || status != tc.status {
				t.Fatalf("method=%q path=%q status=%q err=%v", method, path, status, err)
			}
		})
	}
}

func TestAgentApprovalUsesPendingDecisionBody(t *testing.T) {
	for _, decision := range []string{"approve", "reject"} {
		t.Run(decision, func(t *testing.T) {
			var gotPath, assetID string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				var body struct {
					AssetID string `json:"asset_id"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Errorf("decode body: %v", err)
				}
				assetID = body.AssetID
				_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}})
			}))
			defer server.Close()
			_, _, err := runConfiguredCmd(t, server.URL, "agents", decision, "pending-1")
			if err != nil {
				t.Fatalf("command failed: %v", err)
			}
			if gotPath != "/api/v2/agents/pending/"+decision || assetID != "pending-1" {
				t.Fatalf("request path=%q asset_id=%q", gotPath, assetID)
			}
		})
	}
}

func TestHomeAssistantCallUsesEntityAction(t *testing.T) {
	var path, action, service string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		var body struct{ Action, Service string }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		action, service = body.Action, body.Service
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}})
	}))
	defer server.Close()
	_, _, err := runConfiguredCmd(t, server.URL, "ha", "call", "switch.office", "switch.turn_on")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if path != "/api/v2/homeassistant/entities/switch.office" || action != "service.call" || service != "switch.turn_on" {
		t.Fatalf("request path=%q action=%q service=%q", path, action, service)
	}
}

func TestFilesCatReadsRawBytes(t *testing.T) {
	want := "first line\nsecond line\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/assets/agent-1/files/read" || r.URL.Query().Get("path") != "/tmp/test.txt" {
			t.Errorf("unexpected request: %s", r.URL.RequestURI())
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write([]byte(want))
	}))
	defer server.Close()
	stdout, _, err := runConfiguredCmd(t, server.URL, "files", "cat", "agent-1", "/tmp/test.txt")
	if err != nil || stdout != want {
		t.Fatalf("cat stdout=%q err=%v", stdout, err)
	}
}

func TestProcessKillSendsNumericPID(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}})
	}))
	defer server.Close()
	_, _, err := runConfiguredCmd(t, server.URL, "ps", "kill", "agent-1", "123")
	if err != nil || body["pid"] != float64(123) {
		t.Fatalf("pid=%v err=%v", body["pid"], err)
	}
	_, _, err = runConfiguredCmd(t, server.URL, "ps", "kill", "agent-1", "bad")
	if err == nil || !strings.Contains(err.Error(), "positive integer") {
		t.Fatalf("invalid pid error=%v", err)
	}
}

func TestDockerActionsUseHubActionEndpoint(t *testing.T) {
	for _, verb := range []string{"start", "stop", "restart"} {
		t.Run(verb, func(t *testing.T) {
			var path, action string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path = r.URL.Path
				var body struct {
					Action string `json:"action"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Errorf("decode body: %v", err)
				}
				action = body.Action
				_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"result": map[string]any{"status": "succeeded"}}})
			}))
			defer server.Close()
			_, _, err := runConfiguredCmd(t, server.URL, "docker", verb, "container-1")
			if err != nil || path != "/api/v2/docker/containers/container-1/action" || action != "container."+verb {
				t.Fatalf("path=%q action=%q err=%v", path, action, err)
			}
		})
	}
}

func TestDockerActionReportsFailedResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"result": map[string]any{
			"status": "failed", "message": "agent offline",
		}}})
	}))
	defer server.Close()
	_, _, err := runConfiguredCmd(t, server.URL, "docker", "start", "container-1")
	if err == nil || !strings.Contains(err.Error(), "agent offline") {
		t.Fatalf("failed action error=%v", err)
	}
}

func TestProxmoxPowerActionUsesAssetKindAndChecksResult(t *testing.T) {
	for _, tc := range []struct{ assetType, action string }{
		{"vm", "vm.start"},
		{"container", "ct.start"},
	} {
		t.Run(tc.assetType, func(t *testing.T) {
			var actionPath, targetID string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "GET" {
					_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"asset": map[string]any{"source": "proxmox", "type": tc.assetType}}})
					return
				}
				actionPath = r.URL.Path
				var body struct {
					TargetID string `json:"target_id"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Errorf("decode body: %v", err)
				}
				targetID = body.TargetID
				_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"status": "succeeded"}})
			}))
			defer server.Close()
			_, _, err := runConfiguredCmd(t, server.URL, "proxmox", "start", "asset-1")
			if err != nil || actionPath != "/api/v2/connectors/proxmox/actions/"+tc.action+"/execute" || targetID != "asset-1" {
				t.Fatalf("path=%q target=%q err=%v", actionPath, targetID, err)
			}
		})
	}
}
