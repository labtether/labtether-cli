package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/labtether/labtether-cli/internal/client"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		cfg.Host = strings.TrimRight(strings.TrimSpace(args[0]), "/")
		if err := saveConfig(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		fmt.Printf("Hub host set to: %s\n", cfg.Host)
		return nil
	},
}

var configSetKeyCmd = &cobra.Command{
	Use:   "set-key",
	Short: "Read and store an API key securely",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		key, err := readAPIKey(cmd)
		if err != nil {
			return err
		}
		cfg.APIKey = key
		if err := saveConfig(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		fmt.Println("API key saved.")
		return nil
	},
}

var configSetCACmd = &cobra.Command{
	Use:   "set-ca <pem-file>",
	Short: "Trust a private CA certificate bundle for the hub",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		caFile, err := filepath.Abs(strings.TrimSpace(args[0]))
		if err != nil {
			return fmt.Errorf("resolve TLS CA file: %w", err)
		}
		if err := client.ValidateTLSCAFile(caFile); err != nil {
			return err
		}
		cfg.TLSCAFile = caFile
		if err := saveConfig(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		fmt.Printf("TLS CA file set to: %s\n", caFile)
		return nil
	},
}

func readAPIKey(cmd *cobra.Command) (string, error) {
	reader := cmd.InOrStdin()
	var data []byte
	var err error
	if input, ok := reader.(*os.File); ok && term.IsTerminal(int(input.Fd())) { // #nosec G115 -- x/term requires an int descriptor; os.File descriptors are OS-sized handles.
		_, _ = fmt.Fprint(cmd.ErrOrStderr(), "API key: ")
		data, err = term.ReadPassword(int(input.Fd())) // #nosec G115 -- See descriptor rationale above.
		_, _ = fmt.Fprintln(cmd.ErrOrStderr())
	} else {
		data, err = io.ReadAll(io.LimitReader(reader, 64*1024+1))
	}
	if err != nil {
		return "", fmt.Errorf("read API key: %w", err)
	}
	if len(data) > 64*1024 {
		return "", fmt.Errorf("API key exceeds 64 KiB")
	}
	key := strings.TrimSpace(string(data))
	if key == "" {
		return "", fmt.Errorf("API key is required")
	}
	return key, nil
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		keyStatus := apiKeyStatus(cfg.APIKey != "")

		if jsonOutput {
			printJSON(map[string]any{
				"host":        cfg.Host,
				"api_key":     keyStatus,
				"tls_ca_file": cfg.TLSCAFile,
				"config":      configPath(),
			})
			return nil
		}

		fmt.Printf("Host:    %s\n", cfg.Host)
		fmt.Printf("API Key: %s\n", keyStatus)
		fmt.Printf("TLS CA:  %s\n", configuredPathStatus(cfg.TLSCAFile))
		fmt.Printf("Config:  %s\n", configPath())
		return nil
	},
}

func configuredPathStatus(path string) string {
	if strings.TrimSpace(path) == "" {
		return "(system trust only)"
	}
	return path
}

func apiKeyStatus(configured bool) string {
	if configured {
		return "(set)"
	}
	return "(not set)"
}

func init() {
	configCmd.AddCommand(configSetHostCmd, configSetKeyCmd, configSetCACmd, configShowCmd)
	rootCmd.AddCommand(configCmd)
}
