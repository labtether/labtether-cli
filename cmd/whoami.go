package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show API key info, scopes, and accessible assets",
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
		json.Unmarshal(resp.Data, &data)

		fmt.Printf("Auth:     %v\n", data["auth_type"])
		fmt.Printf("Role:     %v\n", data["role"])
		if name, ok := data["key_name"]; ok {
			fmt.Printf("Key:      %v\n", name)
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
		if assets, ok := data["available_assets"].([]any); ok {
			fmt.Printf("Assets:   %d accessible\n", len(assets))
			for _, a := range assets {
				asset := a.(map[string]any)
				status := "offline"
				if asset["online"] == true {
					status = "online"
				}
				fmt.Printf("  %-20s %-10s %s\n", asset["id"], asset["platform"], status)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(whoamiCmd)
}
