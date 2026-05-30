package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CLI configuration",
}

var configSetHostCmd = &cobra.Command{
	Use:   "set-host <url>",
	Short: "Set the hub URL",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadConfig()
		cfg.Host = strings.TrimRight(strings.TrimSpace(args[0]), "/")
		if err := saveConfig(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		fmt.Printf("Hub host set to: %s\n", cfg.Host)
		return nil
	},
}

var configSetKeyCmd = &cobra.Command{
	Use:   "set-key <key>",
	Short: "Set the API key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadConfig()
		cfg.APIKey = strings.TrimSpace(args[0])
		if err := saveConfig(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		fmt.Println("API key saved.")
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadConfig()
		keyStatus := apiKeyStatus(cfg.APIKey != "")

		if jsonOutput {
			printJSON(map[string]any{
				"host":    cfg.Host,
				"api_key": keyStatus,
				"config":  configPath(),
			})
			return nil
		}

		fmt.Printf("Host:    %s\n", cfg.Host)
		fmt.Printf("API Key: %s\n", keyStatus)
		fmt.Printf("Config:  %s\n", configPath())
		return nil
	},
}

func apiKeyStatus(configured bool) string {
	if configured {
		return "(set)"
	}
	return "(not set)"
}

func init() {
	configCmd.AddCommand(configSetHostCmd, configSetKeyCmd, configShowCmd)
	rootCmd.AddCommand(configCmd)
}
