package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/labtether/labtether-cli/internal/client"
)

var (
	cfgHost       string
	cfgAPIKey     string
	cfgAPIKeyFile string
	cfgTLSCAFile  string
	jsonOutput    bool
)

type config struct {
	Host      string `json:"host"`
	APIKey    string `json:"api_key"` // #nosec G117 -- Deliberately persisted only in an ACL-protected, mode-0600 config file.
	TLSCAFile string `json:"tls_ca_file,omitempty"`
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
		case strings.Contains(errStr, "(status 401)") || strings.Contains(errStr, "(status 403)"):
			return 2 // auth error
		case strings.Contains(errStr, "(status 409)") || strings.Contains(errStr, "(status 404)"):
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

func decodeResponseData(resp *client.V2Response, dst any) error {
	if err := json.Unmarshal(resp.Data, dst); err != nil {
		return fmt.Errorf("decode response data: %w", err)
	}
	return nil
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgHost, "host", "", "Hub URL (overrides config)")
	rootCmd.PersistentFlags().StringVar(&cfgAPIKey, "api-key", "", "API key (overrides config)")
	rootCmd.PersistentFlags().StringVar(&cfgAPIKeyFile, "api-key-file", "", "Read API key from a protected file")
	rootCmd.PersistentFlags().StringVar(&cfgTLSCAFile, "tls-ca-file", "", "Trust additional PEM CA certificates for the hub")
	_ = rootCmd.PersistentFlags().MarkDeprecated("api-key", "command-line secrets are visible in shell history and process listings; use --api-key-file or LABTETHER_API_KEY")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
}

func newClient() (*client.Client, error) {
	// Priority: flag > env > config file

	// Config file (lowest priority)
	cfg, cfgErr := loadConfig()
	host := cfg.Host
	key := cfg.APIKey
	caFile := cfg.TLSCAFile

	// Env vars override config
	if v := os.Getenv("LABTETHER_HOST"); v != "" {
		host = v
	}
	if v := os.Getenv("LABTETHER_API_KEY"); v != "" {
		key = v
	}
	if v := os.Getenv("LABTETHER_TLS_CA_FILE"); v != "" {
		caFile = v
	}
	if cfgAPIKeyFile != "" {
		fileKey, err := readProtectedSecretFile(cfgAPIKeyFile)
		if err != nil {
			return nil, err
		}
		key = fileKey
	}

	// Flags override everything
	if cfgHost != "" {
		host = cfgHost
	}
	if cfgTLSCAFile != "" {
		caFile = cfgTLSCAFile
	}
	if cfgAPIKey != "" {
		return nil, fmt.Errorf("refusing API key in command-line arguments; use --api-key-file or LABTETHER_API_KEY")
	}

	host = strings.TrimSpace(host)
	key = strings.TrimSpace(key)
	caFile = strings.TrimSpace(caFile)
	if cfgErr != nil && (host == "" || key == "") {
		return nil, cfgErr
	}
	if host == "" {
		return nil, fmt.Errorf("hub host not configured -- run: labtether-cli config set-host <url>")
	}
	if key == "" {
		return nil, fmt.Errorf("api key not configured -- run: labtether-cli config set-key")
	}

	if err := client.ValidateBaseURL(host); err != nil {
		return nil, err
	}

	return client.NewWithTLSCAFile(host, key, caFile)
}

func readProtectedSecretFile(path string) (string, error) {
	file, err := os.Open(path) // #nosec G304 -- The operator explicitly supplies this secret-file path; its opened descriptor is validated before reading.
	if err != nil {
		return "", fmt.Errorf("read API key file: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("inspect API key file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("API key file must be a regular file")
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		return "", fmt.Errorf("API key file permissions are too broad; require mode 0600")
	}
	data, err := io.ReadAll(io.LimitReader(file, 64*1024+1))
	if err != nil {
		return "", fmt.Errorf("read API key file: %w", err)
	}
	if len(data) > 64*1024 {
		return "", fmt.Errorf("API key file exceeds 64 KiB")
	}
	secret := strings.TrimSpace(string(data))
	if secret == "" {
		return "", fmt.Errorf("API key file is empty")
	}
	return secret, nil
}

func configDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(os.TempDir(), "labtether")
	}
	return filepath.Join(home, ".config", "labtether")
}

func configPath() string {
	return filepath.Join(configDir(), "config.json")
}

func loadConfig() (config, error) {
	var cfg config
	path := configPath()
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("inspect config %s: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return cfg, fmt.Errorf("config %s must be a regular file, not a symlink or device", path)
	}
	if err := hardenConfigFile(path); err != nil {
		return cfg, fmt.Errorf("protect config %s: %w", path, err)
	}
	file, err := os.Open(path) // #nosec G304 -- path is the fixed per-user LabTether configuration path and symlinks were rejected above.
	if err != nil {
		return cfg, fmt.Errorf("read config %s: %w", path, err)
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 1024*1024+1))
	if err != nil {
		return cfg, fmt.Errorf("read config %s: %w", path, err)
	}
	if len(data) > 1024*1024 {
		return cfg, fmt.Errorf("config %s exceeds 1 MiB", path)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config %s: %w", path, err)
	}
	return cfg, nil
}

func saveConfig(cfg config) error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ") // #nosec G117 -- Serialized only into the protected per-user credential config.
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".config-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := hardenConfigFile(tmpPath); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := replaceConfigFile(tmpPath, configPath()); err != nil {
		return err
	}
	return hardenConfigFile(configPath())
}

func printJSON(v any) {
	data, _ := json.MarshalIndent(redactSensitiveJSON(v), "", "  ")
	fmt.Println(string(data))
}

func redactSensitiveJSON(v any) any {
	var decoded any
	switch value := v.(type) {
	case json.RawMessage:
		if err := json.Unmarshal(value, &decoded); err != nil {
			return v
		}
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return v
		}
		if err := json.Unmarshal(data, &decoded); err != nil {
			return v
		}
	}
	return redactDecodedJSON(decoded)
}

func redactDecodedJSON(v any) any {
	switch value := v.(type) {
	case map[string]any:
		for key, child := range value {
			if isSensitiveJSONKey(key) && !isSafeSecretStatus(child) {
				value[key] = "[redacted]"
				continue
			}
			value[key] = redactDecodedJSON(child)
		}
		return value
	case []any:
		for i, child := range value {
			value[i] = redactDecodedJSON(child)
		}
		return value
	default:
		return value
	}
}

func isSensitiveJSONKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "-", "_"), " ", "_"))
	if strings.Contains(normalized, "password") ||
		strings.Contains(normalized, "secret") ||
		strings.Contains(normalized, "token") ||
		strings.Contains(normalized, "api_key") ||
		strings.Contains(normalized, "private_key") {
		return true
	}
	return normalized == "apikey"
}

func isSafeSecretStatus(v any) bool {
	switch value := v.(type) {
	case string:
		return value == "(set)" || value == "(not set)" || value == "[redacted]"
	case bool:
		return true
	default:
		return false
	}
}

func printError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
}
