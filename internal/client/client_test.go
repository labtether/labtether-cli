package client

import (
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewWithTLSCAFileTrustsPrivateCA(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"request_id": "req_tls",
			"data":       map[string]string{"status": "ok"},
		})
	}))
	defer srv.Close()

	caPath := filepath.Join(t.TempDir(), "hub-ca.crt")
	pemData := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})
	if err := os.WriteFile(caPath, pemData, 0o644); err != nil {
		t.Fatal(err)
	}

	c, err := NewWithTLSCAFile(srv.URL, "test-key", caPath)
	if err != nil {
		t.Fatalf("NewWithTLSCAFile: %v", err)
	}
	resp, err := c.Get("/api/v2/whoami")
	if err != nil {
		t.Fatalf("private-CA request failed: %v", err)
	}
	if resp.RequestID != "req_tls" {
		t.Fatalf("request ID = %q, want req_tls", resp.RequestID)
	}
}

func TestValidateTLSCAFileRejectsInvalidPEM(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid.pem")
	if err := os.WriteFile(path, []byte("not a certificate"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ValidateTLSCAFile(path); err == nil || !strings.Contains(err.Error(), "no valid PEM") {
		t.Fatalf("expected invalid PEM error, got %v", err)
	}
}

func TestValidateBaseURLRejectsPlaintextNonLoopback(t *testing.T) {
	if err := ValidateBaseURL("http://192.0.2.10:8080"); err == nil {
		t.Fatal("expected plaintext non-loopback URL to be rejected")
	}
	for _, raw := range []string{"http://127.0.0.1:8080", "http://[::1]:8080", "https://hub.example.com"} {
		if err := ValidateBaseURL(raw); err != nil {
			t.Fatalf("expected %s to be accepted: %v", raw, err)
		}
	}
}

func TestClientRejectsOversizedResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":"`)
		chunk := make([]byte, 1024*1024)
		for i := range chunk {
			chunk[i] = 'a'
		}
		for range 17 {
			_, _ = w.Write(chunk)
		}
		_, _ = fmt.Fprint(w, `"}`)
	}))
	defer srv.Close()

	_, err := New(srv.URL, "test-key").Get("/large")
	if err == nil || !strings.Contains(err.Error(), "response exceeds") {
		t.Fatalf("expected bounded response error, got %v", err)
	}
}

type countingWriter struct{ n int64 }

func (w *countingWriter) Write(data []byte) (int, error) {
	w.n += int64(len(data))
	return len(data), nil
}

func TestDownloadStreamsFilesLargerThanJSONLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("missing authorization")
		}
		chunk := make([]byte, 1024*1024)
		for range 17 {
			_, _ = w.Write(chunk)
		}
	}))
	defer srv.Close()

	var output countingWriter
	if err := New(srv.URL, "test-key").Download("/api/v2/assets/agent/files/read", &output); err != nil {
		t.Fatalf("Download: %v", err)
	}
	if output.n != 17*1024*1024 {
		t.Fatalf("downloaded %d bytes", output.n)
	}
}

func TestDownloadDoesNotWriteErrorBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"message": "file missing"})
	}))
	defer srv.Close()

	var output countingWriter
	err := New(srv.URL, "test-key").Download("/api/v2/assets/agent/files/read", &output)
	if err == nil || !strings.Contains(err.Error(), "file missing") || output.n != 0 {
		t.Fatalf("error=%v, wrote %d bytes", err, output.n)
	}
}

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

func TestClient_NormalizesBaseURLAndAPIKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/whoami" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		json.NewEncoder(w).Encode(map[string]any{
			"request_id": "req_test",
			"data":       map[string]string{"role": "admin"},
		})
	}))
	defer srv.Close()

	c := New(" "+srv.URL+"/ ", " test-key ")
	if c.BaseURL != srv.URL {
		t.Fatalf("BaseURL = %q, want %q", c.BaseURL, srv.URL)
	}
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
