package cmd

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

var hubCmd = &cobra.Command{
	Use:   "hub",
	Short: "View hub status and configuration",
}

var hubStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show hub status",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/hub/status")
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var hubAgentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "Show agents connected to the hub",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/hub/agents")
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

func init() {
	hubCmd.AddCommand(hubStatusCmd, hubAgentsCmd)
	rootCmd.AddCommand(hubCmd)
}
