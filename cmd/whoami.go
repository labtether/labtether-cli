package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show this API key's role, scopes and devices",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}

		resp, err := c.Get("/api/v2/whoami")
		if err != nil {
			return err
		}

		if jsonOutput {
			printJSON(json.RawMessage(resp.Data))
			return nil
		}

		var data map[string]any
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		fmt.Printf("Auth:     %s\n", cleanCell(cellText(data["auth_type"])))
		fmt.Printf("Role:     %s\n", cleanCell(cellText(data["role"])))
		if name, ok := data["key_name"]; ok {
			fmt.Printf("Key:      %s\n", cleanCell(cellText(name)))
		}
		if scopes, ok := data["scopes"].([]any); ok {
			fmt.Printf("Scopes:   ")
			for i, s := range scopes {
				if i > 0 {
					fmt.Print(", ")
				}
				fmt.Print(s)
			}
			fmt.Println()
		}
		if devices, ok := data["available_assets"].([]any); ok {
			fmt.Printf("Devices:  %d\n", len(devices))
			for _, d := range devices {
				device, ok := d.(map[string]any)
				if !ok {
					continue
				}
				status := "Offline"
				if device["online"] == true {
					status = "Online"
				}
				fmt.Printf("  %-20s %-10s %s\n", cleanCell(cellText(device["id"])), cleanCell(cellText(device["platform"])), status)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(whoamiCmd)
}
