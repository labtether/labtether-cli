package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var servicesCmd = &cobra.Command{
	Use:   "services",
	Short: "Manage system services on assets",
}

var servicesListCmd = &cobra.Command{
	Use:   "list <asset>",
	Short: "List services on an asset",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/assets/" + args[0] + "/services")
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var servicesStartCmd = &cobra.Command{
	Use:   "start <asset> <service>",
	Short: "Start a service",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		_, err = c.Post(fmt.Sprintf("/api/v2/assets/%s/services/%s/start", args[0], args[1]), nil)
		if err != nil {
			return err
		}
		fmt.Printf("Service %s started on %s\n", args[1], args[0])
		return nil
	},
}

var servicesStopCmd = &cobra.Command{
	Use:   "stop <asset> <service>",
	Short: "Stop a service",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		_, err = c.Post(fmt.Sprintf("/api/v2/assets/%s/services/%s/stop", args[0], args[1]), nil)
		if err != nil {
			return err
		}
		fmt.Printf("Service %s stopped on %s\n", args[1], args[0])
		return nil
	},
}

var servicesRestartCmd = &cobra.Command{
	Use:   "restart <asset> <service>",
	Short: "Restart a service",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		_, err = c.Post(fmt.Sprintf("/api/v2/assets/%s/services/%s/restart", args[0], args[1]), nil)
		if err != nil {
			return err
		}
		fmt.Printf("Service %s restarted on %s\n", args[1], args[0])
		return nil
	},
}

func init() {
	servicesCmd.AddCommand(servicesListCmd, servicesStartCmd, servicesStopCmd, servicesRestartCmd)
	rootCmd.AddCommand(servicesCmd)
}
