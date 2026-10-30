package cmd

import (
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
		resp, err := c.Get("/api/v2/assets/" + pathSegment(args[0]) + "/services")
		if err != nil {
			return err
		}
		return printList(resp, "services",
			col("NAME", "name"), col("ACTIVE", "active_state"), col("SUB", "sub_state"),
			col("ENABLED", "enabled"), col("DESCRIPTION", "description"))
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
		resp, err := c.Post(fmt.Sprintf("/api/v2/assets/%s/services/%s/start", pathSegment(args[0]), pathSegment(args[1])), nil)
		if err != nil {
			return err
		}
		printMutationResult(resp, "Service %s started on %s\n", args[1], args[0])
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
		resp, err := c.Post(fmt.Sprintf("/api/v2/assets/%s/services/%s/stop", pathSegment(args[0]), pathSegment(args[1])), nil)
		if err != nil {
			return err
		}
		printMutationResult(resp, "Service %s stopped on %s\n", args[1], args[0])
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
		resp, err := c.Post(fmt.Sprintf("/api/v2/assets/%s/services/%s/restart", pathSegment(args[0]), pathSegment(args[1])), nil)
		if err != nil {
			return err
		}
		printMutationResult(resp, "Service %s restarted on %s\n", args[1], args[0])
		return nil
	},
}

func init() {
	servicesCmd.AddCommand(servicesListCmd, servicesStartCmd, servicesStopCmd, servicesRestartCmd)
	rootCmd.AddCommand(servicesCmd)
}
