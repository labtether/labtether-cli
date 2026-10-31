package cmd

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

var agentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "Manage labtether-agent installations",
}

var agentsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List connected agents",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/agents")
		if err != nil {
			return err
		}
		if jsonOutput {
			printJSON(json.RawMessage(resp.Data))
			return nil
		}
		agents, err := decodeTableRows(resp, "assetsInfo")
		if err != nil {
			return err
		}
		sortRows(agents, "id")
		printTable(agents, col("ID", "id"), col("PLATFORM", "platform"))
		return nil
	},
}

var agentsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get an agent asset's details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/assets/" + pathSegment(args[0]))
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
		return printList(resp, "agents",
			col("ID", "asset_id"), col("HOSTNAME", "hostname"), col("PLATFORM", "platform"),
			col("IP", "remote_ip"), colFormat("CONNECTED", "connected_at", timeText),
			col("VERIFIED", "identity_verified"))
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
		resp, err := c.Post("/api/v2/agents/pending/approve", map[string]string{"asset_id": args[0]})
		if err != nil {
			return err
		}
		printMutationResult(resp, "Agent %s approved\n", args[0])
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
		resp, err := c.Post("/api/v2/agents/pending/reject", map[string]string{"asset_id": args[0]})
		if err != nil {
			return err
		}
		printMutationResult(resp, "Agent %s rejected\n", args[0])
		return nil
	},
}

func init() {
	agentsCmd.AddCommand(agentsListCmd, agentsGetCmd, agentsPendingCmd, agentsApproveCmd, agentsRejectCmd)
	rootCmd.AddCommand(agentsCmd)
}
