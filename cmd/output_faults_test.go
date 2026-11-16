package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// runAgainstHub runs the CLI once against a stand-in Hub that answers every
// request with status and body.
func runAgainstHub(t *testing.T, status int, body string, args ...string) (int, string, string) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()
	isolateConfig(t)
	t.Setenv("LABTETHER_HOST", server.URL)
	t.Setenv("LABTETHER_API_KEY", "test-key")
	t.Setenv("LABTETHER_TLS_CA_FILE", "")
	resetTestCommandState()
	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	code := executeArgs(args)
	return code, stdout.String(), stderr.String()
}

// Fault: when an answer held a number too large for float64, the JSON
// decoder failed and --json printed the answer raw, so a "password" or
// "token" field printed in clear text.
func TestJSONOutputRedactsAnswerWithNumberBeyondFloat64(t *testing.T) {
	body := `{"data":{"password":"pw-in-clear","nested":{"access_token":"tok-in-clear"},"size":1e400,"name":"nas01"}}`
	code, stdout, stderr := runAgainstHub(t, 200, body, "--json", "tls", "status")
	for _, leaked := range []string{"pw-in-clear", "tok-in-clear"} {
		if strings.Contains(stdout+stderr, leaked) {
			t.Fatalf("secret %q printed: code=%d stdout=%q stderr=%q", leaked, code, stdout, stderr)
		}
	}
	if code != 0 || !strings.Contains(stdout, "[redacted]") || !strings.Contains(stdout, "1e400") || !strings.Contains(stdout, "nas01") {
		t.Fatalf("answer not printed redacted with its number kept: code=%d stdout=%q", code, stdout)
	}
}

// Fault: an envelope the client could not decode (a number beyond the range
// of its "status" or "meta" fields) was echoed raw in the error text, on
// stderr or in the --json "error" field, secrets included.
func TestUnreadableEnvelopeErrorRedactsSecrets(t *testing.T) {
	for _, body := range []string{
		`{"data":{"password":"pw-in-clear"},"status":1e400}`,
		`{"data":{"api_key":"key-in-clear"},"meta":{"total":1.5}}`,
	} {
		for _, json := range []bool{false, true} {
			args := []string{"tls", "status"}
			if json {
				args = append([]string{"--json"}, args...)
			}
			code, stdout, stderr := runAgainstHub(t, 200, body, args...)
			if strings.Contains(stdout+stderr, "-in-clear") || code != exitFailed {
				t.Fatalf("body %s json=%v: code=%d stdout=%q stderr=%q", body, json, code, stdout, stderr)
			}
			if !strings.Contains(stdout+stderr, "[redacted]") {
				t.Fatalf("body %s json=%v: error lost the answer: stdout=%q stderr=%q", body, json, stdout, stderr)
			}
		}
	}
}

// Fault: a table cell holding an object printed it as JSON without
// redaction, so a nested secret showed in table output.
func TestTableObjectCellRedactsSecrets(t *testing.T) {
	text := cellText(map[string]any{"user": "admin", "password": "pw-in-clear"})
	if strings.Contains(text, "pw-in-clear") || !strings.Contains(text, "admin") {
		t.Fatalf("cell text = %q", text)
	}
}

// Fault: the exit code was chosen by matching "(status 404)" and similar in
// the error text, so a Hub message could change it. Real answers must keep
// their exit codes: 401 and 403 exit 2, 404 and 409 exit 3, others exit 1.
func TestExitCodeFollowsHTTPStatusNotMessageText(t *testing.T) {
	for _, tc := range []struct {
		status int
		body   string
		want   int
	}{
		{400, `{"error":"bad_request"}`, exitFailed},
		{401, `{"error":"unauthorized"}`, exitAuth},
		{403, `{"error":"forbidden","message":"scope denied"}`, exitAuth},
		{404, `{"error":"not_found"}`, exitNotFound},
		{405, `{"error":"method not allowed"}`, exitFailed},
		{409, `{"error":"conflict","message":"device offline"}`, exitNotFound},
		{410, `{"error":"gone"}`, exitFailed},
		{422, `{"error":"invalid"}`, exitFailed},
		{429, `{"error":"rate limited"}`, exitFailed},
		{500, `{"error":"internal"}`, exitFailed},
		{502, `bad gateway`, exitFailed},
		{503, `{"error":"unavailable"}`, exitFailed},
		// The message words must not change the exit code.
		{500, `{"message":"lookup failed (status 404)"}`, exitFailed},
		{500, `{"message":"upstream said (status 401)"}`, exitFailed},
		{404, `{"message":"upstream said (status 401)"}`, exitNotFound},
		{502, `proxy error (status 404)`, exitFailed},
		{200, `{"data":{"result":{"status":"failed","message":"engine said (status 403)"}}}`, exitFailed},
	} {
		args := []string{"tls", "status"}
		if tc.status == 200 {
			args = []string{"docker", "restart", "container-1"}
		}
		code, _, stderr := runAgainstHub(t, tc.status, tc.body, args...)
		t.Logf("status %d body %s -> exit %d", tc.status, tc.body, code)
		if code != tc.want {
			t.Errorf("status %d body %s: exit %d, want %d (stderr %q)", tc.status, tc.body, code, tc.want, stderr)
		}
	}
}
