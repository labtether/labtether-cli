package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var assetsCmd = &cobra.Command{
	Use:   "assets",
	Short: "Manage assets (servers, VMs, containers)",
}

var assetsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all accessible assets",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}

		path := "/api/v2/assets"
		if online, _ := cmd.Flags().GetBool("online"); online {
			path += "?status=online"
		}

		resp, err := c.Get(path)
		if err != nil {
			return err
		}

		if jsonOutput {
			printJSON(json.RawMessage(resp.Data))
			return nil
		}

		assets, err := decodeRows(resp.Data)
		if err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
		printTable(assets, col("ID", "id"), col("PLATFORM", "platform"), col("STATUS", "status"),
			col("IP", "metadata.ip"), col("NAME", "name"))
		return nil
	},
}

var assetsGetCmd = &cobra.Command{
	Use:   "get <asset-id>",
	Short: "Get asset details",
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

func init() {
	assetsListCmd.Flags().Bool("online", false, "Show only online assets")
	assetsCmd.AddCommand(assetsListCmd, assetsGetCmd)
	rootCmd.AddCommand(assetsCmd)
}
