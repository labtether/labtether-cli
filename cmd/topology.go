package cmd

import (
	"encoding/json"
	"net/url"

	"github.com/spf13/cobra"
)

var topologyCmd = &cobra.Command{
	Use:   "topology",
	Short: "Explore asset dependency topology",
}

var topologyDependenciesCmd = &cobra.Command{
	Use:   "dependencies <asset>",
	Short: "Show dependencies for an asset",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/dependencies?asset_id=" + url.QueryEscape(args[0]))
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var topologyBlastRadiusCmd = &cobra.Command{
	Use:   "blast-radius <asset>",
	Short: "Show downstream edges for an asset",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/edges/tree?root=" + url.QueryEscape(args[0]))
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var topologyUpstreamCmd = &cobra.Command{
	Use:   "upstream <asset>",
	Short: "Show upstream dependencies for an asset",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/edges/ancestors?id=" + url.QueryEscape(args[0]))
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var topologyEdgesCmd = &cobra.Command{
	Use:   "edges <asset>",
	Short: "List topology edges for an asset",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/edges?asset_id=" + url.QueryEscape(args[0]))
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

func init() {
	topologyCmd.AddCommand(topologyDependenciesCmd, topologyBlastRadiusCmd, topologyUpstreamCmd, topologyEdgesCmd)
	rootCmd.AddCommand(topologyCmd)
}
