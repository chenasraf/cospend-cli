package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chenasraf/cospend-cli/internal/api"
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

func TestInfoCommandWithProject(t *testing.T) {
	project := api.Project{
		ID:           "test-project",
		Name:         "Test Project",
		CurrencyName: "EUR",
		Members: []api.Member{
			{ID: 1, Name: "Alice", UserID: "alice"},
			{ID: 2, Name: "Bob", UserID: "bob"},
		},
		Categories: []api.Category{
			{ID: 5, Name: "Food", Icon: "\U0001F354", Color: "#ff0000"},
			{ID: 12, Name: "Transport", Icon: "\U0001F697", Color: "#00ff00"},
		},
		PaymentModes: []api.PaymentMode{
			{ID: 3, Name: "Credit Card", Icon: "\U0001F4B3", Color: "#0000ff"},
		},
		Currencies: []api.Currency{
			{ID: 1, Name: "USD", ExchangeRate: 1.1},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ocs/v2.php/cloud/user" {
			_ = json.NewEncoder(w).Encode(makeOCSResponse(200, map[string]string{
				"locale":   "en_US",
				"language": "en",
			}))
			return
		}
		if r.URL.Path == "/ocs/v2.php/apps/cospend/api/v1/projects/test-project" {
			_ = json.NewEncoder(w).Encode(makeOCSResponse(200, project))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cleanup := setupTestEnv(t, server.URL)
	defer cleanup()

	ProjectID = "test-project"
	cmd := NewInfoCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := stdout.String()

	expected := []string{
		"Project:  Test Project",
		"Currency: EUR",
		"Members:",
		"Alice",
		"Bob",
		"Categories:",
		"Food",
		"Transport",
		"Payment Modes:",
		"Credit Card",
		"Currencies:",
		"USD",
		"1.1",
	}
	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("Output missing %q, got:\n%s", exp, output)
		}
	}
}
