package cmd

import (
	"github.com/spf13/cobra"
)

var netCmd = &cobra.Command{
	Use:   "net <device>",
	Short: "Show network interfaces on a device",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/assets/" + pathSegment(args[0]) + "/network")
		if err != nil {
			return err
		}
		return printList(resp, "interfaces",
			col("NAME", "name"),
			col("STATE", "state"),
			col("MAC", "mac"),
			col("MTU", "mtu"),
			col("ADDRESSES", "ips"))
	},
}

func init() {
	rootCmd.AddCommand(netCmd)
}
