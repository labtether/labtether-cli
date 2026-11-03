package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var connectorsCmd = &cobra.Command{
	Use:   "connectors",
	Short: "List and test hub connectors",
}

var connectorsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List connectors",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/connectors")
		if err != nil {
			return err
		}
		return printList(resp, "connectors",
			col("ID", "id"),
			col("NAME", "display_name"))
	},
}

var connectorsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Show one connector",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/connectors")
		if err != nil {
			return err
		}
		var result struct {
			Connectors []struct {
				ID           string          `json:"id"`
				DisplayName  string          `json:"display_name"`
				Capabilities json.RawMessage `json:"capabilities"`
			} `json:"connectors"`
		}
		if err := json.Unmarshal(resp.Data, &result); err != nil || result.Connectors == nil {
			return fmt.Errorf("decode connector list: invalid response")
		}
		for _, connector := range result.Connectors {
			if connector.ID == args[0] {
				printJSON(connector)
				return nil
			}
		}
		return fmt.Errorf("connector %q not found", args[0])
	},
}

var connectorsTestCmd = &cobra.Command{
	Use:   "test <connector-or-collector-id>",
	Short: "Test a saved connection",
	Long: "Test the saved connection for a connector, such as proxmox or truenas. " +
		"When a connector has more than one saved connection, give its collector ID, as shown by collectors list.",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		connector, body, err := connectorTestRequest(c, args[0])
		if err != nil {
			return err
		}
		resp, err := c.Post(fmt.Sprintf("/api/v2/connectors/%s/test", pathSegment(connector)), body)
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

func init() {
	connectorsCmd.AddCommand(connectorsListCmd, connectorsGetCmd, connectorsTestCmd)
	rootCmd.AddCommand(connectorsCmd)
}
