package cmd

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

var pbsCmd = &cobra.Command{
	Use:   "pbs",
	Short: "Interact with Proxmox Backup Server",
}

var pbsGetCmd = &cobra.Command{
	Use:   "get <srv>",
	Short: "Get PBS server details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/pbs/" + args[0])
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var pbsDatastoresCmd = &cobra.Command{
	Use:   "datastores <srv>",
	Short: "List datastores on a PBS server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/pbs/" + args[0] + "/datastores")
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var pbsSnapshotsCmd = &cobra.Command{
	Use:   "snapshots <srv>",
	Short: "List snapshots on a PBS server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/pbs/" + args[0] + "/snapshots")
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

func init() {
	pbsCmd.AddCommand(pbsGetCmd, pbsDatastoresCmd, pbsSnapshotsCmd)
	rootCmd.AddCommand(pbsCmd)
}
