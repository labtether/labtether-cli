package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

const (
	minExecTimeoutSeconds = 1
	maxExecTimeoutSeconds = 300
)

type remoteExecResult struct {
	ExitCode *int   `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Error    string `json:"error,omitempty"`
	Message  string `json:"message,omitempty"`
}

type multiExecResponse struct {
	Results map[string]remoteExecResult `json:"results"`
}

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
			return fmt.Errorf("timeout must be between %d and %d seconds", minExecTimeoutSeconds, maxExecTimeoutSeconds)
		}
		if targets == "" && group == "" && len(args) < 2 {
			return fmt.Errorf("usage: labtether-cli exec <asset> <command>\n  or:  labtether-cli exec --targets a,b,c <command>\n  or:  labtether-cli exec --group <name> <command>")
		}
		if (targets != "" || group != "") && len(args) < 1 {
			return fmt.Errorf("command is required")
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
				if data.Results[target].ExitCode == nil {
					return fmt.Errorf("multi-target result for %s is missing exit_code", target)
				}
			}
			failed := 0
			for _, target := range targetNames {
				result := data.Results[target]
				exitCode := *result.ExitCode
				if result.Error != "" || exitCode != 0 {
					failed++
				}
				if jsonOutput {
					continue
				}
				if result.Error != "" {
					message := strings.TrimSpace(result.Message)
					if message == "" {
						message = result.Error
					}
					fmt.Printf("[%s] Error: %s\n", target, message)
				} else if exitCode != 0 {
					fmt.Printf("[%s] Exit code %d: %s\n", target, exitCode, result.Stdout)
				} else {
					fmt.Printf("[%s] %s\n", target, result.Stdout)
				}
			}
			if failed > 0 {
				return fmt.Errorf("%d of %d remote commands failed", failed, len(data.Results))
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
		if data.ExitCode == nil {
			return fmt.Errorf("remote command response is missing exit_code")
		}
		if jsonOutput {
			printJSON(json.RawMessage(resp.Data))
		} else if data.Stdout != "" {
			fmt.Println(data.Stdout)
		}
		if data.Error != "" {
			message := strings.TrimSpace(data.Message)
			if message == "" {
				message = data.Error
			}
			return fmt.Errorf("remote command failed: %s", message)
		}
		if *data.ExitCode != 0 {
			if !jsonOutput {
				fmt.Fprintf(os.Stderr, "Exit code: %d\n", *data.ExitCode)
			}
			return fmt.Errorf("remote command failed with exit code %d", *data.ExitCode)
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
