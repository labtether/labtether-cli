package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// issues was called incidents before 2.0; the old name stays as a hidden alias.
var issuesCmd = &cobra.Command{
	Use:     "issues",
	Aliases: []string{"incidents"},
	Short:   "List and resolve issues",
	Example: "  labtether-cli issues list\n" +
		"  labtether-cli issues investigate <id>\n" +
		"  labtether-cli issues resolve <id>",
}

var issuesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List issues",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/incidents")
		if err != nil {
			return err
		}
		return printList(resp, "incidents",
			col("ID", "id"),
			colFormat("SEVERITY", "severity", statusWord),
			colFormat("STATE", "status", statusWord),
			colFormat("OPENED", "opened_at", timeText),
			col("TITLE", "title"))
	},
}

var issuesGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Show one issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/incidents/" + pathSegment(args[0]))
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

var issuesInvestigateCmd = &cobra.Command{
	Use:     "investigate <id>",
	Aliases: []string{"ack"},
	Short:   "Move an issue to Investigating",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Patch(fmt.Sprintf("/api/v2/incidents/%s", pathSegment(args[0])), map[string]string{"status": "investigating"})
		if err != nil {
			return err
		}
		printMutationResult(resp, "Issue %s moved to Investigating\n", args[0])
		return nil
	},
}

var issuesResolveCmd = &cobra.Command{
	Use:   "resolve <id>",
	Short: "Resolve an issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Patch(fmt.Sprintf("/api/v2/incidents/%s", pathSegment(args[0])), map[string]string{"status": "resolved"})
		if err != nil {
			return err
		}
		printMutationResult(resp, "Issue %s resolved\n", args[0])
		return nil
	},
}

func init() {
	issuesCmd.AddCommand(issuesListCmd, issuesGetCmd, issuesInvestigateCmd, issuesResolveCmd)
	rootCmd.AddCommand(issuesCmd)
}
