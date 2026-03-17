package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var agentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "Manage labtether-agent installations",
}

var agentsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered agents",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/agents")
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var agentsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get agent details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/agents/" + args[0])
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var agentsPendingCmd = &cobra.Command{
	Use:   "pending",
	Short: "List agents awaiting approval",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/agents/pending")
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var agentsApproveCmd = &cobra.Command{
	Use:   "approve <id>",
	Short: "Approve a pending agent",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		_, err = c.Post(fmt.Sprintf("/api/v2/agents/%s/approve", args[0]), nil)
		if err != nil {
			return err
		}
		fmt.Printf("Agent %s approved\n", args[0])
		return nil
	},
}

var agentsRejectCmd = &cobra.Command{
	Use:   "reject <id>",
	Short: "Reject a pending agent",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		_, err = c.Post(fmt.Sprintf("/api/v2/agents/%s/reject", args[0]), nil)
		if err != nil {
			return err
		}
		fmt.Printf("Agent %s rejected\n", args[0])
		return nil
	},
}

func init() {
	agentsCmd.AddCommand(agentsListCmd, agentsGetCmd, agentsPendingCmd, agentsApproveCmd, agentsRejectCmd)
	rootCmd.AddCommand(agentsCmd)
}
