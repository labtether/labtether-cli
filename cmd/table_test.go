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
// print <nil> for a container without published ports.
func TestTablesPrintWholeNumbersAndEmptyCells(t *testing.T) {
	server := serveHubData(t, map[string]any{
		"/api/v2/assets/node-1/processes": map[string]any{"processes": []any{map[string]any{
			"pid": 1942945, "user": "root", "cpu_pct": 12.345, "mem_pct": 0.5, "command": "/usr/sbin/sshd -D",
		}}},
		"/api/v2/docker/hosts/node-1/containers": map[string]any{"containers": []any{map[string]any{
			"id": "web-1", "image": "nginx:1.27", "status": "Up 2 hours", "name": "web",
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
}
