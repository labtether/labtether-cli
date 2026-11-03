package cmd

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

var checksCmd = &cobra.Command{
	Use:   "checks",
	Short: "Show service checks",
}

var checksListCmd = &cobra.Command{
	Use:   "list",
	Short: "List service checks",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/synthetic-checks")
		if err != nil {
			return err
		}
		return printList(resp, "checks",
			col("ID", "id"),
			col("NAME", "name"),
			col("TYPE", "check_type"),
			col("ENABLED", "enabled"),
			colFormat("LAST STATUS", "last_status", statusWord),
			col("TARGET", "target"))
	},
}

var checksGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Show one check",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/synthetic-checks/" + pathSegment(args[0]))
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

func init() {
	checksCmd.AddCommand(checksListCmd, checksGetCmd)
	rootCmd.AddCommand(checksCmd)
}
