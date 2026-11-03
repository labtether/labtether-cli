package cmd

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

var truenasCmd = &cobra.Command{
	Use:   "truenas",
	Short: "Show TrueNAS pools, datasets and shares",
}

var truenasGetCmd = &cobra.Command{
	Use:   "get <device>",
	Short: "Show one TrueNAS system",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/truenas/assets/" + pathSegment(args[0]) + "/overview")
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var truenasPoolsCmd = &cobra.Command{
	Use:   "pools <device>",
	Short: "List pools on a TrueNAS system",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/truenas/assets/" + pathSegment(args[0]) + "/pools")
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var truenasDatasetsCmd = &cobra.Command{
	Use:   "datasets <device>",
	Short: "List datasets on a TrueNAS system",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/truenas/assets/" + pathSegment(args[0]) + "/datasets")
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var truenasSharesCmd = &cobra.Command{
	Use:   "shares <device>",
	Short: "List shares on a TrueNAS system",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/truenas/assets/" + pathSegment(args[0]) + "/shares")
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

func init() {
	truenasCmd.AddCommand(truenasGetCmd, truenasPoolsCmd, truenasDatasetsCmd, truenasSharesCmd)
	rootCmd.AddCommand(truenasCmd)
}
