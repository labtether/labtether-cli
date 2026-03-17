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
		printJSON(json.RawMessage(resp.Data))
		return nil
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
		_, err = c.Post(fmt.Sprintf("/api/v2/updates/plans/%s/execute", args[0]), nil)
		if err != nil {
			return err
		}
		fmt.Printf("Update plan %s execution started\n", args[0])
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
		printJSON(json.RawMessage(resp.Data))
		return nil
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
		resp, err := c.Get("/api/v2/updates/runs/" + args[0])
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
