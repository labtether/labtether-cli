package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rebootCmd = &cobra.Command{
	Use:   "reboot <device>",
	Short: "Reboot a device",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Post(fmt.Sprintf("/api/v2/assets/%s/reboot", pathSegment(args[0])), nil)
		if err != nil {
			return err
		}
		printMutationResult(resp, "Reboot requested for %s\n", args[0])
		return nil
	},
}

var shutdownCmd = &cobra.Command{
	Use:   "shutdown <device>",
	Short: "Shut down a device",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Post(fmt.Sprintf("/api/v2/assets/%s/shutdown", pathSegment(args[0])), nil)
		if err != nil {
			return err
		}
		printMutationResult(resp, "Shutdown requested for %s\n", args[0])
		return nil
	},
}

var wakeCmd = &cobra.Command{
	Use:   "wake <device>",
	Short: "Wake a device with Wake-on-LAN",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Post(fmt.Sprintf("/api/v2/assets/%s/wake", pathSegment(args[0])), nil)
		if err != nil {
			return err
		}
		printMutationResult(resp, "Wake-on-LAN sent to %s\n", args[0])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(rebootCmd, shutdownCmd, wakeCmd)
}
