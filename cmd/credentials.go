package cmd

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

var credentialsCmd = &cobra.Command{
	Use:   "credentials",
	Short: "View stored credentials",
}

var credentialsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List credentials (metadata only)",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/credentials")
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var credentialsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get credential metadata",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/credentials/" + args[0])
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
