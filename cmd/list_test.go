package cmd

import (
	"bytes"
	"testing"

	"github.com/chenasraf/cospend-cli/internal/api"
	"github.com/chenasraf/cospend-cli/internal/format"
)

func TestParseAmountFilter(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantOp  string
		wantVal float64
		wantErr bool
	}{
		{"plain number", "50", "=", 50, false},
		{"equals", "=25", "=", 25, false},
		{"greater than", ">30", ">", 30, false},
		{"less than", "<100", "<", 100, false},
		{"greater or equal", ">=50", ">=", 50, false},
		{"less or equal", "<=75.5", "<=", 75.5, false},
		{"with spaces", " >= 100 ", ">=", 100, false},
		{"decimal", "25.99", "=", 25.99, false},
		{"invalid number", ">abc", "", 0, true},
		{"empty string", "", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			af, err := parseAmountFilter(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseAmountFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if af.operator != tt.wantOp {
					t.Errorf("parseAmountFilter() operator = %v, want %v", af.operator, tt.wantOp)
				}
				if af.value != tt.wantVal {
					t.Errorf("parseAmountFilter() value = %v, want %v", af.value, tt.wantVal)
				}
			}
		})
	}
}

func TestMatchAmount(t *testing.T) {
	tests := []struct {
		name   string
		amount float64
		filter amountFilter
		want   bool
	}{
		{"equals match", 50, amountFilter{"=", 50}, true},
		{"equals no match", 50, amountFilter{"=", 51}, false},
		{"greater match", 60, amountFilter{">", 50}, true},
		{"greater no match", 50, amountFilter{">", 50}, false},
		{"greater edge", 50, amountFilter{">", 49.99}, true},
		{"less match", 40, amountFilter{"<", 50}, true},
		{"less no match", 50, amountFilter{"<", 50}, false},
		{"greater equal match exact", 50, amountFilter{">=", 50}, true},
		{"greater equal match above", 51, amountFilter{">=", 50}, true},
		{"greater equal no match", 49, amountFilter{">=", 50}, false},
		{"less equal match exact", 50, amountFilter{"<=", 50}, true},
		{"less equal match below", 49, amountFilter{"<=", 50}, true},
		{"less equal no match", 51, amountFilter{"<=", 50}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchAmount(tt.amount, tt.filter); got != tt.want {
				t.Errorf("matchAmount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApplyFilters(t *testing.T) {
	bills := []api.BillResponse{
		{ID: 1, What: "Groceries", Amount: 50, PayerID: 1, CategoryID: 1},
		{ID: 2, What: "Dinner", Amount: 100, PayerID: 2, CategoryID: 2},
		{ID: 3, What: "Lunch", Amount: 25, PayerID: 1, CategoryID: 2},
		{ID: 4, What: "Coffee", Amount: 5, PayerID: 3, CategoryID: 1},
	}

	t.Run("no filters", func(t *testing.T) {
		result := applyFilters(bills, nil)
		if len(result) != 4 {
			t.Errorf("applyFilters() returned %d bills, want 4", len(result))
		}
	})

	t.Run("single filter", func(t *testing.T) {
		filters := []billFilter{
			func(b api.BillResponse) bool { return b.PayerID == 1 },
		}
		result := applyFilters(bills, filters)
		if len(result) != 2 {
			t.Errorf("applyFilters() returned %d bills, want 2", len(result))
		}
	})

	t.Run("multiple filters AND", func(t *testing.T) {
		filters := []billFilter{
			func(b api.BillResponse) bool { return b.PayerID == 1 },
			func(b api.BillResponse) bool { return b.Amount > 30 },
		}
		result := applyFilters(bills, filters)
		if len(result) != 1 {
			t.Errorf("applyFilters() returned %d bills, want 1", len(result))
		}
		if result[0].ID != 1 {
			t.Errorf("applyFilters() returned bill ID %d, want 1", result[0].ID)
		}
	})

	t.Run("filter with no matches", func(t *testing.T) {
		filters := []billFilter{
			func(b api.BillResponse) bool { return b.Amount > 1000 },
		}
		result := applyFilters(bills, filters)
		if len(result) != 0 {
			t.Errorf("applyFilters() returned %d bills, want 0", len(result))
		}
	})
}

func TestPrintBillsTable(t *testing.T) {
	// Reset global flags
	resetListFlags()

	project := &api.Project{
		Members: []api.Member{
			{ID: 1, Name: "Alice", UserID: "alice"},
			{ID: 2, Name: "Bob", UserID: "bob"},
		},
		Categories: []api.Category{
			{ID: 1, Name: "Food"},
			{ID: 2, Name: "Transport"},
		},
		PaymentModes: []api.PaymentMode{
			{ID: 1, Name: "Cash"},
			{ID: 2, Name: "Card"},
		},
	}

	bills := []api.BillResponse{
		{
			ID:            1,
			What:          "Groceries",
			Amount:        50.00,
			Date:          "2026-02-03",
			PayerID:       1,
			Owers:         []api.Ower{{ID: 1, Weight: 1}, {ID: 2, Weight: 1}},
			CategoryID:    1,
			PaymentModeID: 1,
		},
	}

	cmd := NewListCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	formatter := format.NewAmountFormatter("en_US", "USD")
	printBillsTable(cmd, project, bills, formatter)

	output := buf.String()

	// Check that key elements are present
	if !bytes.Contains([]byte(output), []byte("Groceries")) {
		t.Error("Output should contain bill name 'Groceries'")
	}
	if !bytes.Contains([]byte(output), []byte("Alice")) {
		t.Error("Output should contain payer name 'Alice'")
	}
	if !bytes.Contains([]byte(output), []byte("Bob")) {
		t.Error("Output should contain ower name 'Bob'")
	}
	if !bytes.Contains([]byte(output), []byte("Food")) {
		t.Error("Output should contain category 'Food'")
	}
	if !bytes.Contains([]byte(output), []byte("Cash")) {
		t.Error("Output should contain payment method 'Cash'")
	}
	if !bytes.Contains([]byte(output), []byte("$ 50.00")) {
		t.Errorf("Output should contain formatted amount '$ 50.00', got:\n%s", output)
	}
	if !bytes.Contains([]byte(output), []byte("Total: 1 bill(s), $ 50.00")) {
		t.Errorf("Output should contain total with formatted amount, got:\n%s", output)
	}
}

func TestPrintBillsTableEmpty(t *testing.T) {
	resetListFlags()

	project := &api.Project{}
	bills := []api.BillResponse{}

	cmd := NewListCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	formatter := format.NewAmountFormatter("en_US", "")
	printBillsTable(cmd, project, bills, formatter)

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("No bills found")) {
		t.Error("Output should indicate no bills found")
	}
}

func TestBuildFiltersNameFilter(t *testing.T) {
	resetListFlags()

	project := &api.Project{}

	// Set name filter
	listName = "grocery"

	filters, err := buildFilters(project)
	if err != nil {
		t.Fatalf("buildFilters() error = %v", err)
	}

	if len(filters) != 1 {
		t.Fatalf("buildFilters() returned %d filters, want 1", len(filters))
	}

	// Test the filter
	bill1 := api.BillResponse{What: "Grocery shopping"}
	bill2 := api.BillResponse{What: "Dinner"}

	if !filters[0](bill1) {
		t.Error("Filter should match 'Grocery shopping'")
	}
	if filters[0](bill2) {
		t.Error("Filter should not match 'Dinner'")
	}

	resetListFlags()
}

func TestBuildFiltersAmountFilter(t *testing.T) {
	resetListFlags()

	project := &api.Project{}

	// Set amount filter
	listAmount = ">50"

	filters, err := buildFilters(project)
	if err != nil {
		t.Fatalf("buildFilters() error = %v", err)
	}

	if len(filters) != 1 {
		t.Fatalf("buildFilters() returned %d filters, want 1", len(filters))
	}

	// Test the filter
	bill1 := api.BillResponse{Amount: 100}
	bill2 := api.BillResponse{Amount: 50}
	bill3 := api.BillResponse{Amount: 25}

	if !filters[0](bill1) {
		t.Error("Filter should match amount 100")
	}
	if filters[0](bill2) {
		t.Error("Filter should not match amount 50 (not strictly greater)")
	}
	if filters[0](bill3) {
		t.Error("Filter should not match amount 25")
	}

	resetListFlags()
}

func resetListFlags() {
	ProjectID = ""
	listPaidBy = ""
	listPaidFor = nil
	listAmount = ""
	listName = ""
	listPaymentMethod = ""
	listCategory = ""
}
