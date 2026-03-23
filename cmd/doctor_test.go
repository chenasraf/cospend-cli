package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/chenasraf/cospend-cli/internal/api"
)

func TestNewDoctorCommand(t *testing.T) {
	cmd := NewDoctorCommand()
	if cmd.Use != "doctor" {
		t.Errorf("Wrong Use: %s", cmd.Use)
	}
}

func TestDoctorAllPassing(t *testing.T) {
	project := api.Project{
		ID:   "myproject",
		Name: "My Project",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/status.php":
			_, _ = w.Write([]byte(`{"installed":true}`))
		case "/ocs/v2.php/cloud/user":
			_ = json.NewEncoder(w).Encode(makeOCSResponse(200, map[string]string{"locale": "en_US", "language": "en"}))
		case "/ocs/v2.php/apps/cospend/api/v1/projects/myproject":
			_ = json.NewEncoder(w).Encode(makeOCSResponse(200, project))
		}
	}))
	defer server.Close()

	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", tempDir)

	configDir := filepath.Join(tempDir, "cospend")
	_ = os.MkdirAll(configDir, 0700)
	configContent := `{"domain": "` + server.URL + `", "user": "testuser", "password": "testpass", "default_project": "myproject"}`
	_ = os.WriteFile(filepath.Join(configDir, "cospend.json"), []byte(configContent), 0600)

	cmd := NewDoctorCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := stdout.String()
	if !bytes.Contains([]byte(output), []byte("[ok] Config file")) {
		t.Errorf("Should show config ok, got: %s", output)
	}
	if !bytes.Contains([]byte(output), []byte("[ok] Required fields")) {
		t.Errorf("Should show fields ok, got: %s", output)
	}
	if !bytes.Contains([]byte(output), []byte("[ok] Server reachable")) {
		t.Errorf("Should show server ok, got: %s", output)
	}
	if !bytes.Contains([]byte(output), []byte("[ok] Authentication")) {
		t.Errorf("Should show auth ok, got: %s", output)
	}
	if !bytes.Contains([]byte(output), []byte("[ok] Default project")) {
		t.Errorf("Should show project ok, got: %s", output)
	}
	if !bytes.Contains([]byte(output), []byte("All checks passed")) {
		t.Errorf("Should show all passed, got: %s", output)
	}
}

func TestDoctorNoConfig(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	t.Setenv("HOME", tempDir)

	cmd := NewDoctorCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := stdout.String()
	if !bytes.Contains([]byte(output), []byte("[!!] Config file")) {
		t.Errorf("Should show config error, got: %s", output)
	}
	if !bytes.Contains([]byte(output), []byte("Some checks failed")) {
		t.Errorf("Should show failure summary, got: %s", output)
	}
}

func TestDoctorMissingFields(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	t.Setenv("HOME", tempDir)

	configDir := filepath.Join(tempDir, "cospend")
	_ = os.MkdirAll(configDir, 0700)
	_ = os.WriteFile(filepath.Join(configDir, "cospend.json"), []byte(`{"domain": "https://example.com"}`), 0600)

	cmd := NewDoctorCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := stdout.String()
	if !bytes.Contains([]byte(output), []byte("[!!] Required fields")) {
		t.Errorf("Should show missing fields, got: %s", output)
	}
	if !bytes.Contains([]byte(output), []byte("user and password")) {
		t.Errorf("Should list missing fields, got: %s", output)
	}
}

func TestDoctorAuthFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/status.php":
			_, _ = w.Write([]byte(`{"installed":true}`))
		case "/ocs/v2.php/cloud/user":
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("Unauthorized"))
		}
	}))
	defer server.Close()

	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", tempDir)

	configDir := filepath.Join(tempDir, "cospend")
	_ = os.MkdirAll(configDir, 0700)
	configContent := `{"domain": "` + server.URL + `", "user": "testuser", "password": "badpass"}`
	_ = os.WriteFile(filepath.Join(configDir, "cospend.json"), []byte(configContent), 0600)

	cmd := NewDoctorCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := stdout.String()
	if !bytes.Contains([]byte(output), []byte("[ok] Server reachable")) {
		t.Errorf("Server should be reachable, got: %s", output)
	}
	if !bytes.Contains([]byte(output), []byte("[!!] Authentication")) {
		t.Errorf("Auth should fail, got: %s", output)
	}
}

func TestDoctorNoDefaultProject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/status.php":
			_, _ = w.Write([]byte(`{"installed":true}`))
		case "/ocs/v2.php/cloud/user":
			_ = json.NewEncoder(w).Encode(makeOCSResponse(200, map[string]string{"locale": "en_US", "language": "en"}))
		}
	}))
	defer server.Close()

	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", tempDir)

	configDir := filepath.Join(tempDir, "cospend")
	_ = os.MkdirAll(configDir, 0700)
	configContent := `{"domain": "` + server.URL + `", "user": "testuser", "password": "testpass"}`
	_ = os.WriteFile(filepath.Join(configDir, "cospend.json"), []byte(configContent), 0600)

	cmd := NewDoctorCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := stdout.String()
	if !bytes.Contains([]byte(output), []byte("not configured (optional)")) {
		t.Errorf("Should show default project as optional, got: %s", output)
	}
	if !bytes.Contains([]byte(output), []byte("All checks passed")) {
		t.Errorf("Should pass with no default project, got: %s", output)
	}
}

func TestJoinWords(t *testing.T) {
	tests := []struct {
		input []string
		want  string
	}{
		{nil, ""},
		{[]string{"a"}, "a"},
		{[]string{"a", "b"}, "a and b"},
		{[]string{"a", "b", "c"}, "a, b and c"},
	}
	for _, tt := range tests {
		got := joinWords(tt.input)
		if got != tt.want {
			t.Errorf("joinWords(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
