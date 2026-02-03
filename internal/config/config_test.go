package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromEnvVars(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	t.Setenv("NEXTCLOUD_DOMAIN", "https://cloud.example.com")
	t.Setenv("NEXTCLOUD_USER", "testuser")
	t.Setenv("NEXTCLOUD_PASSWORD", "testpass")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Domain != "https://cloud.example.com" {
		t.Errorf("Domain = %v, want %v", cfg.Domain, "https://cloud.example.com")
	}
	if cfg.User != "testuser" {
		t.Errorf("User = %v, want %v", cfg.User, "testuser")
	}
	if cfg.Password != "testpass" {
		t.Errorf("Password = %v, want %v", cfg.Password, "testpass")
	}
}

func TestLoadFromJSONFile(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	t.Setenv("NEXTCLOUD_DOMAIN", "")
	t.Setenv("NEXTCLOUD_USER", "")
	t.Setenv("NEXTCLOUD_PASSWORD", "")

	// Create config directory and file
	configDir := filepath.Join(tempDir, "cospend")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configContent := `{
  "domain": "https://json.example.com",
  "user": "jsonuser",
  "password": "jsonpass"
}`
	configPath := filepath.Join(configDir, "cospend.json")
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Domain != "https://json.example.com" {
		t.Errorf("Domain = %v, want %v", cfg.Domain, "https://json.example.com")
	}
	if cfg.User != "jsonuser" {
		t.Errorf("User = %v, want %v", cfg.User, "jsonuser")
	}
}

func TestLoadFromYAMLFile(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	t.Setenv("NEXTCLOUD_DOMAIN", "")
	t.Setenv("NEXTCLOUD_USER", "")
	t.Setenv("NEXTCLOUD_PASSWORD", "")

	configDir := filepath.Join(tempDir, "cospend")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configContent := `domain: https://yaml.example.com
user: yamluser
password: yamlpass
`
	configPath := filepath.Join(configDir, "cospend.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Domain != "https://yaml.example.com" {
		t.Errorf("Domain = %v, want %v", cfg.Domain, "https://yaml.example.com")
	}
	if cfg.User != "yamluser" {
		t.Errorf("User = %v, want %v", cfg.User, "yamluser")
	}
}

func TestLoadFromTOMLFile(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	t.Setenv("NEXTCLOUD_DOMAIN", "")
	t.Setenv("NEXTCLOUD_USER", "")
	t.Setenv("NEXTCLOUD_PASSWORD", "")

	configDir := filepath.Join(tempDir, "cospend")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configContent := `domain = "https://toml.example.com"
user = "tomluser"
password = "tomlpass"
`
	configPath := filepath.Join(configDir, "cospend.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Domain != "https://toml.example.com" {
		t.Errorf("Domain = %v, want %v", cfg.Domain, "https://toml.example.com")
	}
	if cfg.User != "tomluser" {
		t.Errorf("User = %v, want %v", cfg.User, "tomluser")
	}
}

func TestEnvVarsOverrideConfigFile(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	t.Setenv("NEXTCLOUD_DOMAIN", "https://env.example.com")
	t.Setenv("NEXTCLOUD_USER", "")
	t.Setenv("NEXTCLOUD_PASSWORD", "")

	configDir := filepath.Join(tempDir, "cospend")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configContent := `{
  "domain": "https://file.example.com",
  "user": "fileuser",
  "password": "filepass"
}`
	configPath := filepath.Join(configDir, "cospend.json")
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Domain should come from env var
	if cfg.Domain != "https://env.example.com" {
		t.Errorf("Domain = %v, want %v", cfg.Domain, "https://env.example.com")
	}
	// User/Password should come from file
	if cfg.User != "fileuser" {
		t.Errorf("User = %v, want %v", cfg.User, "fileuser")
	}
	if cfg.Password != "filepass" {
		t.Errorf("Password = %v, want %v", cfg.Password, "filepass")
	}
}

func TestLoadMissingRequired(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir) // Isolate from real home
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	t.Setenv("NEXTCLOUD_DOMAIN", "")
	t.Setenv("NEXTCLOUD_USER", "")
	t.Setenv("NEXTCLOUD_PASSWORD", "")

	_, err := Load()
	if err == nil {
		t.Error("Load() expected error for missing required fields")
	}
}

func TestSaveJSON(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	cfg := &Config{
		Domain:   "https://test.example.com",
		User:     "testuser",
		Password: "testpass",
	}

	path, err := Save(cfg, "json")
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	expectedPath := filepath.Join(tempDir, "cospend", "cospend.json")
	if path != expectedPath {
		t.Errorf("Save() path = %v, want %v", path, expectedPath)
	}

	// Verify file contents
	loaded, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}
	if loaded.Domain != cfg.Domain {
		t.Errorf("Domain = %v, want %v", loaded.Domain, cfg.Domain)
	}
}

func TestSaveYAML(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	cfg := &Config{
		Domain:   "https://test.example.com",
		User:     "testuser",
		Password: "testpass",
	}

	path, err := Save(cfg, "yaml")
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	expectedPath := filepath.Join(tempDir, "cospend", "cospend.yaml")
	if path != expectedPath {
		t.Errorf("Save() path = %v, want %v", path, expectedPath)
	}

	loaded, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}
	if loaded.Domain != cfg.Domain {
		t.Errorf("Domain = %v, want %v", loaded.Domain, cfg.Domain)
	}
}

func TestSaveTOML(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	cfg := &Config{
		Domain:   "https://test.example.com",
		User:     "testuser",
		Password: "testpass",
	}

	path, err := Save(cfg, "toml")
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	expectedPath := filepath.Join(tempDir, "cospend", "cospend.toml")
	if path != expectedPath {
		t.Errorf("Save() path = %v, want %v", path, expectedPath)
	}

	loaded, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}
	if loaded.Domain != cfg.Domain {
		t.Errorf("Domain = %v, want %v", loaded.Domain, cfg.Domain)
	}
}

func TestGetConfigPath(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir) // Isolate from real home
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	// No config file exists
	if path := GetConfigPath(); path != "" {
		t.Errorf("GetConfigPath() = %v, want empty string", path)
	}

	// Create JSON config
	configDir := filepath.Join(tempDir, "cospend")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}
	jsonPath := filepath.Join(configDir, "cospend.json")
	if err := os.WriteFile(jsonPath, []byte("{}"), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	if path := GetConfigPath(); path != jsonPath {
		t.Errorf("GetConfigPath() = %v, want %v", path, jsonPath)
	}
}

func TestConfigFilePrecedence(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	t.Setenv("NEXTCLOUD_DOMAIN", "")
	t.Setenv("NEXTCLOUD_USER", "")
	t.Setenv("NEXTCLOUD_PASSWORD", "")

	configDir := filepath.Join(tempDir, "cospend")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Create both JSON and YAML - JSON should take precedence
	jsonContent := `{"domain": "https://json.example.com", "user": "jsonuser", "password": "jsonpass"}`
	yamlContent := `domain: https://yaml.example.com
user: yamluser
password: yamlpass`

	if err := os.WriteFile(filepath.Join(configDir, "cospend.json"), []byte(jsonContent), 0600); err != nil {
		t.Fatalf("Failed to write JSON config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "cospend.yaml"), []byte(yamlContent), 0600); err != nil {
		t.Fatalf("Failed to write YAML config: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// JSON should take precedence
	if cfg.Domain != "https://json.example.com" {
		t.Errorf("Domain = %v, want %v (JSON should take precedence)", cfg.Domain, "https://json.example.com")
	}
}

func TestFallbackToDotConfig(t *testing.T) {
	// Create a temp dir to act as HOME
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	// Set XDG_CONFIG_HOME to a different location (simulating macOS default behavior)
	xdgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgDir)
	t.Setenv("NEXTCLOUD_DOMAIN", "")
	t.Setenv("NEXTCLOUD_USER", "")
	t.Setenv("NEXTCLOUD_PASSWORD", "")

	// Create config in ~/.config/cospend/ (fallback location)
	dotConfigDir := filepath.Join(tempHome, ".config", "cospend")
	if err := os.MkdirAll(dotConfigDir, 0700); err != nil {
		t.Fatalf("Failed to create .config dir: %v", err)
	}

	configContent := `{"domain": "https://dotconfig.example.com", "user": "dotconfiguser", "password": "dotconfigpass"}`
	configPath := filepath.Join(dotConfigDir, "cospend.json")
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Should find config in fallback ~/.config/cospend/
	foundPath := GetConfigPath()
	if foundPath != configPath {
		t.Errorf("GetConfigPath() = %v, want %v", foundPath, configPath)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Domain != "https://dotconfig.example.com" {
		t.Errorf("Domain = %v, want %v", cfg.Domain, "https://dotconfig.example.com")
	}
}

func TestXDGTakesPrecedenceOverDotConfig(t *testing.T) {
	// Create a temp dir to act as HOME
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	// Set XDG_CONFIG_HOME
	xdgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgDir)
	t.Setenv("NEXTCLOUD_DOMAIN", "")
	t.Setenv("NEXTCLOUD_USER", "")
	t.Setenv("NEXTCLOUD_PASSWORD", "")

	// Create config in both locations
	xdgConfigDir := filepath.Join(xdgDir, "cospend")
	if err := os.MkdirAll(xdgConfigDir, 0700); err != nil {
		t.Fatalf("Failed to create XDG config dir: %v", err)
	}
	xdgContent := `{"domain": "https://xdg.example.com", "user": "xdguser", "password": "xdgpass"}`
	xdgPath := filepath.Join(xdgConfigDir, "cospend.json")
	if err := os.WriteFile(xdgPath, []byte(xdgContent), 0600); err != nil {
		t.Fatalf("Failed to write XDG config file: %v", err)
	}

	dotConfigDir := filepath.Join(tempHome, ".config", "cospend")
	if err := os.MkdirAll(dotConfigDir, 0700); err != nil {
		t.Fatalf("Failed to create .config dir: %v", err)
	}
	dotContent := `{"domain": "https://dotconfig.example.com", "user": "dotconfiguser", "password": "dotconfigpass"}`
	if err := os.WriteFile(filepath.Join(dotConfigDir, "cospend.json"), []byte(dotContent), 0600); err != nil {
		t.Fatalf("Failed to write .config file: %v", err)
	}

	// XDG should take precedence
	foundPath := GetConfigPath()
	if foundPath != xdgPath {
		t.Errorf("GetConfigPath() = %v, want %v (XDG should take precedence)", foundPath, xdgPath)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Domain != "https://xdg.example.com" {
		t.Errorf("Domain = %v, want %v (XDG should take precedence)", cfg.Domain, "https://xdg.example.com")
	}
}
