package cmd

import (
	"fmt"

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
		cfg.Host = args[0]
		if err := saveConfig(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		fmt.Printf("Hub host set to: %s\n", args[0])
		return nil
	},
}

var configSetKeyCmd = &cobra.Command{
	Use:   "set-key <key>",
	Short: "Set the API key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadConfig()
		cfg.APIKey = args[0]
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

		// Mask the API key consistently
		masked := ""
		if len(cfg.APIKey) >= 10 {
			masked = cfg.APIKey[:6] + "..." + cfg.APIKey[len(cfg.APIKey)-4:]
		} else if cfg.APIKey != "" {
			masked = "(set)"
		}

		if jsonOutput {
			printJSON(map[string]any{
				"host":    cfg.Host,
				"api_key": masked,
				"config":  configPath(),
			})
			return nil
		}

		fmt.Printf("Host:    %s\n", cfg.Host)
		if cfg.APIKey != "" {
			fmt.Printf("API Key: %s\n", masked)
		} else {
			fmt.Println("API Key: (not set)")
		}
		fmt.Printf("Config:  %s\n", configPath())
		return nil
	},
}

func init() {
	configCmd.AddCommand(configSetHostCmd, configSetKeyCmd, configShowCmd)
	rootCmd.AddCommand(configCmd)
}
