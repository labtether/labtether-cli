package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var psCmd = &cobra.Command{
	Use:   "ps",
	Short: "Manage processes on assets",
}

var psListCmd = &cobra.Command{
	Use:   "list <asset>",
	Short: "List running processes on an asset",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}

		resp, err := c.Get("/api/v2/assets/" + args[0] + "/processes")
		if err != nil {
			return err
		}

		if jsonOutput {
			printJSON(json.RawMessage(resp.Data))
			return nil
		}

		var processes []map[string]any
		json.Unmarshal(resp.Data, &processes)

		fmt.Printf("%-8s %-8s %-6s %-6s %s\n", "PID", "USER", "CPU%", "MEM%", "COMMAND")
		for _, p := range processes {
			fmt.Printf("%-8v %-8v %-6v %-6v %v\n",
				p["pid"], p["user"], p["cpu_percent"], p["mem_percent"], p["command"])
		}
		return nil
	},
}

var psKillCmd = &cobra.Command{
	Use:   "kill <asset> <pid>",
	Short: "Kill a process on an asset",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}

		signal, _ := cmd.Flags().GetString("signal")
		body := map[string]any{
			"pid":    args[1],
			"signal": signal,
		}

		_, err = c.Post("/api/v2/assets/"+args[0]+"/processes/kill", body)
		if err != nil {
			return err
		}

		fmt.Printf("Signal %s sent to PID %s on %s\n", signal, args[1], args[0])
		return nil
	},
}

func init() {
	psKillCmd.Flags().String("signal", "SIGTERM", "Signal to send (SIGTERM, SIGKILL, etc.)")
	psCmd.AddCommand(psListCmd, psKillCmd)
	rootCmd.AddCommand(psCmd)
}
