package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var webServicesCmd = &cobra.Command{
	Use:   "web-services",
	Short: "Manage discovered web services",
}

var webServicesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all discovered web services",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/services/web")
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var webServicesSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Trigger a web services sync",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		_, err = c.Post("/api/v2/services/web/sync", nil)
		if err != nil {
			return err
		}
		fmt.Println("Web services sync started")
		return nil
	},
}

func init() {
	webServicesCmd.AddCommand(webServicesListCmd, webServicesSyncCmd)
	rootCmd.AddCommand(webServicesCmd)
}
