package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var packagesCmd = &cobra.Command{
	Use:   "packages",
	Short: "Install and update packages on a device",
}

var packagesListCmd = &cobra.Command{
	Use:   "list <device>",
	Short: "List installed packages on a device",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/assets/" + pathSegment(args[0]) + "/packages")
		if err != nil {
			return err
		}
		return printList(resp, "packages",
			col("NAME", "name"),
			col("VERSION", "version"),
			col("AVAILABLE", "available_version"),
			col("STATUS", "status"))
	},
}

var packagesInstallCmd = &cobra.Command{
	Use:   "install <device> <package>",
	Short: "Install a package on a device",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		c.HTTPClient.Timeout = hubLongActionTimeout
		resp, err := c.Post(fmt.Sprintf("/api/v2/assets/%s/packages/install", pathSegment(args[0])),
			map[string]string{"package": args[1]})
		if err != nil {
			return err
		}
		printMutationResult(resp, "Package %s install requested on %s\n", args[1], args[0])
		return nil
	},
}

var packagesUpdateCmd = &cobra.Command{
	Use:   "update <device>",
	Short: "Update all packages on a device",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		c.HTTPClient.Timeout = hubLongActionTimeout
		resp, err := c.Post(fmt.Sprintf("/api/v2/assets/%s/packages/upgrade", pathSegment(args[0])), map[string]any{})
		if err != nil {
			return err
		}
		printMutationResult(resp, "Package update requested on %s\n", args[0])
		return nil
	},
}

func init() {
	packagesCmd.AddCommand(packagesListCmd, packagesInstallCmd, packagesUpdateCmd)
	rootCmd.AddCommand(packagesCmd)
}
