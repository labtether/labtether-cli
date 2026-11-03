package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/labtether/labtether-cli/internal/client"
)

var alertsCmd = &cobra.Command{
	Use:   "alerts",
	Short: "List and resolve alerts",
}

var alertsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List active alerts",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/alerts")
		if err != nil {
			return err
		}
		if jsonOutput {
			printJSON(json.RawMessage(resp.Data))
			return nil
		}
		rows, err := decodeTableRows(resp, "instances")
		if err != nil {
			return err
		}
		names := alertRuleNames(c)
		printTable(rows,
			col("ID", "id"),
			colFormat("SEVERITY", "severity", statusWord),
			colFormat("STATE", "status", statusWord),
			colFormat("STARTED", "started_at", timeText),
			colRow("RULE", func(row map[string]any) string {
				id := cellText(lookupPath(row, "rule_id"))
				if name := names[id]; name != "" {
					return name
				}
				return id
			}),
			col("DEVICE", "firing_scope.asset_ids"))
		return nil
	},
}

// alertRuleNames maps rule IDs to their names, so the alert list says what
// fired. Without access to the rules it returns no names and the list shows
// rule IDs.
func alertRuleNames(c *client.Client) map[string]string {
	names := map[string]string{}
	resp, err := c.Get("/api/v2/alerts/rules")
	if err != nil {
		return names
	}
	rows, err := decodeTableRows(resp, "rules")
	if err != nil {
		return names
	}
	for _, row := range rows {
		if id, ok := row["id"].(string); ok {
			if name, ok := row["name"].(string); ok {
				names[id] = strings.TrimSpace(name)
			}
		}
	}
	return names
}

var alertsAckCmd = &cobra.Command{
	Use:   "ack <alert>",
	Short: "Acknowledge an alert",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Post(fmt.Sprintf("/api/v2/alerts/%s/ack", pathSegment(args[0])), nil)
		if err != nil {
			return err
		}
		printMutationResult(resp, "Alert %s acknowledged\n", args[0])
		return nil
	},
}

var alertsResolveCmd = &cobra.Command{
	Use:   "resolve <alert>",
	Short: "Resolve an alert",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Post(fmt.Sprintf("/api/v2/alerts/%s/resolve", pathSegment(args[0])), nil)
		if err != nil {
			return err
		}
		printMutationResult(resp, "Alert %s resolved\n", args[0])
		return nil
	},
}

func init() {
	alertsCmd.AddCommand(alertsListCmd, alertsAckCmd, alertsResolveCmd)
	rootCmd.AddCommand(alertsCmd)
}
