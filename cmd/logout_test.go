package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestNewLogoutCommand(t *testing.T) {
	cmd := NewLogoutCommand()
	if cmd.Use != "logout" {
		t.Errorf("Wrong Use: %s", cmd.Use)
	}
}

func TestLogoutRemovesConfigAndCache(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	t.Setenv("XDG_CACHE_HOME", tempDir)
	t.Setenv("HOME", tempDir)

	// Create config file
	configDir := filepath.Join(tempDir, "cospend")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}
	configPath := filepath.Join(configDir, "cospend.json")
	if err := os.WriteFile(configPath, []byte(`{"domain":"x","user":"u","password":"p"}`), 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Create cache directory with a file in a separate temp dir
	cacheTempDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheTempDir)
	cacheDir := filepath.Join(cacheTempDir, "cospend")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("Failed to create cache dir: %v", err)
	}
	cachePath := filepath.Join(cacheDir, "test-project.json")
	if err := os.WriteFile(cachePath, []byte(`{}`), 0600); err != nil {
		t.Fatalf("Failed to write cache: %v", err)
	}

	cmd := NewLogoutCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Config should be removed
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Error("Config file should have been removed")
	}

	// Cache dir should be removed
	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Error("Cache directory should have been removed")
	}

	output := stdout.String()
	if !bytes.Contains([]byte(output), []byte("Logged out successfully")) {
		t.Errorf("Expected success message, got: %s", output)
	}
}

func TestLogoutNothingToRemove(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	t.Setenv("XDG_CACHE_HOME", tempDir)
	t.Setenv("HOME", tempDir)

	cmd := NewLogoutCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := stdout.String()
	if !bytes.Contains([]byte(output), []byte("Nothing to remove")) {
		t.Errorf("Expected 'Nothing to remove' message, got: %s", output)
	}
}

func TestLogoutConfigOnly(t *testing.T) {
	tempDir := t.TempDir()
	cacheTempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	t.Setenv("XDG_CACHE_HOME", cacheTempDir)
	t.Setenv("HOME", tempDir)

	// Create config file only (no cache)
	configDir := filepath.Join(tempDir, "cospend")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}
	configPath := filepath.Join(configDir, "cospend.json")
	if err := os.WriteFile(configPath, []byte(`{"domain":"x","user":"u","password":"p"}`), 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	cmd := NewLogoutCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Error("Config file should have been removed")
	}

	output := stdout.String()
	if !bytes.Contains([]byte(output), []byte("Removed config")) {
		t.Errorf("Expected 'Removed config' message, got: %s", output)
	}
	if !bytes.Contains([]byte(output), []byte("Logged out successfully")) {
		t.Errorf("Expected success message, got: %s", output)
	}
}
