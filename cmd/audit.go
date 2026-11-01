package cmd

import (
	"github.com/spf13/cobra"
)

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "View audit event log",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/audit/events")
		if err != nil {
			return err
		}
		return printList(resp, "events",
			colFormat("TIME", "timestamp", timeText),
			col("TYPE", "type"),
			col("ACTOR", "actor_id"),
			col("TARGET", "target"),
			col("DECISION", "decision"))
	},
}

func init() {
	rootCmd.AddCommand(auditCmd)
}
