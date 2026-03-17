package cmd

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

var topologyCmd = &cobra.Command{
	Use:   "topology",
	Short: "Explore asset dependency topology",
}

var topologyDependenciesCmd = &cobra.Command{
	Use:   "dependencies",
	Short: "Show all dependency edges",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/topology/dependencies")
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var topologyBlastRadiusCmd = &cobra.Command{
	Use:   "blast-radius <asset>",
	Short: "Show blast radius for an asset",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/topology/blast-radius/" + args[0])
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
		resp, err := c.Get("/api/v2/topology/upstream/" + args[0])
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var topologyEdgesCmd = &cobra.Command{
	Use:   "edges",
	Short: "List all topology edges",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/topology/edges")
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
