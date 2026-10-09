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
		resp, err := c.Get("/api/v2/failover-pairs")
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
		resp, err := c.Get("/api/v2/failover-pairs/" + pathSegment(args[0]))
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var failoverReadinessCmd = &cobra.Command{
	Use:   "check-readiness <id>",
	Short: "Check whether a failover pair is ready",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Post(fmt.Sprintf("/api/v2/failover-pairs/%s/check-readiness", pathSegment(args[0])), nil)
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

func init() {
	failoverCmd.AddCommand(failoverListCmd, failoverGetCmd, failoverReadinessCmd)
	rootCmd.AddCommand(failoverCmd)
}
