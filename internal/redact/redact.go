// Package redact hides secret values in Hub answers before the CLI prints
// them, in JSON output, table cells and error text.
package redact

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

// Hidden replaces a secret value.
const Hidden = "[redacted]"

// Decode reads JSON as the CLI has always read it, with numbers as float64,
// so printed numbers keep their usual form. A number beyond float64 cannot be
// read that way; the answer is then read with each number's text kept, so it
// is still redacted instead of printed raw.
func Decode(data []byte) (any, error) {
	var value any
	if err := json.Unmarshal(data, &value); err == nil {
		return value, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var exact any
	if err := decoder.Decode(&exact); err != nil {
		return nil, err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("invalid JSON after the top-level value")
	}
	return exact, nil
}

// Value replaces secret values in a decoded JSON value, at every depth, and
// reports how many it replaced. Maps and slices are changed in place.
func Value(v any) (any, int) {
	hidden := 0
	var walk func(any) any
	walk = func(v any) any {
		switch value := v.(type) {
		case map[string]any:
			for key, child := range value {
				if SensitiveKey(key) && !safeSecretStatus(child) {
					value[key] = Hidden
					hidden++
					continue
				}
				value[key] = walk(child)
			}
		case []any:
			for i, child := range value {
				value[i] = walk(child)
			}
		}
		return v
	}
	return walk(v), hidden
}

// SensitiveKey reports whether a JSON key names a secret.
func SensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "-", "_"), " ", "_"))
	return MentionsSecret(normalized) || normalized == "apikey"
}

// MentionsSecret reports whether text holds a word that names a secret field.
func MentionsSecret(text string) bool {
	text = strings.ToLower(text)
	for _, word := range []string{"password", "secret", "token", "api_key", "private_key"} {
		if strings.Contains(text, word) {
			return true
		}
	}
	return false
}

// safeSecretStatus reports values that say whether a secret is set without
// showing it.
func safeSecretStatus(v any) bool {
	switch value := v.(type) {
	case string:
		return value == "(set)" || value == "(not set)" || value == Hidden
	case bool:
		return true
	default:
		return false
	}
}
