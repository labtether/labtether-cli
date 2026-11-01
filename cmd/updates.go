package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var updatesCmd = &cobra.Command{
	Use:   "updates",
	Short: "Manage update plans and runs",
}

var updatesPlansCmd = &cobra.Command{
	Use:   "plans",
	Short: "Manage update plans",
}

var updatesPlansListCmd = &cobra.Command{
	Use:   "list",
	Short: "List update plans",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/updates/plans")
		if err != nil {
			return err
		}
		return printList(resp, "plans",
			col("ID", "id"),
			col("NAME", "name"),
			col("DRY RUN", "default_dry_run"),
			col("TARGETS", "targets"))
	},
}

var updatesPlansExecuteCmd = &cobra.Command{
	Use:   "execute <plan>",
	Short: "Execute an update plan",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Post(fmt.Sprintf("/api/v2/updates/plans/%s/execute", pathSegment(args[0])), nil)
		if err != nil {
			return err
		}
		printMutationResult(resp, "Update plan %s execution started\n", args[0])
		return nil
	},
}

var updatesRunsCmd = &cobra.Command{
	Use:   "runs",
	Short: "View update run history",
}

var updatesRunsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List update runs",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/updates/runs")
		if err != nil {
			return err
		}
		return printList(resp, "runs",
			col("ID", "id"),
			col("PLAN", "plan_name"),
			col("STATUS", "status"),
			col("DRY RUN", "dry_run"),
			colFormat("CREATED", "created_at", timeText))
	},
}

var updatesRunsGetCmd = &cobra.Command{
	Use:   "get <run>",
	Short: "Get details for an update run",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.Get("/api/v2/updates/runs/" + pathSegment(args[0]))
		if err != nil {
			return err
		}
		printJSON(json.RawMessage(resp.Data))
		return nil
	},
}

func init() {
	updatesPlansCmd.AddCommand(updatesPlansListCmd, updatesPlansExecuteCmd)
	updatesRunsCmd.AddCommand(updatesRunsListCmd, updatesRunsGetCmd)
	updatesCmd.AddCommand(updatesPlansCmd, updatesRunsCmd)
	rootCmd.AddCommand(updatesCmd)
}
