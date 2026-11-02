package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/labtether/labtether-cli/internal/client"
)

// The Hub's connector test checks the connection settings sent with it and
// fails on an empty request. A saved connection keeps those settings in its
// collector config; its secret stays in the Hub under credential_id. The CLI
// sends only these non-secret settings and the Hub adds the secret.
var connectorTestSettings = []string{
	"base_url", "credential_id", "auth_method", "token_id", "username", "skip_verify", "ca_pem",
}

type savedConnection struct {
	ID     string         `json:"id"`
	Type   string         `json:"collector_type"`
	Config map[string]any `json:"config"`
}

// Collectors name Home Assistant without the hyphen the connector list uses.
func collectorTypeFor(connector string) string {
	if connector == "home-assistant" {
		return "homeassistant"
	}
	return connector
}

func connectorFor(collectorType string) string {
	if collectorType == "homeassistant" {
		return "home-assistant"
	}
	return collectorType
}

// connectorTestRequest finds the saved connection named by target, either its
// collector ID or a connector ID with one saved connection, and returns the
// connector to test with that connection's settings.
func connectorTestRequest(c *client.Client, target string) (string, map[string]any, error) {
	// The Docker test checks the Hub's own Docker connection and takes no settings.
	if target == "docker" {
		return target, map[string]any{}, nil
	}
	resp, err := c.Get("/api/v2/collectors")
	if err != nil {
		return "", nil, err
	}
	var data struct {
		Collectors []savedConnection `json:"collectors"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return "", nil, fmt.Errorf("decode saved connections: %w", err)
	}
	var matches []savedConnection
	for _, saved := range data.Collectors {
		if saved.ID == target {
			matches = []savedConnection{saved}
			break
		}
		if saved.Type == collectorTypeFor(target) {
			matches = append(matches, saved)
		}
	}
	switch len(matches) {
	case 0:
		return "", nil, fmt.Errorf("no saved %s connection to test", target)
	case 1:
	default:
		ids := make([]string, len(matches))
		for i, saved := range matches {
			ids[i] = saved.ID
		}
		return "", nil, fmt.Errorf("%d saved %s connections; test one by its ID: %s",
			len(matches), target, strings.Join(ids, ", "))
	}
	body := map[string]any{}
	for _, key := range connectorTestSettings {
		if value, ok := matches[0].Config[key]; ok {
			body[key] = value
		}
	}
	return connectorFor(matches[0].Type), body, nil
}
