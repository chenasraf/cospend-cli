package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chenasraf/cospend-cli/internal/api"
)

func resetEditFlags() {
	ProjectID = ""
	editName = ""
	editAmount = ""
	editCategory = ""
	editPaidBy = ""
	editPaidFor = nil
	editPaymentMethod = ""
	editComment = ""
	editDate = ""
	editRepeat = ""
}

func TestNewEditCommand(t *testing.T) {
	resetEditFlags()
	defer resetEditFlags()

	cmd := NewEditCommand()

	if cmd.Use != "edit <bill_id>" {
		t.Errorf("Wrong Use: %s", cmd.Use)
	}

	if len(cmd.Aliases) != 1 || cmd.Aliases[0] != "update" {
		t.Errorf("Wrong Aliases: %v", cmd.Aliases)
	}

	flags := []string{"name", "amount", "category", "by", "for", "method", "comment", "date"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("Missing flag: %s", flag)
		}
	}

	shortFlags := map[string]string{
		"n": "name",
		"a": "amount",
		"c": "category",
		"b": "by",
		"f": "for",
		"m": "method",
		"o": "comment",
		"d": "date",
	}
	for short, long := range shortFlags {
		flag := cmd.Flags().ShorthandLookup(short)
		if flag == nil {
			t.Errorf("Missing short flag: -%s", short)
		} else if flag.Name != long {
			t.Errorf("Short flag -%s maps to %s, want %s", short, flag.Name, long)
		}
	}
}

func TestEditCommandMissingProject(t *testing.T) {
	resetEditFlags()
	defer resetEditFlags()

	cmd := NewEditCommand()
	cmd.SetArgs([]string{"123"})

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error for missing project flag")
	}
}

func TestEditCommandMissingBillID(t *testing.T) {
	resetEditFlags()
	defer resetEditFlags()

	ProjectID = "myproject"
	cmd := NewEditCommand()
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error for missing bill ID argument")
	}
}

func TestEditCommandInvalidBillID(t *testing.T) {
	resetEditFlags()
	defer resetEditFlags()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("API should not be called with invalid bill ID")
	}))
	defer server.Close()

	t.Setenv("NEXTCLOUD_DOMAIN", server.URL)
	t.Setenv("NEXTCLOUD_USER", "testuser")
	t.Setenv("NEXTCLOUD_PASSWORD", "testpass")

	ProjectID = "myproject"
	cmd := NewEditCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"not-a-number"})

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error for invalid bill ID")
	}
}

func testEditServer(t *testing.T, project api.Project, bills []api.BillResponse, onPut func(r *http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ocs/v2.php/apps/cospend/api/v1/projects/test-project" {
			_ = json.NewEncoder(w).Encode(makeOCSResponse(200, project))
			return
		}
		if r.URL.Path == "/ocs/v2.php/cloud/user" {
			_ = json.NewEncoder(w).Encode(makeOCSResponse(200, map[string]string{"locale": "en_US", "language": "en"}))
			return
		}
		if r.URL.Path == "/ocs/v2.php/apps/cospend/api/v1/projects/test-project/bills" ||
			r.URL.Path == "/ocs/v2.php/apps/cospend/api/v1/projects/test-project/bills/42" {
			if r.Method == "GET" {
				billsData := struct {
					Bills []api.BillResponse `json:"bills"`
				}{Bills: bills}
				_ = json.NewEncoder(w).Encode(makeOCSResponse(200, billsData))
				return
			}
			if r.Method == "PUT" {
				if onPut != nil {
					onPut(r)
				}
				_ = json.NewEncoder(w).Encode(makeOCSResponse(200, "OK"))
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	}))
}

func TestEditCommandSuccess(t *testing.T) {
	resetEditFlags()
	defer resetEditFlags()

	project := api.Project{
		ID:   "test-project",
		Name: "Test Project",
		Members: []api.Member{
			{ID: 1, Name: "testuser", UserID: "testuser"},
			{ID: 2, Name: "Alice", UserID: "alice"},
			{ID: 3, Name: "Bob", UserID: "bob"},
		},
		Categories: []api.Category{
			{ID: 5, Name: "Restaurant"},
		},
		PaymentModes: []api.PaymentMode{
			{ID: 3, Name: "Credit Card"},
		},
	}

	bills := []api.BillResponse{
		{
			ID:      42,
			What:    "Old Dinner",
			Amount:  30.00,
			Date:    "2026-01-15",
			PayerID: 1,
			Owers:   []api.Ower{{ID: 1, Weight: 1}, {ID: 2, Weight: 1}},
			Comment: "old comment",
		},
	}

	var receivedBill map[string]string
	server := testEditServer(t, project, bills, func(r *http.Request) {
		_ = r.ParseForm()
		receivedBill = make(map[string]string)
		for k, v := range r.Form {
			if len(v) > 0 {
				receivedBill[k] = v[0]
			}
		}
	})
	defer server.Close()

	cleanup := setupTestEnv(t, server.URL)
	defer cleanup()

	ProjectID = "test-project"
	cmd := NewEditCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{
		"42",
		"-n", "Updated Dinner",
		"-a", "50.00",
		"-b", "alice",
		"-f", "bob",
		"-c", "restaurant",
		"-m", "credit card",
		"-o", "new comment",
		"-d", "2026-06-15",
	})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if receivedBill["what"] != "Updated Dinner" {
		t.Errorf("Wrong what: %s", receivedBill["what"])
	}
	if receivedBill["amount"] != "50.00" {
		t.Errorf("Wrong amount: %s", receivedBill["amount"])
	}
	if receivedBill["payer"] != "2" { // Alice's ID
		t.Errorf("Wrong payer: %s", receivedBill["payer"])
	}
	if receivedBill["payedFor"] != "3" { // Bob only
		t.Errorf("Wrong payedFor: %s", receivedBill["payedFor"])
	}
	if receivedBill["categoryId"] != "5" {
		t.Errorf("Wrong categoryId: %s", receivedBill["categoryId"])
	}
	if receivedBill["paymentModeId"] != "3" {
		t.Errorf("Wrong paymentModeId: %s", receivedBill["paymentModeId"])
	}
	if receivedBill["comment"] != "new comment" {
		t.Errorf("Wrong comment: %s", receivedBill["comment"])
	}
	if receivedBill["date"] != "2026-06-15" {
		t.Errorf("Wrong date: %s", receivedBill["date"])
	}

	if !bytes.Contains(stdout.Bytes(), []byte("Updated bill #42")) {
		t.Errorf("Missing success message in output: %s", stdout.String())
	}
}

func TestEditCommandPartialUpdate(t *testing.T) {
	resetEditFlags()
	defer resetEditFlags()

	project := api.Project{
		ID:   "test-project",
		Name: "Test Project",
		Members: []api.Member{
			{ID: 1, Name: "testuser", UserID: "testuser"},
			{ID: 2, Name: "Alice", UserID: "alice"},
		},
	}

	bills := []api.BillResponse{
		{
			ID:      42,
			What:    "Original Name",
			Amount:  25.00,
			Date:    "2026-01-15",
			PayerID: 1,
			Owers:   []api.Ower{{ID: 1, Weight: 1}, {ID: 2, Weight: 1}},
			Comment: "original comment",
		},
	}

	var receivedBill map[string]string
	server := testEditServer(t, project, bills, func(r *http.Request) {
		_ = r.ParseForm()
		receivedBill = make(map[string]string)
		for k, v := range r.Form {
			if len(v) > 0 {
				receivedBill[k] = v[0]
			}
		}
	})
	defer server.Close()

	cleanup := setupTestEnv(t, server.URL)
	defer cleanup()

	ProjectID = "test-project"
	cmd := NewEditCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	// Only change the name
	cmd.SetArgs([]string{"42", "-n", "New Name"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Name should be updated
	if receivedBill["what"] != "New Name" {
		t.Errorf("Wrong what: %s", receivedBill["what"])
	}
	// Other fields should be preserved from the original bill
	if receivedBill["amount"] != "25.00" {
		t.Errorf("Amount should be preserved: got %s, want 25.00", receivedBill["amount"])
	}
	if receivedBill["payer"] != "1" {
		t.Errorf("Payer should be preserved: got %s, want 1", receivedBill["payer"])
	}
	if receivedBill["payedFor"] != "1,2" {
		t.Errorf("PayedFor should be preserved: got %s, want 1,2", receivedBill["payedFor"])
	}
	if receivedBill["date"] != "2026-01-15" {
		t.Errorf("Date should be preserved: got %s, want 2026-01-15", receivedBill["date"])
	}
	if receivedBill["comment"] != "original comment" {
		t.Errorf("Comment should be preserved: got %s, want 'original comment'", receivedBill["comment"])
	}
}

func TestEditCommandBillNotFound(t *testing.T) {
	resetEditFlags()
	defer resetEditFlags()

	project := api.Project{
		ID:   "test-project",
		Name: "Test Project",
		Members: []api.Member{
			{ID: 1, Name: "testuser", UserID: "testuser"},
		},
	}

	// Empty bills list
	server := testEditServer(t, project, []api.BillResponse{}, nil)
	defer server.Close()

	cleanup := setupTestEnv(t, server.URL)
	defer cleanup()

	ProjectID = "test-project"
	cmd := NewEditCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"999", "-n", "whatever"})

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error for bill not found")
	}
}

func TestEditCommandMemberNotFound(t *testing.T) {
	resetEditFlags()
	defer resetEditFlags()

	project := api.Project{
		ID:   "test-project",
		Name: "Test Project",
		Members: []api.Member{
			{ID: 1, Name: "testuser", UserID: "testuser"},
		},
	}

	bills := []api.BillResponse{
		{
			ID:      42,
			What:    "Test",
			Amount:  10.00,
			Date:    "2026-01-15",
			PayerID: 1,
			Owers:   []api.Ower{{ID: 1, Weight: 1}},
		},
	}

	server := testEditServer(t, project, bills, nil)
	defer server.Close()

	cleanup := setupTestEnv(t, server.URL)
	defer cleanup()

	ProjectID = "test-project"
	cmd := NewEditCommand()
	cmd.SetArgs([]string{"42", "-b", "nonexistent"})

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error for nonexistent member")
	}
}

func TestEditCommandAPIError(t *testing.T) {
	resetEditFlags()
	defer resetEditFlags()

	project := api.Project{
		ID:   "test-project",
		Name: "Test Project",
		Members: []api.Member{
			{ID: 1, Name: "testuser", UserID: "testuser"},
		},
	}

	bills := []api.BillResponse{
		{
			ID:      42,
			What:    "Test",
			Amount:  10.00,
			Date:    "2026-01-15",
			PayerID: 1,
			Owers:   []api.Ower{{ID: 1, Weight: 1}},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ocs/v2.php/apps/cospend/api/v1/projects/test-project" {
			_ = json.NewEncoder(w).Encode(makeOCSResponse(200, project))
			return
		}
		if r.URL.Path == "/ocs/v2.php/cloud/user" {
			_ = json.NewEncoder(w).Encode(makeOCSResponse(200, map[string]string{"locale": "en_US", "language": "en"}))
			return
		}
		if r.Method == "GET" && r.URL.Path == "/ocs/v2.php/apps/cospend/api/v1/projects/test-project/bills" {
			billsData := struct {
				Bills []api.BillResponse `json:"bills"`
			}{Bills: bills}
			_ = json.NewEncoder(w).Encode(makeOCSResponse(200, billsData))
			return
		}
		// Return error for PUT
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	cleanup := setupTestEnv(t, server.URL)
	defer cleanup()

	ProjectID = "test-project"
	cmd := NewEditCommand()
	cmd.SetArgs([]string{"42", "-n", "New Name"})

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error from API")
	}
}
