package cmd

import (
	"bytes"
	"encoding/json"
	"testing"
)

// JSON output is a stable machine contract: keep indentation, HTML escaping,
// the trailing newline and secret redaction when the writer changes.
func TestWriteJSONKeepsFormatAndRedactsSecrets(t *testing.T) {
	var output bytes.Buffer
	writeJSON(&output, json.RawMessage(`{"name":"<node>","nested":{"api_key":"do-not-leak","token_configured":true},"values":[1,null]}`))
	want := "{\n  \"name\": \"\\u003cnode\\u003e\",\n  \"nested\": {\n    \"api_key\": \"[redacted]\",\n    \"token_configured\": true\n  },\n  \"values\": [\n    1,\n    null\n  ]\n}\n"
	if output.String() != want {
		t.Fatalf("JSON output = %q, want %q", output.String(), want)
	}
}
