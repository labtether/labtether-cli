package cmd

import (
	"encoding/json"
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
		_, err = c.Post("/api/v2/discovery/run", nil)
		if err != nil {
			return err
		}
		fmt.Println("Discovery scan started")
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
		printJSON(json.RawMessage(resp.Data))
		return nil
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
		_, err = c.Post(fmt.Sprintf("/api/v2/discovery/proposals/%s/accept", args[0]), nil)
		if err != nil {
			return err
		}
		fmt.Printf("Discovery proposal %s accepted\n", args[0])
		return nil
	},
}

var discoveryRejectCmd = &cobra.Command{
	Use:   "reject <id>",
	Short: "Reject a discovery proposal",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		_, err = c.Post(fmt.Sprintf("/api/v2/discovery/proposals/%s/reject", args[0]), nil)
		if err != nil {
			return err
		}
		fmt.Printf("Discovery proposal %s rejected\n", args[0])
		return nil
	},
}

func init() {
	discoveryCmd.AddCommand(discoveryRunCmd, discoveryProposalsCmd, discoveryAcceptCmd, discoveryRejectCmd)
	rootCmd.AddCommand(discoveryCmd)
}
