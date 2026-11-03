package cmd

import (
	"strconv"

	"github.com/spf13/cobra"
)

var psCmd = &cobra.Command{
	Use:   "ps",
	Short: "List and signal processes on a device",
}

var psListCmd = &cobra.Command{
	Use:   "list <device>",
	Short: "List running processes on a device",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}

		resp, err := c.Get("/api/v2/assets/" + pathSegment(args[0]) + "/processes")
		if err != nil {
			return err
		}

		return printList(resp, "processes",
			col("PID", "pid"), col("USER", "user"), col("CPU%", "cpu_pct"),
			col("MEM%", "mem_pct"), col("COMMAND", "command"))
	},
}

var psKillCmd = &cobra.Command{
	Use:   "kill <device> <pid>",
	Short: "Send a signal to a process on a device",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, err := strconv.Atoi(args[1])
		if err != nil || pid <= 0 {
			return usageErrorf("pid must be a positive integer")
		}
		c, err := newClient()
		if err != nil {
			return err
		}

		signal, _ := cmd.Flags().GetString("signal")
		body := map[string]any{
			"pid":    pid,
			"signal": signal,
		}

		resp, err := c.Post("/api/v2/assets/"+pathSegment(args[0])+"/processes/kill", body)
		if err != nil {
			return err
		}

		printMutationResult(resp, "Signal %s sent to PID %s on %s\n", signal, args[1], args[0])
		return nil
	},
}

func init() {
	psKillCmd.Flags().String("signal", "SIGTERM", "Signal to send, such as SIGTERM or SIGKILL")
	psCmd.AddCommand(psListCmd, psKillCmd)
	rootCmd.AddCommand(psCmd)
}
