package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var haCmd = &cobra.Command{
	Use:   "ha",
	Short: "Interact with Home Assistant",
}

var haEntitiesCmd = &cobra.Command{
	Use:   "entities",
	Short: "List all Home Assistant entities",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/homeassistant/entities")
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var haEntityCmd = &cobra.Command{
	Use:   "entity <id>",
	Short: "Get a Home Assistant entity by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/homeassistant/entities/" + args[0])
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var haCallCmd = &cobra.Command{
	Use:   "call <entity> <service>",
	Short: "Call a Home Assistant service on an entity",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		_, err = c.Post("/api/v2/homeassistant/call",
			map[string]string{"entity_id": args[0], "service": args[1]})
		if err != nil {
			return err
		}
		fmt.Printf("Called service %s on entity %s\n", args[1], args[0])
		return nil
	},
}

func init() {
	haCmd.AddCommand(haEntitiesCmd, haEntityCmd, haCallCmd)
	rootCmd.AddCommand(haCmd)
}
