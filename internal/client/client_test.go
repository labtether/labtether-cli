package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_Get(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("missing or wrong auth header")
		}
		if r.URL.Path != "/api/v2/whoami" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"request_id": "req_test",
			"data":       map[string]string{"role": "admin"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-key")
	resp, err := c.Get("/api/v2/whoami")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if resp.RequestID != "req_test" {
		t.Errorf("request_id = %q, want req_test", resp.RequestID)
	}
}

func TestClient_Post(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("missing content-type")
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]any{
			"request_id": "req_exec",
			"data":       map[string]any{"exit_code": 0, "stdout": "ok"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-key")
	resp, err := c.Post("/api/v2/assets/srv1/exec", map[string]string{"command": "uptime"})
	if err != nil {
		t.Fatalf("Post error: %v", err)
	}
	if resp.RequestID != "req_exec" {
		t.Error("wrong request_id")
	}
}

func TestClient_ErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		json.NewEncoder(w).Encode(map[string]any{
			"request_id": "req_err",
			"error":      "insufficient_scope",
			"message":    "api key lacks required scope",
			"status":     403,
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-key")
	_, err := c.Get("/api/v2/assets")
	if err == nil {
		t.Fatal("should return error for 403")
	}
}
