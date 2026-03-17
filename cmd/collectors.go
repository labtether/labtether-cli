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
		printJSON(json.RawMessage(resp.Data))
		return nil
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
		resp, err := c.Get("/api/v2/collectors/" + args[0])
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
