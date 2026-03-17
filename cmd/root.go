package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/labtether/labtether-cli/internal/client"
)

var (
	cfgHost    string
	cfgAPIKey  string
	jsonOutput bool
)

type config struct {
	Host   string `json:"host"`
	APIKey string `json:"api_key"`
}

var rootCmd = &cobra.Command{
	Use:   "labtether-cli",
	Short: "LabTether CLI -- control your homelab from the command line",
	Long:  "labtether-cli is a command-line interface for the LabTether hub API. It lets you manage assets, run commands, manage files, and control your entire homelab.",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgHost, "host", "", "Hub URL (overrides config)")
	rootCmd.PersistentFlags().StringVar(&cfgAPIKey, "api-key", "", "API key (overrides config)")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
}

func newClient() (*client.Client, error) {
	host := cfgHost
	key := cfgAPIKey

	// Env vars override flags
	if v := os.Getenv("LABTETHER_HOST"); v != "" {
		host = v
	}
	if v := os.Getenv("LABTETHER_API_KEY"); v != "" {
		key = v
	}

	// Config file as fallback
	if host == "" || key == "" {
		cfg := loadConfig()
		if host == "" {
			host = cfg.Host
		}
		if key == "" {
			key = cfg.APIKey
		}
	}

	if host == "" {
		return nil, fmt.Errorf("hub host not configured -- run: labtether-cli config set-host <url>")
	}
	if key == "" {
		return nil, fmt.Errorf("api key not configured -- run: labtether-cli config set-key <key>")
	}

	return client.New(host, key), nil
}

func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "labtether")
}

func configPath() string {
	return filepath.Join(configDir(), "config.json")
}

func loadConfig() config {
	var cfg config
	data, err := os.ReadFile(configPath())
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal(data, &cfg)
	return cfg
}

func saveConfig(cfg config) error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	return os.WriteFile(configPath(), data, 0600)
}

func printJSON(v any) {
	data, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(data))
}

func printError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
}
