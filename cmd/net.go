package cmd

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

var netCmd = &cobra.Command{
	Use:   "net <asset>",
	Short: "Show network info for an asset",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/assets/" + args[0] + "/network")
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(netCmd)
}
