package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chenasraf/cospend-cli/internal/api"
)

func TestResolveMember(t *testing.T) {
	project := &api.Project{
		Members: []api.Member{
			{ID: 1, Name: "Alice", UserID: "alice"},
			{ID: 2, Name: "Bob", UserID: "bob"},
			{ID: 3, Name: "Charlie", UserID: "charlie123"},
		},
	}

	tests := []struct {
		name     string
		username string
		wantID   int
		wantErr  bool
	}{
		{"by name exact", "Alice", 1, false},
		{"by name lowercase", "alice", 1, false},
		{"by name uppercase", "ALICE", 1, false},
		{"by name mixed case", "aLiCe", 1, false},
		{"by userid", "bob", 2, false},
		{"by userid different from name", "charlie123", 3, false},
		{"by name when userid differs", "Charlie", 3, false},
		{"not found", "unknown", 0, true},
		{"empty string", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, err := ResolveMember(project, tt.username)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveMember() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotID != tt.wantID {
				t.Errorf("ResolveMember() = %v, want %v", gotID, tt.wantID)
			}
		})
	}
}

func TestResolveCategory(t *testing.T) {
	project := &api.Project{
		Categories: []api.Category{
			{ID: 1, Name: "Groceries"},
			{ID: 2, Name: "Restaurant"},
			{ID: 10, Name: "Transport"},
		},
	}

	tests := []struct {
		name     string
		nameOrID string
		wantID   int
		wantErr  bool
	}{
		{"by id", "1", 1, false},
		{"by id second", "2", 2, false},
		{"by id double digit", "10", 10, false},
		{"by name exact", "Groceries", 1, false},
		{"by name lowercase", "groceries", 1, false},
		{"by name uppercase", "RESTAURANT", 2, false},
		{"by name mixed case", "tRaNsPoRt", 10, false},
		{"id not found", "99", 0, true},
		{"name not found", "Unknown", 0, true},
		{"empty string", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, err := ResolveCategory(project, tt.nameOrID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveCategory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotID != tt.wantID {
				t.Errorf("ResolveCategory() = %v, want %v", gotID, tt.wantID)
			}
		})
	}
}

func TestResolvePaymentMode(t *testing.T) {
	project := &api.Project{
		PaymentModes: []api.PaymentMode{
			{ID: 1, Name: "Cash"},
			{ID: 2, Name: "Credit Card"},
			{ID: 3, Name: "Bank Transfer"},
		},
	}

	tests := []struct {
		name     string
		nameOrID string
		wantID   int
		wantErr  bool
	}{
		{"by id", "1", 1, false},
		{"by id second", "2", 2, false},
		{"by name exact", "Cash", 1, false},
		{"by name lowercase", "cash", 1, false},
		{"by name with space", "Credit Card", 2, false},
		{"by name with space lowercase", "credit card", 2, false},
		{"by name uppercase", "BANK TRANSFER", 3, false},
		{"id not found", "99", 0, true},
		{"name not found", "Bitcoin", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, err := ResolvePaymentMode(project, tt.nameOrID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolvePaymentMode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotID != tt.wantID {
				t.Errorf("ResolvePaymentMode() = %v, want %v", gotID, tt.wantID)
			}
		})
	}
}

func TestResolveCurrency(t *testing.T) {
	project := &api.Project{
		Currencies: []api.Currency{
			{ID: 1, Name: "$", ExchangeRate: 1.0},
			{ID: 2, Name: "€", ExchangeRate: 0.85},
			{ID: 3, Name: "£", ExchangeRate: 0.73},
			{ID: 4, Name: "US Dollar ($)", ExchangeRate: 1.0},
			{ID: 5, Name: "Japanese Yen (¥)", ExchangeRate: 110.0},
		},
	}

	tests := []struct {
		name     string
		nameOrID string
		wantID   int
		wantErr  bool
	}{
		{"by id", "1", 1, false},
		{"by id second", "2", 2, false},
		{"by name exact symbol", "$", 1, false},
		{"by name exact euro", "€", 2, false},
		{"by name with description", "US Dollar ($)", 4, false},
		{"by currency code usd", "usd", 1, false},
		{"by currency code USD uppercase", "USD", 1, false},
		{"by currency code eur", "eur", 2, false},
		{"by currency code gbp", "gbp", 3, false},
		{"by currency code jpy", "jpy", 5, false},
		{"id not found", "99", 0, true},
		{"name not found", "Bitcoin", 0, true},
		{"unknown currency code", "xyz", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, err := ResolveCurrency(project, tt.nameOrID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveCurrency() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotID != tt.wantID {
				t.Errorf("ResolveCurrency() = %v, want %v", gotID, tt.wantID)
			}
		})
	}
}

func TestCurrencyCodeToSymbolMapping(t *testing.T) {
	// Test that common currency codes are mapped
	expectedMappings := map[string]string{
		"usd": "$",
		"eur": "€",
		"gbp": "£",
		"jpy": "¥",
		"cny": "¥",
		"inr": "₹",
		"krw": "₩",
		"brl": "R$",
	}

	for code, expectedSymbol := range expectedMappings {
		if symbol, ok := currencyCodeToSymbol[code]; !ok {
			t.Errorf("Currency code %q not found in mapping", code)
		} else if symbol != expectedSymbol {
			t.Errorf("Currency code %q maps to %q, want %q", code, symbol, expectedSymbol)
		}
	}
}

func TestSaveAndLoad(t *testing.T) {
	// Use a temp directory for testing
	tempDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tempDir)

	project := &api.Project{
		ID:   "test-project",
		Name: "Test Project",
		Members: []api.Member{
			{ID: 1, Name: "Alice", UserID: "alice"},
		},
		Categories: []api.Category{
			{ID: 1, Name: "Food"},
		},
		PaymentModes: []api.PaymentMode{
			{ID: 1, Name: "Cash"},
		},
		Currencies: []api.Currency{
			{ID: 1, Name: "$", ExchangeRate: 1.0},
		},
	}

	// Test Save
	err := Save("test-project", project)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file exists
	cachePath := filepath.Join(tempDir, "cospend", "test-project.json")
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Errorf("Cache file not created at %s", cachePath)
	}

	// Test Load
	loaded, ok := Load("test-project")
	if !ok {
		t.Fatal("Load() returned false, expected true")
	}

	if loaded.ID != project.ID {
		t.Errorf("Load() ID = %v, want %v", loaded.ID, project.ID)
	}
	if loaded.Name != project.Name {
		t.Errorf("Load() Name = %v, want %v", loaded.Name, project.Name)
	}
	if len(loaded.Members) != len(project.Members) {
		t.Errorf("Load() Members count = %v, want %v", len(loaded.Members), len(project.Members))
	}
}

func TestLoadNonExistent(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tempDir)

	_, ok := Load("non-existent-project")
	if ok {
		t.Error("Load() returned true for non-existent project, expected false")
	}
}

func TestLoadExpired(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tempDir)

	project := &api.Project{
		ID:   "expired-project",
		Name: "Expired Project",
	}

	// Save the project
	err := Save("expired-project", project)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Modify the cache file to have an old timestamp
	cachePath := filepath.Join(tempDir, "cospend", "expired-project.json")
	oldTime := time.Now().Add(-2 * time.Hour) // 2 hours ago, TTL is 1 hour
	_ = os.Chtimes(cachePath, oldTime, oldTime)

	// Manually update the cached_at field in the file
	// Replace the timestamp in the JSON (crude but works for testing)
	oldTimestamp := time.Now().Add(-2 * time.Hour).Format(time.RFC3339Nano)
	newData := []byte(`{"project":{"id":"expired-project","name":"Expired Project","members":null,"categories":null,"paymentmodes":null,"currencies":null},"cached_at":"` + oldTimestamp + `"}`)
	_ = os.WriteFile(cachePath, newData, 0644)

	_, ok := Load("expired-project")
	if ok {
		t.Error("Load() returned true for expired cache, expected false")
	}
}
