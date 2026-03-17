package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var alertsCmd = &cobra.Command{
	Use:   "alerts",
	Short: "Manage alerts",
}

var alertsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List active alerts",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/alerts")
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var alertsAckCmd = &cobra.Command{
	Use:   "ack <alert>",
	Short: "Acknowledge an alert",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		_, err = c.Post(fmt.Sprintf("/api/v2/alerts/%s/ack", args[0]), nil)
		if err != nil {
			return err
		}
		fmt.Printf("Alert %s acknowledged\n", args[0])
		return nil
	},
}

var alertsSilenceCmd = &cobra.Command{
	Use:   "silence <alert>",
	Short: "Silence an alert",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		_, err = c.Post(fmt.Sprintf("/api/v2/alerts/%s/silence", args[0]), nil)
		if err != nil {
			return err
		}
		fmt.Printf("Alert %s silenced\n", args[0])
		return nil
	},
}

func init() {
	alertsCmd.AddCommand(alertsListCmd, alertsAckCmd, alertsSilenceCmd)
	rootCmd.AddCommand(alertsCmd)
}
