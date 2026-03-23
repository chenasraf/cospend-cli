package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestNewConfigCommand(t *testing.T) {
	cmd := NewConfigCommand()

	if cmd.Use != "config" {
		t.Errorf("Wrong Use: %s", cmd.Use)
	}

	// Should have set and get subcommands
	subCmds := cmd.Commands()
	names := make(map[string]bool)
	for _, c := range subCmds {
		names[c.Name()] = true
	}
	if !names["set"] {
		t.Error("Missing 'set' subcommand")
	}
	if !names["get"] {
		t.Error("Missing 'get' subcommand")
	}
}

func TestConfigSetDefaultProject(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	t.Setenv("HOME", tempDir)

	// Create initial config file
	configDir := filepath.Join(tempDir, "cospend")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}
	configPath := filepath.Join(configDir, "cospend.json")
	initialConfig := `{"domain": "https://example.com", "user": "alice", "password": "pass"}`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	cmd := NewConfigCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"set", "default-project", "myproject"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !bytes.Contains(stdout.Bytes(), []byte("Set default-project = myproject")) {
		t.Errorf("Expected success message, got: %s", stdout.String())
	}

	// Verify it was saved
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}
	if !bytes.Contains(data, []byte("myproject")) {
		t.Errorf("Config file should contain 'myproject', got: %s", string(data))
	}
}

func TestConfigGetDefaultProject(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	t.Setenv("HOME", tempDir)

	configDir := filepath.Join(tempDir, "cospend")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}
	configPath := filepath.Join(configDir, "cospend.json")
	configContent := `{"domain": "https://example.com", "user": "alice", "password": "pass", "default_project": "myproject"}`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	cmd := NewConfigCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"get", "default-project"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !bytes.Contains(stdout.Bytes(), []byte("myproject")) {
		t.Errorf("Expected 'myproject', got: %s", stdout.String())
	}
}

func TestConfigGetDefaultProjectNotSet(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	t.Setenv("HOME", tempDir)

	configDir := filepath.Join(tempDir, "cospend")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}
	configPath := filepath.Join(configDir, "cospend.json")
	configContent := `{"domain": "https://example.com", "user": "alice", "password": "pass"}`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	cmd := NewConfigCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"get", "default-project"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !bytes.Contains(stdout.Bytes(), []byte("(not set)")) {
		t.Errorf("Expected '(not set)', got: %s", stdout.String())
	}
}

func TestConfigSetUnknownKey(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	t.Setenv("HOME", tempDir)

	configDir := filepath.Join(tempDir, "cospend")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}
	configPath := filepath.Join(configDir, "cospend.json")
	if err := os.WriteFile(configPath, []byte(`{"domain":"x","user":"u","password":"p"}`), 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	cmd := NewConfigCommand()
	cmd.SetArgs([]string{"set", "unknown-key", "value"})

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error for unknown key")
	}
}

func TestConfigNoConfigFile(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	t.Setenv("HOME", tempDir)

	cmd := NewConfigCommand()
	cmd.SetArgs([]string{"get", "default-project"})

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error when no config file exists")
	}
}

func TestConfigSetPreservesExistingFields(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	t.Setenv("HOME", tempDir)

	configDir := filepath.Join(tempDir, "cospend")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}
	configPath := filepath.Join(configDir, "cospend.json")
	initialConfig := `{"domain": "https://example.com", "user": "alice", "password": "secret123"}`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	cmd := NewConfigCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"set", "default-project", "myproject"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify existing fields are preserved
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}
	content := string(data)
	if !bytes.Contains(data, []byte("https://example.com")) {
		t.Errorf("Domain should be preserved, got: %s", content)
	}
	if !bytes.Contains(data, []byte("alice")) {
		t.Errorf("User should be preserved, got: %s", content)
	}
	if !bytes.Contains(data, []byte("secret123")) {
		t.Errorf("Password should be preserved, got: %s", content)
	}
}

func TestConfigSetYAML(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	t.Setenv("HOME", tempDir)

	configDir := filepath.Join(tempDir, "cospend")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}
	configPath := filepath.Join(configDir, "cospend.yaml")
	initialConfig := "domain: https://example.com\nuser: alice\npassword: pass\n"
	if err := os.WriteFile(configPath, []byte(initialConfig), 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	cmd := NewConfigCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"set", "default-project", "yamlproject"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}
	if !bytes.Contains(data, []byte("yamlproject")) {
		t.Errorf("Config should contain 'yamlproject', got: %s", string(data))
	}
}

func TestConfigSetTOML(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	t.Setenv("HOME", tempDir)

	configDir := filepath.Join(tempDir, "cospend")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}
	configPath := filepath.Join(configDir, "cospend.toml")
	initialConfig := "domain = \"https://example.com\"\nuser = \"alice\"\npassword = \"pass\"\n"
	if err := os.WriteFile(configPath, []byte(initialConfig), 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	cmd := NewConfigCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"set", "default-project", "tomlproject"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}
	if !bytes.Contains(data, []byte("tomlproject")) {
		t.Errorf("Config should contain 'tomlproject', got: %s", string(data))
	}
}
