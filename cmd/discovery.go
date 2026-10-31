package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var discoveryCmd = &cobra.Command{
	Use:   "discovery",
	Short: "Manage asset discovery",
}

var discoveryRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Trigger a discovery scan",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Post("/api/v2/discovery/run", nil)
		if err != nil {
			return err
		}
		printMutationResult(resp, "Discovery scan started\n")
		return nil
	},
}

var discoveryProposalsCmd = &cobra.Command{
	Use:   "proposals",
	Short: "List pending discovery proposals",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/discovery/proposals")
		if err != nil {
			return err
		}
		return printList(resp, "proposals",
			col("ID", "id"),
			col("SOURCE", "source_asset_id"),
			col("TARGET", "target_asset_id"),
			col("RELATIONSHIP", "relationship_type"),
			col("CONFIDENCE", "confidence"))
	},
}

var discoveryAcceptCmd = &cobra.Command{
	Use:   "accept <id>",
	Short: "Accept a discovery proposal",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Post(fmt.Sprintf("/api/v2/discovery/proposals/%s/accept", pathSegment(args[0])), nil)
		if err != nil {
			return err
		}
		printMutationResult(resp, "Discovery proposal %s accepted\n", args[0])
		return nil
	},
}

var discoveryRejectCmd = &cobra.Command{
	Use:     "dismiss <id>",
	Aliases: []string{"reject"},
	Short:   "Dismiss a discovery proposal",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Post(fmt.Sprintf("/api/v2/discovery/proposals/%s/dismiss", pathSegment(args[0])), nil)
		if err != nil {
			return err
		}
		printMutationResult(resp, "Discovery proposal %s dismissed\n", args[0])
		return nil
	},
}

func init() {
	discoveryCmd.AddCommand(discoveryRunCmd, discoveryProposalsCmd, discoveryAcceptCmd, discoveryRejectCmd)
	rootCmd.AddCommand(discoveryCmd)
}
