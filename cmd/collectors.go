package cmd

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

var collectorsCmd = &cobra.Command{
	Use:   "collectors",
	Short: "View data collectors",
}

var collectorsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all collectors",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/collectors")
		if err != nil {
			return err
		}
		return printList(resp, "collectors",
			col("ID", "id"),
			col("TYPE", "collector_type"),
			col("ASSET", "asset_id"),
			col("ENABLED", "enabled"),
			col("LAST STATUS", "last_status"),
			colFormat("LAST RUN", "last_collected_at", timeText))
	},
}

var collectorsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get collector details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/collectors/" + pathSegment(args[0]))
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

func init() {
	collectorsCmd.AddCommand(collectorsListCmd, collectorsGetCmd)
	rootCmd.AddCommand(collectorsCmd)
}
