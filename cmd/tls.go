package cmd

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

var tlsCmd = &cobra.Command{
	Use:   "tls",
	Short: "View TLS certificate status",
}

var tlsStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show TLS certificate status",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/tls/status")
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

func init() {
	tlsCmd.AddCommand(tlsStatusCmd)
	rootCmd.AddCommand(tlsCmd)
}
