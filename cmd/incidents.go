package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var incidentsCmd = &cobra.Command{
	Use:   "incidents",
	Short: "Manage incidents",
}

var incidentsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List incidents",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/incidents")
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var incidentsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get incident details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/incidents/" + args[0])
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var incidentsAckCmd = &cobra.Command{
	Use:   "ack <id>",
	Short: "Acknowledge an incident",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		_, err = c.Post(fmt.Sprintf("/api/v2/incidents/%s/ack", args[0]), nil)
		if err != nil {
			return err
		}
		fmt.Printf("Incident %s acknowledged\n", args[0])
		return nil
	},
}

var incidentsResolveCmd = &cobra.Command{
	Use:   "resolve <id>",
	Short: "Resolve an incident",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		_, err = c.Post(fmt.Sprintf("/api/v2/incidents/%s/resolve", args[0]), nil)
		if err != nil {
			return err
		}
		fmt.Printf("Incident %s resolved\n", args[0])
		return nil
	},
}

func init() {
	incidentsCmd.AddCommand(incidentsListCmd, incidentsGetCmd, incidentsAckCmd, incidentsResolveCmd)
	rootCmd.AddCommand(incidentsCmd)
}
