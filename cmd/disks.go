package cmd

import (
	"github.com/spf13/cobra"
)

var disksCmd = &cobra.Command{
	Use:   "disks <asset>",
	Short: "Show disk info for an asset",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/assets/" + pathSegment(args[0]) + "/disks")
		if err != nil {
			return err
		}
		return printList(resp, "mounts",
			col("DEVICE", "device"),
			col("MOUNT", "mount_point"),
			col("TYPE", "fs_type"),
			colFormat("SIZE", "total", byteSize),
			colFormat("USED", "used", byteSize),
			colFormat("AVAIL", "available", byteSize),
			col("USE%", "use_pct"))
	},
}

func init() {
	rootCmd.AddCommand(disksCmd)
}
