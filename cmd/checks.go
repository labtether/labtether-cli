package cmd

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

var checksCmd = &cobra.Command{
	Use:   "checks",
	Short: "View health checks",
}

var checksListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all checks",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/checks")
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var checksGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get check details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/checks/" + args[0])
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
