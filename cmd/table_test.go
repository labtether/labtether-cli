package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func serveHubData(t *testing.T, data map[string]any) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload, ok := data[r.URL.Path]
		if !ok {
			t.Errorf("unexpected request %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": payload})
	}))
	t.Cleanup(server.Close)
	return server
}

// Lab QA saw `ps list` print PID 1942945 as 1.942945e+06 and `docker ps`
// print <nil> for a container without published ports. `docker ps` also
// printed the 64-character ID instead of the short ID other commands take.
func TestTablesPrintWholeNumbersAndEmptyCells(t *testing.T) {
	const dockerID = "4f3c2b1a0d9e8f7a6b5c4d3e2f1a0b9c8d7e6f5a4b3c2d1e0f9a8b7c6d5e4f3a"
	server := serveHubData(t, map[string]any{
		"/api/v2/assets/node-1/processes": map[string]any{"processes": []any{map[string]any{
			"pid": 1942945, "user": "root", "cpu_pct": 12.345, "mem_pct": 0.5, "command": "/usr/sbin/sshd -D",
		}}},
		"/api/v2/docker/hosts/node-1/containers": map[string]any{"containers": []any{map[string]any{
			"id": dockerID, "image": "nginx:1.27", "status": "Up 2 hours", "name": "web",
		}}},
	})
	stdout, _, err := runConfiguredCmd(t, server.URL, "ps", "list", "node-1")
	if err != nil {
		t.Fatalf("ps list: %v", err)
	}
	fields := strings.Fields(strings.Split(strings.TrimSpace(stdout), "\n")[1])
	if strings.Join(fields[:4], " ") != "1942945 root 12.35 0.5" {
		t.Fatalf("process row = %q", stdout)
	}
	stdout, _, err = runConfiguredCmd(t, server.URL, "docker", "ps", "node-1")
	if err != nil {
		t.Fatalf("docker ps: %v", err)
	}
	if strings.Contains(stdout, "<nil>") || !strings.Contains(stdout, " - ") {
		t.Fatalf("missing ports not shown as -: %q", stdout)
	}
	if !strings.Contains(stdout, "\n"+dockerID[:12]+" ") || strings.Contains(stdout, dockerID) {
		t.Fatalf("container ID not shortened: %q", stdout)
	}
}

// Lab QA saw these list commands print raw JSON. They print a table, and
// --json still prints the Hub payload for scripts.
func TestListCommandsPrintTablesAndKeepJSON(t *testing.T) {
	server := serveHubData(t, map[string]any{
		"/api/v2/agents": map[string]any{"count": 1, "assets": []any{"node-1"},
			"assetsInfo": []any{map[string]any{"id": "node-1", "platform": "linux", "has_tmux": true}}},
		"/api/v2/assets/node-1/services": map[string]any{"request_id": "r1", "services": []any{map[string]any{
			"name": "nginx.service", "description": "Web server", "active_state": "active",
			"sub_state": "running", "enabled": "enabled", "load_state": "loaded",
		}}},
		"/api/v2/web-services": map[string]any{"services": []any{map[string]any{
			"id": "ws-1", "name": "Grafana", "category": "Monitoring", "url": "http://192.0.2.10:3000",
			"status": "up", "response_ms": 12, "host_asset_id": "node-1",
		}}},
	})
	for _, tc := range []struct {
		args   []string
		header string
		row    string
	}{
		{[]string{"agents", "list"}, "ID PLATFORM", "node-1 linux"},
		{[]string{"services", "list", "node-1"}, "NAME ACTIVE SUB ENABLED DESCRIPTION", "nginx.service active running enabled Web server"},
		{[]string{"web-services", "list"}, "NAME CATEGORY STATUS HOST URL", "Grafana Monitoring up node-1 http://192.0.2.10:3000"},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			stdout, _, err := runConfiguredCmd(t, server.URL, tc.args...)
			lines := strings.Split(strings.TrimSpace(stdout), "\n")
			if err != nil || len(lines) != 2 ||
				strings.Join(strings.Fields(lines[0]), " ") != tc.header ||
				strings.Join(strings.Fields(lines[1]), " ") != tc.row {
				t.Fatalf("table output = %q, err = %v", stdout, err)
			}
			stdout, _, err = runConfiguredCmd(t, server.URL, append([]string{"--json"}, tc.args...)...)
			var payload map[string]any
			if err != nil || json.Unmarshal([]byte(stdout), &payload) != nil {
				t.Fatalf("--json output = %q, err = %v", stdout, err)
			}
		})
	}
}

// Lab QA saw `search` print {} when nothing matched.
func TestSearchPrintsResultsOrNoResults(t *testing.T) {
	for _, tc := range []struct {
		name string
		data map[string]any
		want string
	}{
		{"no match", map[string]any{}, "No results."},
		{"matches", map[string]any{
			"assets": []any{map[string]any{"id": "nas01", "name": "nas01", "type": "asset", "status": "online", "platform": "linux"}},
			"groups": []any{map[string]any{"id": "g1", "name": "Storage", "type": "group"}},
		}, "TYPE ID NAME STATUS PLATFORM\nasset nas01 nas01 online linux\ngroup g1 Storage - -"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := serveHubData(t, map[string]any{"/api/v2/search": tc.data})
			stdout, _, err := runConfiguredCmd(t, server.URL, "search", "nas")
			lines := strings.Split(strings.TrimSpace(stdout), "\n")
			for i, line := range lines {
				lines[i] = strings.Join(strings.Fields(line), " ")
			}
			if err != nil || strings.Join(lines, "\n") != tc.want {
				t.Fatalf("search output = %q, err = %v", stdout, err)
			}
		})
	}
}

// `pbs datastores` called /datastores, a route the Hub does not serve, so it
// always failed with 404. The Hub returns the list in the details payload.
func TestPBSDatastoresReadsDetailsList(t *testing.T) {
	server := serveHubData(t, map[string]any{"/api/v2/pbs/assets/pbs-1/details": map[string]any{
		"asset_id": "pbs-1", "kind": "server", "datastores": []any{map[string]any{
			"store": "main", "status": "online", "total_bytes": 1099511627776, "used_bytes": 549755813888,
			"avail_bytes": 549755813888, "usage_percent": 50, "group_count": 4, "snapshot_count": 120,
		}},
	}})
	stdout, _, err := runConfiguredCmd(t, server.URL, "pbs", "datastores", "pbs-1")
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if err != nil || len(lines) != 2 || strings.Join(strings.Fields(lines[1]), " ") !=
		"main online 1.0 TiB 512.0 GiB 512.0 GiB 50 4 120 -" {
		t.Fatalf("table output = %q, err = %v", stdout, err)
	}
	stdout, _, err = runConfiguredCmd(t, server.URL, "--json", "pbs", "datastores", "pbs-1")
	var list []map[string]any
	if err != nil || json.Unmarshal([]byte(stdout), &list) != nil || len(list) != 1 || list[0]["store"] != "main" {
		t.Fatalf("--json output = %q, err = %v", stdout, err)
	}
}
