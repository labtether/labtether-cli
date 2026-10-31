package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

const (
	minExecTimeoutSeconds = 1
	maxExecTimeoutSeconds = 300
)

var execCmd = &cobra.Command{
	Use:   "exec <asset> <command>",
	Short: "Run a command on an asset",
	Long:  "Execute a shell command on a managed asset. Use --targets for multi-asset execution.",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		targets, _ := cmd.Flags().GetString("targets")
		group, _ := cmd.Flags().GetString("group")
		timeout, _ := cmd.Flags().GetInt("timeout")

		if timeout < minExecTimeoutSeconds || timeout > maxExecTimeoutSeconds {
			return usageErrorf("timeout must be between %d and %d seconds", minExecTimeoutSeconds, maxExecTimeoutSeconds)
		}
		if targets == "" && group == "" && len(args) < 2 {
			return usageErrorf("an asset and a command are required, or --targets or --group with a command")
		}
		if (targets != "" || group != "") && len(args) < 1 {
			return usageErrorf("command is required")
		}

		c, err := newClient()
		if err != nil {
			return err
		}

		if targets != "" || group != "" {
			// Multi-target exec
			command := strings.Join(args, " ")
			body := map[string]any{
				"command": command,
				"timeout": timeout,
			}
			if targets != "" {
				body["targets"] = strings.Split(targets, ",")
			}
			if group != "" {
				body["group"] = group
			}

			resp, err := c.Post("/api/v2/exec", body)
			if err != nil {
				return err
			}

			var data multiExecResponse
			if err := json.Unmarshal(resp.Data, &data); err != nil {
				return fmt.Errorf("failed to parse multi-target response: %w", err)
			}
			if len(data.Results) == 0 {
				return fmt.Errorf("multi-target response did not contain any results")
			}
			if jsonOutput {
				printJSON(json.RawMessage(resp.Data))
			}

			targetNames := make([]string, 0, len(data.Results))
			for target := range data.Results {
				targetNames = append(targetNames, target)
			}
			sort.Strings(targetNames)
			for _, target := range targetNames {
				if !data.Results[target].known() {
					return fmt.Errorf("multi-target result for %s is missing status and exit_code", target)
				}
			}
			failed := 0
			for _, target := range targetNames {
				result := data.Results[target]
				if result.failed() {
					failed++
				}
				if jsonOutput {
					continue
				}
				switch {
				case result.Error != "":
					fmt.Fprintf(cmd.OutOrStdout(), "[%s] Error: %s\n", target, result.reason())
				case result.failed() && result.ExitCode != nil:
					fmt.Fprintf(cmd.OutOrStdout(), "[%s] Exit code %d: %s\n", target, *result.ExitCode, result.Stdout)
				case result.failed():
					fmt.Fprintf(cmd.OutOrStdout(), "[%s] Failed: %s\n", target, result.Stdout)
				default:
					fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s\n", target, result.Stdout)
				}
			}
			if failed > 0 {
				return remoteCommandError{
					message:  fmt.Sprintf("%d of %d remote commands failed", failed, len(data.Results)),
					exitCode: 1,
				}
			}
			return nil
		}

		// Single-target exec
		assetID := args[0]
		command := strings.Join(args[1:], " ")

		resp, err := c.Post(fmt.Sprintf("/api/v2/assets/%s/exec", pathSegment(assetID)), map[string]any{
			"command": command,
			"timeout": timeout,
		})
		if err != nil {
			return err
		}

		var data remoteExecResult
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
		if !data.known() {
			return fmt.Errorf("remote command response is missing status and exit_code")
		}
		if jsonOutput {
			printJSON(json.RawMessage(resp.Data))
		} else if data.Stdout != "" {
			fmt.Fprintln(cmd.OutOrStdout(), data.Stdout)
		}
		if data.failed() {
			return data.failure()
		}
		return nil
	},
}

func init() {
	execCmd.Flags().String("targets", "", "Comma-separated list of asset IDs for multi-target exec")
	execCmd.Flags().String("group", "", "Group name for multi-target exec")
	execCmd.Flags().Var(
		newBoundedIntValue(30, "timeout", "seconds", minExecTimeoutSeconds, maxExecTimeoutSeconds),
		"timeout",
		"Command timeout in seconds (max 300)",
	)
	rootCmd.AddCommand(execCmd)
}
