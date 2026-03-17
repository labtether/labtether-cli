package cmd

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

var portainerCmd = &cobra.Command{
	Use:   "portainer",
	Short: "Interact with Portainer environments",
}

var portainerGetCmd = &cobra.Command{
	Use:   "get <env>",
	Short: "Get Portainer environment details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/portainer/environments/" + args[0])
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var portainerStacksCmd = &cobra.Command{
	Use:   "stacks <env>",
	Short: "List stacks in a Portainer environment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/portainer/environments/" + args[0] + "/stacks")
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var portainerContainersCmd = &cobra.Command{
	Use:   "containers <env>",
	Short: "List containers in a Portainer environment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/portainer/environments/" + args[0] + "/containers")
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

func init() {
	portainerCmd.AddCommand(portainerGetCmd, portainerStacksCmd, portainerContainersCmd)
	rootCmd.AddCommand(portainerCmd)
}
