package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

func Execute() int {
	if err := rootCmd.Execute(); err != nil {
		// Determine exit code based on error
		errStr := err.Error()
		switch {
		case strings.Contains(errStr, "status 401") || strings.Contains(errStr, "status 403"):
			return 2 // auth error
		case strings.Contains(errStr, "status 409") || strings.Contains(errStr, "not found"):
			return 3 // asset offline or not found
		case strings.Contains(errStr, "not configured"):
			return 4 // CLI usage error
		default:
			return 1 // general error
		}
	}
	return 0
}

func outputResult(resp *client.V2Response, err error) error {
	if err != nil {
		if jsonOutput {
			printJSON(map[string]string{"error": err.Error()})
		}
		return err
	}
	if jsonOutput {
		printJSON(json.RawMessage(resp.Data))
	}
	return nil
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
