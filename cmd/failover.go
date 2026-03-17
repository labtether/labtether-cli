package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var failoverCmd = &cobra.Command{
	Use:   "failover",
	Short: "Manage failover configurations",
}

var failoverListCmd = &cobra.Command{
	Use:   "list",
	Short: "List failover configurations",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/failover")
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var failoverGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get failover configuration details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/failover/" + args[0])
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var failoverTriggerCmd = &cobra.Command{
	Use:   "trigger <id>",
	Short: "Manually trigger a failover",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		_, err = c.Post(fmt.Sprintf("/api/v2/failover/%s/trigger", args[0]), nil)
		if err != nil {
			return err
		}
		fmt.Printf("Failover %s triggered\n", args[0])
		return nil
	},
}

func init() {
	failoverCmd.AddCommand(failoverListCmd, failoverGetCmd, failoverTriggerCmd)
	rootCmd.AddCommand(failoverCmd)
}
