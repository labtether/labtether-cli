package cmd

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "View hub and asset metrics",
}

var metricsOverviewCmd = &cobra.Command{
	Use:   "overview",
	Short: "Show fleet-wide metrics overview",
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

var metricsAssetCmd = &cobra.Command{
	Use:   "asset <asset>",
	Short: "Show metrics for a specific asset",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/assets/" + args[0] + "/metrics")
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

func init() {
	metricsCmd.AddCommand(metricsOverviewCmd, metricsAssetCmd)
	rootCmd.AddCommand(metricsCmd)
}
