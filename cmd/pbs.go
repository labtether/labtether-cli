package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/labtether/labtether-cli/internal/client"
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
		resp, err := c.Get("/api/v2/pbs/assets/" + pathSegment(args[0]) + "/details")
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
		// The Hub has no datastore list route; its details payload has the list.
		resp, err := c.Get("/api/v2/pbs/assets/" + pathSegment(args[0]) + "/details")
		if err != nil {
			return err
		}
		datastores, err := pbsDatastoreList(resp)
		if err != nil {
			return err
		}
		if jsonOutput {
			printJSON(datastores)
			return nil
		}
		rows, err := decodeRows(datastores)
		if err != nil {
			return fmt.Errorf("decode response data: datastores: %w", err)
		}
		printTable(rows, col("STORE", "store"), col("STATUS", "status"),
			colFormat("SIZE", "total_bytes", byteSize), colFormat("USED", "used_bytes", byteSize),
			colFormat("AVAIL", "avail_bytes", byteSize), col("USE%", "usage_percent"),
			col("GROUPS", "group_count"), col("SNAPSHOTS", "snapshot_count"),
			colFormat("LAST BACKUP", "last_backup_at", timeText))
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
		resp, err := c.Get("/api/v2/pbs/assets/" + pathSegment(args[0]) + "/snapshots")
		if err != nil {
			return err
		}
		return printList(resp, "snapshots",
			col("TYPE", "backup_type"),
			col("ID", "backup_id"),
			colFormat("TIME", "backup_time", unixTimeText),
			colFormat("SIZE", "size", byteSize))
	},
}

func init() {
	pbsCmd.AddCommand(pbsGetCmd, pbsDatastoresCmd, pbsSnapshotsCmd)
	rootCmd.AddCommand(pbsCmd)
}

// pbsDatastoreList returns the datastores in a PBS details payload as a JSON
// array. A server asset lists them in "datastores", which the Hub leaves out
// when there are none; a datastore asset has its one summary in "datastore".
func pbsDatastoreList(resp *client.V2Response) (json.RawMessage, error) {
	var details struct {
		Datastore  json.RawMessage `json:"datastore"`
		Datastores json.RawMessage `json:"datastores"`
	}
	if err := decodeResponseData(resp, &details); err != nil {
		return nil, err
	}
	switch {
	case len(details.Datastores) > 0 && string(details.Datastores) != "null":
		return details.Datastores, nil
	case len(details.Datastore) > 0 && string(details.Datastore) != "null":
		return json.RawMessage("[" + string(details.Datastore) + "]"), nil
	default:
		return json.RawMessage("[]"), nil
	}
}
