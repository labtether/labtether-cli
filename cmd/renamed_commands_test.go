package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Owner decision, 2026-11-03: 2.0 uses the console's names, and scripts
// written for 1.x keep working. Each old name must send the same Hub request
// and print the same output as its new name, with nothing on stderr. Help
// lists only the new names.
func TestOldCommandNamesStillRun(t *testing.T) {
	data := map[string]any{
		"/api/v2/assets": []any{map[string]any{"id": "nas01", "platform": "linux", "status": "online", "name": "nas01"}},
		"/api/v2/incidents": map[string]any{"incidents": []any{map[string]any{
			"id": "i1", "severity": "high", "status": "open", "title": "Disk full"}}},
	}
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.RequestURI())
		payload, ok := data[r.URL.Path]
		if !ok {
			payload = map[string]any{}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": payload})
	}))
	defer server.Close()
	for _, tc := range []struct{ old, current string }{
		{"assets list", "devices list"},
		{"assets get nas01", "devices get nas01"},
		{"incidents list", "issues list"},
		{"incidents ack i1", "issues investigate i1"},
		{"incidents resolve i1", "issues resolve i1"},
		{"topology edges pve01", "map connections pve01"},
		{"topology blast-radius pve01", "map downstream pve01"},
		{"topology upstream pve01", "map upstream pve01"},
		{"topology dependencies pve01", "map dependencies pve01"},
		{"metrics asset nas01", "metrics device nas01"},
	} {
		t.Run(tc.old, func(t *testing.T) {
			requests = nil
			oldOut, oldErr, err := runConfiguredCmd(t, server.URL, strings.Fields(tc.old)...)
			if err != nil {
				t.Fatalf("%s: %v", tc.old, err)
			}
			newOut, _, err := runConfiguredCmd(t, server.URL, strings.Fields(tc.current)...)
			if err != nil {
				t.Fatalf("%s: %v", tc.current, err)
			}
			if len(requests) != 2 || requests[0] != requests[1] || oldOut != newOut || oldErr != "" {
				t.Fatalf("requests %q; stdout %q, want %q; stderr %q", requests, oldOut, newOut, oldErr)
			}
		})
	}
	t.Run("help", func(t *testing.T) {
		for _, args := range [][]string{{"--help"}, {"devices", "--help"}, {"issues", "--help"}, {"map", "--help"}, {"metrics", "--help"}} {
			stdout, _, err := runCmd(t, args...)
			for _, old := range []string{"assets", "incidents", "topology", "blast-radius", "edges", "Aliases"} {
				if err != nil || strings.Contains(stdout, old) {
					t.Fatalf("%v help shows %q (err %v): %s", args, old, err, stdout)
				}
			}
		}
	})
}
