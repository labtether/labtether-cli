package cmd

import (
	"github.com/spf13/cobra"
)

var cronCmd = &cobra.Command{
	Use:   "cron <asset>",
	Short: "Show cron jobs on an asset",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/assets/" + pathSegment(args[0]) + "/cron")
		if err != nil {
			return err
		}
		return printList(resp, "entries",
			col("SOURCE", "source"),
			col("SCHEDULE", "schedule"),
			col("USER", "user"),
			col("COMMAND", "command"))
	},
}

func init() {
	rootCmd.AddCommand(cronCmd)
}
