package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
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

			if jsonOutput {
				printJSON(json.RawMessage(resp.Data))
				return nil
			}

			var data map[string]any
			if err := json.Unmarshal(resp.Data, &data); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			results, _ := data["results"].(map[string]any)
			for target, res := range results {
				result, ok := res.(map[string]any)
				if !ok {
					fmt.Printf("[%s] unexpected response format\n", target)
					continue
				}
				if errMsg, ok := result["error"]; ok {
					fmt.Printf("[%s] Error: %v\n", target, errMsg)
				} else {
					fmt.Printf("[%s] %v\n", target, result["stdout"])
				}
			}
			return nil
		}

		// Single-target exec
		assetID := args[0]
		command := strings.Join(args[1:], " ")

		resp, err := c.Post("/api/v2/assets/"+assetID+"/exec", map[string]any{
			"command": command,
			"timeout": timeout,
		})
		if err != nil {
			return err
		}

		if jsonOutput {
			printJSON(json.RawMessage(resp.Data))
			return nil
		}

		var data map[string]any
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
		if output, ok := data["stdout"].(string); ok && output != "" {
			fmt.Println(output)
		}
		if exitCode, ok := data["exit_code"].(float64); ok && exitCode != 0 {
			fmt.Fprintf(os.Stderr, "Exit code: %d\n", int(exitCode))
		}
		return nil
	},
}

func init() {
	execCmd.Flags().String("targets", "", "Comma-separated list of asset IDs for multi-target exec")
	execCmd.Flags().String("group", "", "Group name for multi-target exec")
	execCmd.Flags().Int("timeout", 30, "Command timeout in seconds (max 300)")
	rootCmd.AddCommand(execCmd)
}
