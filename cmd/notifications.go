package cmd

import (
	"github.com/spf13/cobra"
)

var notificationsCmd = &cobra.Command{
	Use:   "notifications",
	Short: "View notification channels and history",
}

var notificationsChannelsCmd = &cobra.Command{
	Use:   "channels",
	Short: "List notification channels",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/notifications/channels")
		if err != nil {
			return err
		}
		return printList(resp, "channels",
			col("ID", "id"),
			col("NAME", "name"),
			col("TYPE", "type"),
			col("ENABLED", "enabled"))
	},
}

var notificationsHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "View notification history",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/notifications/history")
		if err != nil {
			return err
		}
		return printList(resp, "history",
			col("ID", "id"),
			col("CHANNEL", "channel_id"),
			col("STATUS", "status"),
			colFormat("SENT", "sent_at", timeText),
			col("RETRIES", "retry_count"),
			col("ERROR", "error"))
	},
}

func init() {
	notificationsCmd.AddCommand(notificationsChannelsCmd, notificationsHistoryCmd)
	rootCmd.AddCommand(notificationsCmd)
}
