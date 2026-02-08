package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/adrg/xdg"
	"gopkg.in/yaml.v3"
)

const appName = "cospend"

// NormalizeURL trims trailing slashes and prepends https:// if no scheme is present.
func NormalizeURL(url string) string {
	url = strings.TrimRight(url, "/")
	lower := strings.ToLower(url)
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		url = "https://" + url
	}
	return url
}

// Config holds the Nextcloud configuration
type Config struct {
	Domain   string `json:"domain" yaml:"domain" toml:"domain"`
	User     string `json:"user" yaml:"user" toml:"user"`
	Password string `json:"password" yaml:"password" toml:"password"`
}

// configExtensions lists supported config file extensions in order of preference
var configExtensions = []string{".json", ".yaml", ".yml", ".toml"}

// GetConfigDir returns the primary config directory path (used for saving)
func GetConfigDir() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, appName)
	}
	return filepath.Join(xdg.ConfigHome, appName)
}

// getConfigDirs returns all config directories to search, in order of preference
func getConfigDirs() []string {
	dirs := []string{GetConfigDir()}

	// Also check ~/.config/cospend/ as fallback (even on macOS)
	if home, err := os.UserHomeDir(); err == nil {
		dotConfigDir := filepath.Join(home, ".config", appName)
		// Only add if it's different from the primary dir
		if dotConfigDir != dirs[0] {
			dirs = append(dirs, dotConfigDir)
		}
	}

	return dirs
}

// GetConfigPath returns the path to an existing config file, or empty string if none found
func GetConfigPath() string {
	for _, configDir := range getConfigDirs() {
		for _, ext := range configExtensions {
			path := filepath.Join(configDir, appName+ext)
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}
	return ""
}

// LoadFromFile reads configuration from a config file
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	ext := filepath.Ext(path)

	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("parsing JSON config: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("parsing YAML config: %w", err)
		}
	case ".toml":
		if err := toml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("parsing TOML config: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported config format: %s", ext)
	}

	return &cfg, nil
}

// Load reads configuration with the following precedence:
// 1. Environment variables (override config file)
// 2. Config file
func Load() (*Config, error) {
	var cfg Config

	// Try to load from config file first
	if configPath := GetConfigPath(); configPath != "" {
		fileCfg, err := LoadFromFile(configPath)
		if err != nil {
			return nil, err
		}
		cfg = *fileCfg
	}

	// Environment variables override config file values
	if domain := os.Getenv("NEXTCLOUD_DOMAIN"); domain != "" {
		cfg.Domain = domain
	}
	if user := os.Getenv("NEXTCLOUD_USER"); user != "" {
		cfg.User = user
	}
	if password := os.Getenv("NEXTCLOUD_PASSWORD"); password != "" {
		cfg.Password = password
	}

	// Validate required fields
	if cfg.Domain == "" {
		return nil, errors.New("domain is required (set in config file or NEXTCLOUD_DOMAIN env var)")
	}
	if cfg.User == "" {
		return nil, errors.New("user is required (set in config file or NEXTCLOUD_USER env var)")
	}
	if cfg.Password == "" {
		return nil, errors.New("password is required (set in config file or NEXTCLOUD_PASSWORD env var)")
	}

	return &cfg, nil
}

// Save writes configuration to a file in the specified format in the default config directory
func Save(cfg *Config, format string) (string, error) {
	configDir := GetConfigDir()
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", fmt.Errorf("creating config directory: %w", err)
	}

	var ext string
	switch format {
	case "json":
		ext = ".json"
	case "yaml", "yml":
		ext = ".yaml"
	case "toml":
		ext = ".toml"
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}

	path := filepath.Join(configDir, appName+ext)
	return SaveToPath(cfg, path)
}

// SaveToPath writes configuration to a specific file path (format determined by extension)
func SaveToPath(cfg *Config, path string) (string, error) {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("creating config directory: %w", err)
	}

	var data []byte
	var err error
	ext := filepath.Ext(path)

	switch ext {
	case ".json":
		data, err = json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return "", fmt.Errorf("encoding JSON: %w", err)
		}
		data = append(data, '\n')
	case ".yaml", ".yml":
		data, err = yaml.Marshal(cfg)
		if err != nil {
			return "", fmt.Errorf("encoding YAML: %w", err)
		}
	case ".toml":
		data, err = tomlMarshal(cfg)
		if err != nil {
			return "", fmt.Errorf("encoding TOML: %w", err)
		}
	default:
		return "", fmt.Errorf("unsupported config format: %s", ext)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return "", fmt.Errorf("writing config file: %w", err)
	}

	return path, nil
}

// tomlMarshal encodes config to TOML format
func tomlMarshal(cfg *Config) ([]byte, error) {
	content := fmt.Sprintf(`domain = %q
user = %q
password = %q
`, cfg.Domain, cfg.User, cfg.Password)
	return []byte(content), nil
}
