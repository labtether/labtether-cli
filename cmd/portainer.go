package cmd

import (
	"encoding/json"
	"strings"

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
		resp, err := c.Get("/api/v2/portainer/assets/" + pathSegment(args[0]) + "/overview")
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
		resp, err := c.Get("/api/v2/portainer/assets/" + pathSegment(args[0]) + "/stacks")
		if err != nil {
			return err
		}
		return printList(resp, "data",
			col("ID", "Id"),
			col("NAME", "Name"),
			colFormat("STATUS", "Status", portainerStackStatus))
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
		resp, err := c.Get("/api/v2/portainer/assets/" + pathSegment(args[0]) + "/containers")
		if err != nil {
			return err
		}
		return printList(resp, "data",
			colFormat("ID", "Id", shortID),
			colFormat("NAME", "Names", containerNames),
			col("IMAGE", "Image"),
			col("STATE", "State"),
			col("STATUS", "Status"))
	},
}

func init() {
	portainerCmd.AddCommand(portainerGetCmd, portainerStacksCmd, portainerContainersCmd)
	rootCmd.AddCommand(portainerCmd)
}

// Portainer reports stack status as a number: 1 is active, 2 is inactive.
func portainerStackStatus(value any) string {
	switch text := cellText(value); text {
	case "1":
		return "active"
	case "2":
		return "inactive"
	default:
		return text
	}
}

// Docker lists container names with a leading slash, such as "/web".
func containerNames(value any) string {
	names, ok := value.([]any)
	if !ok {
		return cellText(value)
	}
	trimmed := make([]any, len(names))
	for i, name := range names {
		trimmed[i] = strings.TrimPrefix(cellText(name), "/")
	}
	return cellText(trimmed)
}
