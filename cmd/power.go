package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rebootCmd = &cobra.Command{
	Use:   "reboot <asset>",
	Short: "Reboot an asset",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		_, err = c.Post(fmt.Sprintf("/api/v2/assets/%s/reboot", args[0]), nil)
		if err != nil {
			return err
		}
		fmt.Printf("Reboot requested for %s\n", args[0])
		return nil
	},
}

var shutdownCmd = &cobra.Command{
	Use:   "shutdown <asset>",
	Short: "Shut down an asset",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		_, err = c.Post(fmt.Sprintf("/api/v2/assets/%s/shutdown", args[0]), nil)
		if err != nil {
			return err
		}
		fmt.Printf("Shutdown requested for %s\n", args[0])
		return nil
	},
}

var wakeCmd = &cobra.Command{
	Use:   "wake <asset>",
	Short: "Wake an asset via Wake-on-LAN",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		_, err = c.Post(fmt.Sprintf("/api/v2/assets/%s/wake", args[0]), nil)
		if err != nil {
			return err
		}
		fmt.Printf("Wake-on-LAN sent to %s\n", args[0])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(rebootCmd, shutdownCmd, wakeCmd)
}
