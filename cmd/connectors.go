package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var connectorsCmd = &cobra.Command{
	Use:   "connectors",
	Short: "Manage hub connectors",
}

var connectorsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all connectors",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/connectors")
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var connectorsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get connector details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/connectors/" + args[0])
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var connectorsTestCmd = &cobra.Command{
	Use:   "test <id>",
	Short: "Test a connector connection",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Post(fmt.Sprintf("/api/v2/connectors/%s/test", args[0]), nil)
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
