package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chenasraf/cospend-cli/internal/api"
)

// OCSResponse for test responses
type ocsResponse struct {
	OCS struct {
		Meta struct {
			Status     string `json:"status"`
			StatusCode int    `json:"statuscode"`
			Message    string `json:"message"`
		} `json:"meta"`
		Data json.RawMessage `json:"data"`
	} `json:"ocs"`
}

func makeOCSResponse(statusCode int, data any) ocsResponse {
	dataBytes, _ := json.Marshal(data)
	resp := ocsResponse{}
	resp.OCS.Meta.Status = "ok"
	resp.OCS.Meta.StatusCode = statusCode
	resp.OCS.Meta.Message = "OK"
	resp.OCS.Data = dataBytes
	return resp
}

func resetFlags() {
	// Reset global flag variables between tests
	ProjectID = ""
	category = ""
	paidBy = ""
	paidFor = nil
	convertTo = ""
	paymentMethod = ""
	comment = ""
	addDate = ""
	infoCached = false
}

func setupTestEnv(t *testing.T, domain string) func() {
	t.Helper()

	// Reset flags
	resetFlags()

	// Set test env vars (t.Setenv auto-restores after test)
	t.Setenv("NEXTCLOUD_DOMAIN", domain)
	t.Setenv("NEXTCLOUD_USER", "testuser")
	t.Setenv("NEXTCLOUD_PASSWORD", "testpass")
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	return func() {
		resetFlags()
	}
}

func TestNewAddCommand(t *testing.T) {
	resetFlags()
	defer resetFlags()

	cmd := NewAddCommand()

	if cmd.Use != "add <name> <amount>" {
		t.Errorf("Wrong Use: %s", cmd.Use)
	}

	// Check flags exist (project is now a persistent flag on root)
	flags := []string{"category", "by", "for", "convert", "method", "comment", "date"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("Missing flag: %s", flag)
		}
	}

	// Check short flags (project is now on root)
	shortFlags := map[string]string{
		"c": "category",
		"b": "by",
		"f": "for",
		"C": "convert",
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

func TestAddCommandMissingProject(t *testing.T) {
	resetFlags()
	defer resetFlags()

	cmd := NewAddCommand()
	cmd.SetArgs([]string{"Test expense", "10.00"})

	var stderr bytes.Buffer
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error for missing project flag")
	}
}

func TestAddCommandInvalidAmount(t *testing.T) {
	project := api.Project{
		ID:   "test-project",
		Name: "Test",
		Members: []api.Member{
			{ID: 1, Name: "testuser", UserID: "testuser"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(makeOCSResponse(200, project))
	}))
	defer server.Close()

	cleanup := setupTestEnv(t, server.URL)
	defer cleanup()

	ProjectID = "test-project"
	cmd := NewAddCommand()
	cmd.SetArgs([]string{"Test expense", "not-a-number"})

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error for invalid amount")
	}
}

func TestAddCommandSuccess(t *testing.T) {
	project := api.Project{
		ID:   "test-project",
		Name: "Test Project",
		Members: []api.Member{
			{ID: 1, Name: "testuser", UserID: "testuser"},
			{ID: 2, Name: "Alice", UserID: "alice"},
		},
		Categories: []api.Category{
			{ID: 1, Name: "Food"},
		},
		PaymentModes: []api.PaymentMode{
			{ID: 1, Name: "Cash"},
		},
	}

	var receivedBill map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ocs/v2.php/apps/cospend/api/v1/projects/test-project" {
			_ = json.NewEncoder(w).Encode(makeOCSResponse(200, project))
			return
		}

		if r.URL.Path == "/ocs/v2.php/cloud/user" {
			_ = json.NewEncoder(w).Encode(makeOCSResponse(200, map[string]string{"locale": "en_US", "language": "en"}))
			return
		}

		if r.URL.Path == "/ocs/v2.php/apps/cospend/api/v1/projects/test-project/bills" {
			_ = r.ParseForm()
			receivedBill = make(map[string]string)
			for k, v := range r.Form {
				if len(v) > 0 {
					receivedBill[k] = v[0]
				}
			}
			_ = json.NewEncoder(w).Encode(makeOCSResponse(200, map[string]int{"id": 1}))
			return
		}
	}))
	defer server.Close()

	cleanup := setupTestEnv(t, server.URL)
	defer cleanup()

	ProjectID = "test-project"
	cmd := NewAddCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"Groceries", "25.50"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify bill data
	if receivedBill["what"] != "Groceries" {
		t.Errorf("Wrong what: %s", receivedBill["what"])
	}
	if receivedBill["amount"] != "25.50" {
		t.Errorf("Wrong amount: %s", receivedBill["amount"])
	}
	if receivedBill["payer"] != "1" {
		t.Errorf("Wrong payer: %s", receivedBill["payer"])
	}
	// Default owed to payer
	if receivedBill["payedFor"] != "1" {
		t.Errorf("Wrong payedFor: %s", receivedBill["payedFor"])
	}

	// Check output
	if !bytes.Contains(stdout.Bytes(), []byte("Added expense")) {
		t.Errorf("Missing success message in output: %s", stdout.String())
	}
}

func TestAddCommandWithAllFlags(t *testing.T) {
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
		Currencies: []api.Currency{
			{ID: 2, Name: "€", ExchangeRate: 0.85},
		},
	}

	var receivedBill map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ocs/v2.php/apps/cospend/api/v1/projects/test-project" {
			_ = json.NewEncoder(w).Encode(makeOCSResponse(200, project))
			return
		}

		if r.URL.Path == "/ocs/v2.php/cloud/user" {
			_ = json.NewEncoder(w).Encode(makeOCSResponse(200, map[string]string{"locale": "en_US", "language": "en"}))
			return
		}

		if r.URL.Path == "/ocs/v2.php/apps/cospend/api/v1/projects/test-project/bills" {
			_ = r.ParseForm()
			receivedBill = make(map[string]string)
			for k, v := range r.Form {
				if len(v) > 0 {
					receivedBill[k] = v[0]
				}
			}
			_ = json.NewEncoder(w).Encode(makeOCSResponse(200, map[string]int{"id": 1}))
			return
		}
	}))
	defer server.Close()

	cleanup := setupTestEnv(t, server.URL)
	defer cleanup()

	ProjectID = "test-project"
	cmd := NewAddCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{
		"Dinner",
		"45.00",
		"-c", "restaurant",
		"-b", "alice",
		"-f", "alice",
		"-f", "bob",
		"-m", "credit card",
		"-o", "Team dinner",
		"-C", "eur",
	})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify bill data
	if receivedBill["what"] != "Dinner (€ 45.00)" {
		t.Errorf("Wrong what: %s", receivedBill["what"])
	}
	if receivedBill["amount"] != "38.25" { // 45.00 * 0.85 exchange rate
		t.Errorf("Wrong amount: got %s, want 38.25 (45.00 * 0.85)", receivedBill["amount"])
	}
	if receivedBill["payer"] != "2" { // Alice's ID
		t.Errorf("Wrong payer: %s", receivedBill["payer"])
	}
	if receivedBill["payedFor"] != "2,3" { // Alice and Bob
		t.Errorf("Wrong payedFor: %s", receivedBill["payedFor"])
	}
	if receivedBill["categoryId"] != "5" {
		t.Errorf("Wrong categoryid: %s", receivedBill["categoryId"])
	}
	if receivedBill["paymentModeId"] != "3" {
		t.Errorf("Wrong paymentmodeid: %s", receivedBill["paymentModeId"])
	}
	if receivedBill["comment"] != "Team dinner" {
		t.Errorf("Wrong comment: %s", receivedBill["comment"])
	}
	if receivedBill["original_currency_id"] != "2" {
		t.Errorf("Wrong original_currency_id: %s", receivedBill["original_currency_id"])
	}
}

func TestAddCommandMemberNotFound(t *testing.T) {
	project := api.Project{
		ID:   "test-project",
		Name: "Test Project",
		Members: []api.Member{
			{ID: 1, Name: "testuser", UserID: "testuser"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(makeOCSResponse(200, project))
	}))
	defer server.Close()

	cleanup := setupTestEnv(t, server.URL)
	defer cleanup()

	ProjectID = "test-project"
	cmd := NewAddCommand()
	cmd.SetArgs([]string{"Test", "10.00", "-b", "nonexistent"})

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error for nonexistent member")
	}
}

func TestAddCommandCategoryNotFound(t *testing.T) {
	project := api.Project{
		ID:   "test-project",
		Name: "Test Project",
		Members: []api.Member{
			{ID: 1, Name: "testuser", UserID: "testuser"},
		},
		Categories: []api.Category{
			{ID: 1, Name: "Food"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(makeOCSResponse(200, project))
	}))
	defer server.Close()

	cleanup := setupTestEnv(t, server.URL)
	defer cleanup()

	ProjectID = "test-project"
	cmd := NewAddCommand()
	cmd.SetArgs([]string{"Test", "10.00", "-c", "nonexistent"})

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error for nonexistent category")
	}
}

func TestAddCommandPaymentModeNotFound(t *testing.T) {
	project := api.Project{
		ID:   "test-project",
		Name: "Test Project",
		Members: []api.Member{
			{ID: 1, Name: "testuser", UserID: "testuser"},
		},
		PaymentModes: []api.PaymentMode{
			{ID: 1, Name: "Cash"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(makeOCSResponse(200, project))
	}))
	defer server.Close()

	cleanup := setupTestEnv(t, server.URL)
	defer cleanup()

	ProjectID = "test-project"
	cmd := NewAddCommand()
	cmd.SetArgs([]string{"Test", "10.00", "-m", "bitcoin"})

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error for nonexistent payment mode")
	}
}

func TestAddCommandCurrencyNotFound(t *testing.T) {
	project := api.Project{
		ID:   "test-project",
		Name: "Test Project",
		Members: []api.Member{
			{ID: 1, Name: "testuser", UserID: "testuser"},
		},
		Currencies: []api.Currency{
			{ID: 1, Name: "$"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(makeOCSResponse(200, project))
	}))
	defer server.Close()

	cleanup := setupTestEnv(t, server.URL)
	defer cleanup()

	ProjectID = "test-project"
	cmd := NewAddCommand()
	cmd.SetArgs([]string{"Test", "10.00", "-C", "btc"})

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error for nonexistent currency")
	}
}

func TestAddCommandMissingEnvVars(t *testing.T) {
	resetFlags()
	defer resetFlags()

	// Clear all env vars using t.Setenv (restores automatically)
	t.Setenv("NEXTCLOUD_DOMAIN", "")
	t.Setenv("NEXTCLOUD_USER", "")
	t.Setenv("NEXTCLOUD_PASSWORD", "")

	ProjectID = "test-project"
	cmd := NewAddCommand()
	cmd.SetArgs([]string{"Test", "10.00"})

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error for missing env vars")
	}
}

func TestAddCommandAPIError(t *testing.T) {
	project := api.Project{
		ID:   "test-project",
		Name: "Test Project",
		Members: []api.Member{
			{ID: 1, Name: "testuser", UserID: "testuser"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ocs/v2.php/apps/cospend/api/v1/projects/test-project" {
			_ = json.NewEncoder(w).Encode(makeOCSResponse(200, project))
			return
		}

		// Return error for bill creation
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	cleanup := setupTestEnv(t, server.URL)
	defer cleanup()

	ProjectID = "test-project"
	cmd := NewAddCommand()
	cmd.SetArgs([]string{"Test", "10.00"})

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error from API")
	}
}

func TestParseDate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantDate string
		wantErr  bool
	}{
		{"full date", "2026-03-15", "2026-03-15", false},
		{"short date", "03-15", fmt.Sprintf("%d-03-15", time.Now().Year()), false},
		{"with spaces", " 2026-01-01 ", "2026-01-01", false},
		{"relative -1d", "-1d", time.Now().AddDate(0, 0, -1).Format("2006-01-02"), false},
		{"relative +2d", "+2d", time.Now().AddDate(0, 0, 2).Format("2006-01-02"), false},
		{"relative -1w", "-1w", time.Now().AddDate(0, 0, -7).Format("2006-01-02"), false},
		{"relative +2w", "+2w", time.Now().AddDate(0, 0, 14).Format("2006-01-02"), false},
		{"relative -1m", "-1m", time.Now().AddDate(0, -1, 0).Format("2006-01-02"), false},
		{"relative +3m", "+3m", time.Now().AddDate(0, 3, 0).Format("2006-01-02"), false},
		{"invalid", "not-a-date", "", true},
		{"invalid short", "13-40", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.wantDate {
				t.Errorf("parseDate() = %v, want %v", got, tt.wantDate)
			}
		})
	}
}

func TestAddCommandWithDate(t *testing.T) {
	project := api.Project{
		ID:   "test-project",
		Name: "Test Project",
		Members: []api.Member{
			{ID: 1, Name: "testuser", UserID: "testuser"},
		},
	}

	var receivedBill map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ocs/v2.php/apps/cospend/api/v1/projects/test-project" {
			_ = json.NewEncoder(w).Encode(makeOCSResponse(200, project))
			return
		}
		if r.URL.Path == "/ocs/v2.php/cloud/user" {
			_ = json.NewEncoder(w).Encode(makeOCSResponse(200, map[string]string{"locale": "en_US", "language": "en"}))
			return
		}
		if r.URL.Path == "/ocs/v2.php/apps/cospend/api/v1/projects/test-project/bills" {
			_ = r.ParseForm()
			receivedBill = make(map[string]string)
			for k, v := range r.Form {
				if len(v) > 0 {
					receivedBill[k] = v[0]
				}
			}
			_ = json.NewEncoder(w).Encode(makeOCSResponse(200, map[string]int{"id": 1}))
			return
		}
	}))
	defer server.Close()

	cleanup := setupTestEnv(t, server.URL)
	defer cleanup()

	ProjectID = "test-project"
	cmd := NewAddCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"Groceries", "25.50", "-d", "2026-06-15"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if receivedBill["date"] != "2026-06-15" {
		t.Errorf("Wrong date: got %s, want 2026-06-15", receivedBill["date"])
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Date:     2026-06-15")) {
		t.Errorf("Output should show date, got:\n%s", stdout.String())
	}
}
