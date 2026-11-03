package cmd

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

var credentialsCmd = &cobra.Command{
	Use:   "credentials",
	Short: "Show saved credentials without their secrets",
}

var credentialsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List saved credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/credentials/profiles")
		if err != nil {
			return err
		}
		return printList(resp, "profiles",
			col("ID", "id"),
			col("NAME", "name"),
			col("KIND", "kind"),
			col("USERNAME", "username"),
			colFormat("STATUS", "status", statusWord))
	},
}

var credentialsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Show one credential",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/credentials/profiles/" + pathSegment(args[0]))
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

func init() {
	credentialsCmd.AddCommand(credentialsListCmd, credentialsGetCmd)
	rootCmd.AddCommand(credentialsCmd)
}
