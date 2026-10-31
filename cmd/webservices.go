package cmd

import (
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
		resp, err := c.Get("/api/v2/web-services")
		if err != nil {
			return err
		}
		return printList(resp, "services",
			col("NAME", "name"), col("CATEGORY", "category"), col("STATUS", "status"),
			col("HOST", "host_asset_id"), col("URL", "url"))
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
		resp, err := c.Post("/api/v2/web-services/sync", nil)
		if err != nil {
			return err
		}
		printMutationResult(resp, "Web services sync started\n")
		return nil
	},
}

func init() {
	webServicesCmd.AddCommand(webServicesListCmd, webServicesSyncCmd)
	rootCmd.AddCommand(webServicesCmd)
}
