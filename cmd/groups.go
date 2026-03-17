package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var groupsCmd = &cobra.Command{
	Use:   "groups",
	Short: "Manage asset groups",
}

var groupsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all groups",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/groups")
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var groupsGetCmd = &cobra.Command{
	Use:   "get <group>",
	Short: "Get group details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/groups/" + args[0])
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var groupsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new group",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			return fmt.Errorf("--name is required")
		}
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Post("/api/v2/groups", map[string]string{"name": name})
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

func init() {
	groupsCreateCmd.Flags().String("name", "", "Group name (required)")
	groupsCmd.AddCommand(groupsListCmd, groupsGetCmd, groupsCreateCmd)
	rootCmd.AddCommand(groupsCmd)
}
