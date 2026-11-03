package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// devices was called assets before 2.0; the old name stays as a hidden alias.
var devicesCmd = &cobra.Command{
	Use:     "devices",
	Aliases: []string{"assets"},
	Short:   "List devices and show their details",
	Example: "  labtether-cli devices list\n" +
		"  labtether-cli devices list --online\n" +
		"  labtether-cli devices get nas01",
}

var devicesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the devices this API key can reach",
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

		devices, err := decodeRows(resp.Data)
		if err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
		printTable(devices, col("ID", "id"), col("PLATFORM", "platform"), colRow("STATUS", deviceStatus),
			col("ADDRESS", "metadata.ip"), col("NAME", "name"))
		return nil
	},
}

var devicesGetCmd = &cobra.Command{
	Use:   "get <device>",
	Short: "Show one device",
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
	devicesListCmd.Flags().Bool("online", false, "Show only online devices")
	devicesCmd.AddCommand(devicesListCmd, devicesGetCmd)
	rootCmd.AddCommand(devicesCmd)
}
