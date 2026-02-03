package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewDeleteCommand(t *testing.T) {
	cmd := NewDeleteCommand()

	if cmd.Use != "delete <bill_id>" {
		t.Errorf("Use = %v, want %v", cmd.Use, "delete <bill_id>")
	}
}

func TestDeleteCommandMissingProject(t *testing.T) {
	resetDeleteFlags()

	cmd := NewDeleteCommand()
	cmd.SetArgs([]string{"123"})

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error for missing project flag")
	}
}

func TestDeleteCommandMissingBillID(t *testing.T) {
	resetDeleteFlags()

	ProjectID = "myproject"
	cmd := NewDeleteCommand()
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error for missing bill ID argument")
	}
}

func TestDeleteCommandInvalidBillID(t *testing.T) {
	resetDeleteFlags()

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("API should not be called with invalid bill ID")
	}))
	defer server.Close()

	t.Setenv("NEXTCLOUD_DOMAIN", server.URL)
	t.Setenv("NEXTCLOUD_USER", "testuser")
	t.Setenv("NEXTCLOUD_PASSWORD", "testpass")

	ProjectID = "myproject"
	cmd := NewDeleteCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"not-a-number"})

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error for invalid bill ID")
	}
}

func TestDeleteCommandSuccess(t *testing.T) {
	resetDeleteFlags()

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE method, got %s", r.Method)
		}
		if r.URL.Path != "/ocs/v2.php/apps/cospend/api/v1/projects/myproject/bills/123" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"ocs": map[string]interface{}{
				"meta": map[string]interface{}{
					"status":     "ok",
					"statuscode": 200,
					"message":    "OK",
				},
				"data": "OK",
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	t.Setenv("NEXTCLOUD_DOMAIN", server.URL)
	t.Setenv("NEXTCLOUD_USER", "testuser")
	t.Setenv("NEXTCLOUD_PASSWORD", "testpass")

	ProjectID = "myproject"
	cmd := NewDeleteCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"123"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Successfully deleted bill #123")) {
		t.Errorf("Expected success message in output, got: %s", output)
	}
}

func TestDeleteCommandAPIError(t *testing.T) {
	resetDeleteFlags()

	// Create mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"ocs": map[string]interface{}{
				"meta": map[string]interface{}{
					"status":     "failure",
					"statuscode": 404,
					"message":    "Bill not found",
				},
				"data": nil,
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	t.Setenv("NEXTCLOUD_DOMAIN", server.URL)
	t.Setenv("NEXTCLOUD_USER", "testuser")
	t.Setenv("NEXTCLOUD_PASSWORD", "testpass")

	ProjectID = "myproject"
	cmd := NewDeleteCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"999"})

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error for API failure")
	}
}

func resetDeleteFlags() {
	ProjectID = ""
}
