package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInfoCommand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ocs/v2.php/cloud/user" {
			_ = json.NewEncoder(w).Encode(makeOCSResponse(200, map[string]string{
				"locale":   "he_IL",
				"language": "he",
			}))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cleanup := setupTestEnv(t, server.URL)
	defer cleanup()

	cmd := NewInfoCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := stdout.String()

	expected := []string{
		"Server:   " + server.URL,
		"User:     testuser",
		"Locale:   he_IL",
		"Language: he",
	}
	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("Output missing %q, got:\n%s", exp, output)
		}
	}
}

func TestInfoCommandNormalizesURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ocs/v2.php/cloud/user" {
			_ = json.NewEncoder(w).Encode(makeOCSResponse(200, map[string]string{
				"locale":   "en_US",
				"language": "en",
			}))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Test with trailing slash — should be stripped
	cleanup := setupTestEnv(t, server.URL+"/")
	defer cleanup()

	cmd := NewInfoCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Server:   "+server.URL) {
		t.Errorf("Expected trailing slash stripped, got:\n%s", output)
	}
	if strings.Contains(output, server.URL+"/") {
		t.Errorf("Trailing slash should be stripped, got:\n%s", output)
	}
}

func TestInfoCommandAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	cleanup := setupTestEnv(t, server.URL)
	defer cleanup()

	cmd := NewInfoCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error from API")
	}
}
