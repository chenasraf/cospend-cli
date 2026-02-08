package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chenasraf/cospend-cli/internal/config"
)

func resetInitFlags() {
	configFormat = "json"
}

// mockOpenBrowser replaces openBrowser for testing and returns a restore function
func mockOpenBrowser() (openedURL *string, restore func()) {
	original := openBrowser
	var url string
	openBrowser = func(u string) error {
		url = u
		return nil
	}
	return &url, func() { openBrowser = original }
}

func TestNewInitCommand(t *testing.T) {
	resetInitFlags()
	defer resetInitFlags()

	cmd := NewInitCommand()

	if cmd.Use != "init" {
		t.Errorf("Wrong Use: %s", cmd.Use)
	}

	// Check format flag exists
	if cmd.Flags().Lookup("format") == nil {
		t.Error("Missing flag: format")
	}

	// Check short flag
	flag := cmd.Flags().ShorthandLookup("f")
	if flag == nil {
		t.Error("Missing short flag: -f")
	} else if flag.Name != "format" {
		t.Errorf("Short flag -f maps to %s, want format", flag.Name)
	}
}

func TestInitCommandInvalidFormat(t *testing.T) {
	resetInitFlags()
	defer resetInitFlags()

	cmd := NewInitCommand()
	cmd.SetArgs([]string{"--format", "xml"})

	var stderr bytes.Buffer
	cmd.SetErr(&stderr)

	// Provide empty input to avoid blocking
	cmd.SetIn(strings.NewReader("\n"))

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error for invalid format")
	}
	if !strings.Contains(err.Error(), "unsupported format") {
		t.Errorf("Error should mention unsupported format: %v", err)
	}
}

func TestLoginFlowSuccess(t *testing.T) {
	// Mock openBrowser to prevent actual browser opening
	openedURL, restore := mockOpenBrowser()
	defer restore()

	// Track the polling attempts
	pollCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Initial login flow request
		if r.URL.Path == "/index.php/login/v2" && r.Method == "POST" {
			// Check User-Agent
			if ua := r.Header.Get("User-Agent"); ua != "Cospend CLI" {
				t.Errorf("User-Agent = %s, want Cospend CLI", ua)
			}

			resp := map[string]interface{}{
				"poll": map[string]string{
					"token":    "test-token-123",
					"endpoint": "http://" + r.Host + "/login/v2/poll",
				},
				"login": "http://" + r.Host + "/login/v2/flow/abc123",
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		// Poll endpoint
		if r.URL.Path == "/login/v2/poll" && r.Method == "POST" {
			pollCount++
			// Check User-Agent
			if ua := r.Header.Get("User-Agent"); ua != "Cospend CLI" {
				t.Errorf("Poll User-Agent = %s, want Cospend CLI", ua)
			}

			// Check token
			if err := r.ParseForm(); err != nil {
				t.Errorf("ParseForm error: %v", err)
			}
			if token := r.FormValue("token"); token != "test-token-123" {
				t.Errorf("Token = %s, want test-token-123", token)
			}

			// Return success on first poll
			resp := map[string]string{
				"server":      "https://cloud.example.com",
				"loginName":   "testuser",
				"appPassword": "app-password-xyz",
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		t.Errorf("Unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	// Create a mock command to test loginFlowAuth
	cmd := NewInitCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})

	cfg, err := loginFlowAuth(cmd, server.URL)
	if err != nil {
		t.Fatalf("loginFlowAuth error: %v", err)
	}

	if cfg.Domain != "https://cloud.example.com" {
		t.Errorf("Domain = %s, want https://cloud.example.com", cfg.Domain)
	}
	if cfg.User != "testuser" {
		t.Errorf("User = %s, want testuser", cfg.User)
	}
	if cfg.Password != "app-password-xyz" {
		t.Errorf("Password = %s, want app-password-xyz", cfg.Password)
	}

	if pollCount != 1 {
		t.Errorf("Poll count = %d, want 1", pollCount)
	}

	// Verify the correct URL was passed to openBrowser
	if !strings.Contains(*openedURL, "/login/v2/flow/abc123") {
		t.Errorf("openBrowser URL = %s, want to contain /login/v2/flow/abc123", *openedURL)
	}
}

func TestLoginFlowInitError(t *testing.T) {
	// Mock openBrowser to prevent actual browser opening
	_, restore := mockOpenBrowser()
	defer restore()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cmd := NewInitCommand()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	_, err := loginFlowAuth(cmd, server.URL)
	if err == nil {
		t.Error("Expected error for failed login flow initiation")
	}
}

func TestPromptPassword(t *testing.T) {
	// Test password prompt in non-terminal mode (fallback to regular input)
	cmd := NewInitCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetIn(strings.NewReader("secretpass\n"))

	result, err := promptPassword(cmd, "Enter password")
	if err != nil {
		t.Fatalf("promptPassword error: %v", err)
	}

	if result != "secretpass" {
		t.Errorf("Result = %s, want secretpass", result)
	}
}

func TestPromptSelectFallback(t *testing.T) {
	cmd := NewInitCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	options := []selectOption{
		{label: "Option A", description: "First option"},
		{label: "Option B", description: "Second option"},
	}

	tests := []struct {
		name     string
		input    string
		expected int
		wantErr  bool
	}{
		{"default selection", "\n", 0, false},
		{"select first", "1\n", 0, false},
		{"select second", "2\n", 1, false},
		{"invalid choice", "5\n", 0, true},
		{"invalid input", "abc\n", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout.Reset()
			cmd.SetIn(strings.NewReader(tt.input))

			selected, err := promptSelectFallback(cmd, options)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if selected != tt.expected {
				t.Errorf("Selected = %d, want %d", selected, tt.expected)
			}
		})
	}
}

func TestDomainAutoPrependHTTPS(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"cloud.example.com", "https://cloud.example.com"},
		{"https://cloud.example.com", "https://cloud.example.com"},
		{"http://cloud.example.com", "http://cloud.example.com"},
		{"HTTPS://CLOUD.EXAMPLE.COM", "HTTPS://CLOUD.EXAMPLE.COM"},
		{"HTTP://cloud.example.com", "HTTP://cloud.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			domain := config.NormalizeURL(tt.input)
			if domain != tt.expected {
				t.Errorf("Domain = %s, want %s", domain, tt.expected)
			}
		})
	}
}

func TestSaveToPath(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		Domain:   "https://test.example.com",
		User:     "testuser",
		Password: "testpass",
	}

	tests := []struct {
		name string
		ext  string
	}{
		{"JSON", ".json"},
		{"YAML", ".yaml"},
		{"TOML", ".toml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(tempDir, "config"+tt.ext)

			savedPath, err := config.SaveToPath(cfg, path)
			if err != nil {
				t.Fatalf("SaveToPath error: %v", err)
			}

			if savedPath != path {
				t.Errorf("SaveToPath returned %s, want %s", savedPath, path)
			}

			// Verify file exists and can be loaded
			loaded, err := config.LoadFromFile(path)
			if err != nil {
				t.Fatalf("LoadFromFile error: %v", err)
			}

			if loaded.Domain != cfg.Domain {
				t.Errorf("Domain = %s, want %s", loaded.Domain, cfg.Domain)
			}
			if loaded.User != cfg.User {
				t.Errorf("User = %s, want %s", loaded.User, cfg.User)
			}
			if loaded.Password != cfg.Password {
				t.Errorf("Password = %s, want %s", loaded.Password, cfg.Password)
			}
		})
	}
}

func TestSaveToPathCreatesDirectory(t *testing.T) {
	tempDir := t.TempDir()
	nestedPath := filepath.Join(tempDir, "nested", "dir", "config.json")

	cfg := &config.Config{
		Domain:   "https://test.example.com",
		User:     "testuser",
		Password: "testpass",
	}

	_, err := config.SaveToPath(cfg, nestedPath)
	if err != nil {
		t.Fatalf("SaveToPath error: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(nestedPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}
}

func TestSaveToPathUnsupportedFormat(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "config.xml")

	cfg := &config.Config{
		Domain:   "https://test.example.com",
		User:     "testuser",
		Password: "testpass",
	}

	_, err := config.SaveToPath(cfg, path)
	if err == nil {
		t.Error("Expected error for unsupported format")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("Error should mention unsupported format: %v", err)
	}
}

func TestConfigOverwriteSameLocation(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	t.Setenv("HOME", tempDir)

	// Create initial config in a non-default location (simulating ~/.config fallback)
	customDir := filepath.Join(tempDir, "custom", "location")
	if err := os.MkdirAll(customDir, 0700); err != nil {
		t.Fatalf("Failed to create custom dir: %v", err)
	}
	customPath := filepath.Join(customDir, "cospend.yaml")

	initialCfg := &config.Config{
		Domain:   "https://initial.example.com",
		User:     "initialuser",
		Password: "initialpass",
	}
	if _, err := config.SaveToPath(initialCfg, customPath); err != nil {
		t.Fatalf("Failed to save initial config: %v", err)
	}

	// Now save updated config to the same path
	updatedCfg := &config.Config{
		Domain:   "https://updated.example.com",
		User:     "updateduser",
		Password: "updatedpass",
	}
	savedPath, err := config.SaveToPath(updatedCfg, customPath)
	if err != nil {
		t.Fatalf("SaveToPath error: %v", err)
	}

	// Verify it saved to the exact same path
	if savedPath != customPath {
		t.Errorf("SaveToPath returned %s, want %s", savedPath, customPath)
	}

	// Verify contents were updated
	loaded, err := config.LoadFromFile(customPath)
	if err != nil {
		t.Fatalf("LoadFromFile error: %v", err)
	}
	if loaded.Domain != "https://updated.example.com" {
		t.Errorf("Domain = %s, want https://updated.example.com", loaded.Domain)
	}
}

func TestOpenBrowserMock(t *testing.T) {
	// Test that the mock mechanism works correctly
	openedURL, restore := mockOpenBrowser()
	defer restore()

	err := openBrowser("https://example.com/test")
	if err != nil {
		t.Errorf("Mock openBrowser returned error: %v", err)
	}

	if *openedURL != "https://example.com/test" {
		t.Errorf("openBrowser URL = %s, want https://example.com/test", *openedURL)
	}
}

func TestPromptString(t *testing.T) {
	cmd := NewInitCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetIn(strings.NewReader("test input\n"))

	result, err := promptString(cmd, "Enter value")
	if err != nil {
		t.Fatalf("promptString error: %v", err)
	}

	if result != "test input" {
		t.Errorf("Result = %s, want 'test input'", result)
	}

	if !strings.Contains(stdout.String(), "Enter value:") {
		t.Errorf("Prompt not shown: %s", stdout.String())
	}
}

func TestPromptYesNo(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"y\n", true},
		{"Y\n", true},
		{"yes\n", true},
		{"YES\n", true},
		{"n\n", false},
		{"N\n", false},
		{"no\n", false},
		{"\n", false},
		{"anything\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			cmd := NewInitCommand()
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetIn(strings.NewReader(tt.input))

			result, err := promptYesNo(cmd, "Confirm?")
			if err != nil {
				t.Fatalf("promptYesNo error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("Result = %v, want %v for input %q", result, tt.expected, tt.input)
			}
		})
	}
}
