package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var packagesCmd = &cobra.Command{
	Use:   "packages",
	Short: "Manage packages on assets",
}

var packagesListCmd = &cobra.Command{
	Use:   "list <asset>",
	Short: "List installed packages on an asset",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/assets/" + args[0] + "/packages")
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var packagesInstallCmd = &cobra.Command{
	Use:   "install <asset> <pkg>",
	Short: "Install a package on an asset",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		_, err = c.Post(fmt.Sprintf("/api/v2/assets/%s/packages/install", args[0]),
			map[string]string{"package": args[1]})
		if err != nil {
			return err
		}
		fmt.Printf("Package %s install requested on %s\n", args[1], args[0])
		return nil
	},
}

var packagesUpdateCmd = &cobra.Command{
	Use:   "update <asset>",
	Short: "Update all packages on an asset",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		_, err = c.Post(fmt.Sprintf("/api/v2/assets/%s/packages/update", args[0]), nil)
		if err != nil {
			return err
		}
		fmt.Printf("Package update requested on %s\n", args[0])
		return nil
	},
}

func init() {
	packagesCmd.AddCommand(packagesListCmd, packagesInstallCmd, packagesUpdateCmd)
	rootCmd.AddCommand(packagesCmd)
}
