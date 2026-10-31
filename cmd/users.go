package cmd

import (
	"github.com/spf13/cobra"
)

var usersCmd = &cobra.Command{
	Use:   "users <asset>",
	Short: "Show local users on an asset",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/assets/" + pathSegment(args[0]) + "/users")
		if err != nil {
			return err
		}
		return printList(resp, "sessions",
			col("USER", "username"),
			col("TERMINAL", "terminal"),
			col("FROM", "remote_host"),
			colFormat("LOGIN", "login_time", timeText))
	},
}

func init() {
	rootCmd.AddCommand(usersCmd)
}
