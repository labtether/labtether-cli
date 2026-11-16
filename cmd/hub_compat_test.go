package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// The command-to-route cases live in testdata/hub-routes.json, which the
// Rust CLI's tests read too, so a command cannot be added to one CLI
// without the other.
type sharedRouteTable struct {
	Routes []struct {
		Name   string   `json:"name"`
		Args   []string `json:"args"`
		Method string   `json:"method"`
		Path   string   `json:"path"`
		Data   any      `json:"data"`
	} `json:"routes"`
	Local         []string `json:"local"`
	ListEnvelopes []struct {
		Name string   `json:"name"`
		Args []string `json:"args"`
		Data any      `json:"data"`
	} `json:"list_envelopes"`
	Renamed []struct {
		Old     string `json:"old"`
		Current string `json:"current"`
	} `json:"renamed"`
}

func loadSharedRoutes(t *testing.T) sharedRouteTable {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "testdata", "hub-routes.json"))
	if err != nil {
		t.Fatalf("read shared route table: %v", err)
	}
	var table sharedRouteTable
	if err := json.Unmarshal(data, &table); err != nil {
		t.Fatalf("parse shared route table: %v", err)
	}
	return table
}

func TestCommandsUseCurrentHubRoutes(t *testing.T) {
	for _, tc := range loadSharedRoutes(t).Routes {
		t.Run(tc.Name, func(t *testing.T) {
			var gotMethod, gotPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod, gotPath = r.Method, r.URL.RequestURI()
				_ = json.NewEncoder(w).Encode(map[string]any{"data": tc.Data})
			}))
			defer server.Close()
			_, _, err := runConfiguredCmd(t, server.URL, tc.Args...)
			if err != nil {
				t.Fatalf("command failed: %v", err)
			}
			if gotMethod != tc.Method || gotPath != tc.Path {
				t.Fatalf("request %s %s, want %s %s", gotMethod, gotPath, tc.Method, tc.Path)
			}
		})
	}
}

// Every runnable command must have a case in the shared route table, or be
// listed there as local, so the Rust CLI's tests see it too.
func TestRouteTableCoversEveryCommand(t *testing.T) {
	table := loadSharedRoutes(t)
	covered := map[*cobra.Command]bool{}
	for _, tc := range table.Routes {
		cmd, _, err := rootCmd.Find(tc.Args)
		if err != nil || cmd == rootCmd {
			t.Fatalf("route %q names no command: %v", tc.Name, err)
		}
		covered[cmd] = true
	}
	local := map[string]bool{}
	for _, name := range table.Local {
		local[name] = true
	}
	var walk func(*cobra.Command)
	walk = func(cmd *cobra.Command) {
		for _, child := range cmd.Commands() {
			name := strings.TrimPrefix(child.CommandPath(), rootCmd.Name()+" ")
			if child.Runnable() && !child.Hidden && !covered[child] && !local[name] {
				t.Errorf("command %q has no case in testdata/hub-routes.json", name)
			}
			walk(child)
		}
	}
	walk(rootCmd)
}

func TestHumanOutputCommandsAcceptHubListEnvelopes(t *testing.T) {
	for _, tc := range loadSharedRoutes(t).ListEnvelopes {
		t.Run(tc.Name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(map[string]any{"data": tc.Data})
			}))
			defer server.Close()
			if _, _, err := runConfiguredCmd(t, server.URL, tc.Args...); err != nil {
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

func TestIssueChangesUsePatchStatus(t *testing.T) {
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
			_, _, err := runConfiguredCmd(t, server.URL, "issues", tc.command, "incident-1")
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
