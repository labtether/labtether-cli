package cmd

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Show device metrics",
}

var metricsOverviewCmd = &cobra.Command{
	Use:   "overview",
	Short: "Show metrics for all devices",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/metrics/overview")
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

// metrics device was metrics asset before 2.0; the old name stays as a hidden alias.
var metricsDeviceCmd = &cobra.Command{
	Use:     "device <device>",
	Aliases: []string{"asset"},
	Short:   "Show metrics for one device",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/assets/" + pathSegment(args[0]) + "/metrics")
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

func init() {
	metricsCmd.AddCommand(metricsOverviewCmd, metricsDeviceCmd)
	rootCmd.AddCommand(metricsCmd)
}
