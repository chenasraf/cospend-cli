package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

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

	resolved := resolveBillNames(project, bills)

	cmd := NewListCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	formatter := format.NewAmountFormatter("en_US", "USD")
	printBillsTable(cmd, resolved, formatter)

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

	cmd := NewListCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	formatter := format.NewAmountFormatter("en_US", "")
	printBillsTable(cmd, nil, formatter)

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

func TestParseDateFilter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantOp   string
		wantDate string
		wantErr  bool
	}{
		{"full date", "2026-01-15", "=", "2026-01-15", false},
		{"full date with equals", "=2026-01-15", "=", "2026-01-15", false},
		{"full date gte", ">=2026-01-01", ">=", "2026-01-01", false},
		{"full date lte", "<=2026-12-31", "<=", "2026-12-31", false},
		{"full date gt", ">2026-06-15", ">", "2026-06-15", false},
		{"full date lt", "<2026-03-01", "<", "2026-03-01", false},
		{"short date", "01-15", "=", fmt.Sprintf("%d-01-15", time.Now().Year()), false},
		{"short date gte", ">=01-01", ">=", fmt.Sprintf("%d-01-01", time.Now().Year()), false},
		{"short date lte", "<=12-31", "<=", fmt.Sprintf("%d-12-31", time.Now().Year()), false},
		{"with spaces", " >= 2026-01-01 ", ">=", "2026-01-01", false},
		{"invalid date", "not-a-date", "", "", true},
		{"invalid short", "13-40", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			df, err := parseDateFilter(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDateFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if df.operator != tt.wantOp {
					t.Errorf("parseDateFilter() operator = %v, want %v", df.operator, tt.wantOp)
				}
				if df.date != tt.wantDate {
					t.Errorf("parseDateFilter() date = %v, want %v", df.date, tt.wantDate)
				}
			}
		})
	}
}

func TestMatchDate(t *testing.T) {
	tests := []struct {
		name     string
		billDate string
		filter   dateFilter
		want     bool
	}{
		{"equals match", "2026-01-15", dateFilter{"=", "2026-01-15"}, true},
		{"equals no match", "2026-01-15", dateFilter{"=", "2026-01-16"}, false},
		{"gte match exact", "2026-01-15", dateFilter{">=", "2026-01-15"}, true},
		{"gte match after", "2026-01-16", dateFilter{">=", "2026-01-15"}, true},
		{"gte no match", "2026-01-14", dateFilter{">=", "2026-01-15"}, false},
		{"lte match exact", "2026-01-15", dateFilter{"<=", "2026-01-15"}, true},
		{"lte match before", "2026-01-14", dateFilter{"<=", "2026-01-15"}, true},
		{"lte no match", "2026-01-16", dateFilter{"<=", "2026-01-15"}, false},
		{"gt match", "2026-01-16", dateFilter{">", "2026-01-15"}, true},
		{"gt no match exact", "2026-01-15", dateFilter{">", "2026-01-15"}, false},
		{"lt match", "2026-01-14", dateFilter{"<", "2026-01-15"}, true},
		{"lt no match exact", "2026-01-15", dateFilter{"<", "2026-01-15"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchDate(tt.billDate, tt.filter); got != tt.want {
				t.Errorf("matchDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseRecent(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		input   string
		wantDay string
		wantErr bool
	}{
		{"7 days", "7d", now.AddDate(0, 0, -7).Format("2006-01-02"), false},
		{"2 weeks", "2w", now.AddDate(0, 0, -14).Format("2006-01-02"), false},
		{"1 month", "1m", now.AddDate(0, -1, 0).Format("2006-01-02"), false},
		{"3 months", "3m", now.AddDate(0, -3, 0).Format("2006-01-02"), false},
		{"invalid unit", "7x", "", true},
		{"invalid value", "abcd", "", true},
		{"too short", "d", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRecent(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRecent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				gotDay := got.Format("2006-01-02")
				if gotDay != tt.wantDay {
					t.Errorf("parseRecent() = %v, want %v", gotDay, tt.wantDay)
				}
			}
		})
	}
}

func TestBuildFiltersToday(t *testing.T) {
	resetListFlags()
	defer resetListFlags()

	project := &api.Project{}
	listToday = true

	filters, err := buildFilters(project)
	if err != nil {
		t.Fatalf("buildFilters() error = %v", err)
	}

	if len(filters) != 1 {
		t.Fatalf("buildFilters() returned %d filters, want 1", len(filters))
	}

	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	if !filters[0](api.BillResponse{Date: today}) {
		t.Errorf("Filter should match today: %s", today)
	}
	if filters[0](api.BillResponse{Date: yesterday}) {
		t.Errorf("Filter should not match yesterday: %s", yesterday)
	}
}

func TestBuildFiltersDateFilter(t *testing.T) {
	resetListFlags()
	defer resetListFlags()

	project := &api.Project{}
	listDate = ">=2026-01-15"

	filters, err := buildFilters(project)
	if err != nil {
		t.Fatalf("buildFilters() error = %v", err)
	}

	if len(filters) != 1 {
		t.Fatalf("buildFilters() returned %d filters, want 1", len(filters))
	}

	bill1 := api.BillResponse{Date: "2026-01-15"}
	bill2 := api.BillResponse{Date: "2026-02-01"}
	bill3 := api.BillResponse{Date: "2026-01-14"}

	if !filters[0](bill1) {
		t.Error("Filter should match date 2026-01-15")
	}
	if !filters[0](bill2) {
		t.Error("Filter should match date 2026-02-01")
	}
	if filters[0](bill3) {
		t.Error("Filter should not match date 2026-01-14")
	}
}

func TestBuildFiltersThisMonth(t *testing.T) {
	resetListFlags()
	defer resetListFlags()

	project := &api.Project{}
	listThisMonth = true

	filters, err := buildFilters(project)
	if err != nil {
		t.Fatalf("buildFilters() error = %v", err)
	}

	if len(filters) != 1 {
		t.Fatalf("buildFilters() returned %d filters, want 1", len(filters))
	}

	now := time.Now()
	thisMonth := now.Format("2006-01-02")
	lastMonth := now.AddDate(0, -1, 0).Format("2006-01-02")

	if !filters[0](api.BillResponse{Date: thisMonth}) {
		t.Errorf("Filter should match date in current month: %s", thisMonth)
	}
	if filters[0](api.BillResponse{Date: lastMonth}) {
		t.Errorf("Filter should not match date in previous month: %s", lastMonth)
	}
}

func TestBuildFiltersThisWeek(t *testing.T) {
	resetListFlags()
	defer resetListFlags()

	project := &api.Project{}
	listThisWeek = true

	filters, err := buildFilters(project)
	if err != nil {
		t.Fatalf("buildFilters() error = %v", err)
	}

	if len(filters) != 1 {
		t.Fatalf("buildFilters() returned %d filters, want 1", len(filters))
	}

	today := time.Now().Format("2006-01-02")
	twoWeeksAgo := time.Now().AddDate(0, 0, -14).Format("2006-01-02")

	if !filters[0](api.BillResponse{Date: today}) {
		t.Errorf("Filter should match today: %s", today)
	}
	if filters[0](api.BillResponse{Date: twoWeeksAgo}) {
		t.Errorf("Filter should not match two weeks ago: %s", twoWeeksAgo)
	}
}

func TestBuildFiltersRecent(t *testing.T) {
	resetListFlags()
	defer resetListFlags()

	project := &api.Project{}
	listRecent = "7d"

	filters, err := buildFilters(project)
	if err != nil {
		t.Fatalf("buildFilters() error = %v", err)
	}

	if len(filters) != 1 {
		t.Fatalf("buildFilters() returned %d filters, want 1", len(filters))
	}

	today := time.Now().Format("2006-01-02")
	threeDaysAgo := time.Now().AddDate(0, 0, -3).Format("2006-01-02")
	tenDaysAgo := time.Now().AddDate(0, 0, -10).Format("2006-01-02")

	if !filters[0](api.BillResponse{Date: today}) {
		t.Error("Filter should match today")
	}
	if !filters[0](api.BillResponse{Date: threeDaysAgo}) {
		t.Error("Filter should match 3 days ago")
	}
	if filters[0](api.BillResponse{Date: tenDaysAgo}) {
		t.Error("Filter should not match 10 days ago")
	}
}

func TestPrintBillsCSV(t *testing.T) {
	resetListFlags()

	project := &api.Project{
		Members: []api.Member{
			{ID: 1, Name: "Alice", UserID: "alice"},
			{ID: 2, Name: "Bob", UserID: "bob"},
		},
		Categories: []api.Category{
			{ID: 1, Name: "Food"},
		},
		PaymentModes: []api.PaymentMode{
			{ID: 1, Name: "Cash"},
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
		{
			ID:      2,
			What:    "Coffee",
			Amount:  5.50,
			Date:    "2026-02-04",
			PayerID: 2,
			Owers:   []api.Ower{{ID: 2, Weight: 1}},
		},
	}

	resolved := resolveBillNames(project, bills)

	cmd := NewListCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	printBillsCSV(cmd, resolved)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 3 {
		t.Fatalf("Expected 3 lines (header + 2 rows), got %d:\n%s", len(lines), output)
	}
	if lines[0] != "ID,Date,Name,Amount,Paid By,Paid For,Category,Payment Method" {
		t.Errorf("Wrong CSV header: %s", lines[0])
	}
	if !strings.Contains(lines[1], "Coffee") {
		t.Errorf("First data row should contain 'Coffee' (newest first), got: %s", lines[1])
	}
	if !strings.Contains(lines[2], "Groceries") {
		t.Errorf("Second data row should contain 'Groceries', got: %s", lines[2])
	}
}

func TestPrintBillsJSON(t *testing.T) {
	resetListFlags()

	project := &api.Project{
		Members: []api.Member{
			{ID: 1, Name: "Alice", UserID: "alice"},
		},
		Categories: []api.Category{
			{ID: 1, Name: "Food"},
		},
		PaymentModes: []api.PaymentMode{
			{ID: 1, Name: "Cash"},
		},
	}

	bills := []api.BillResponse{
		{
			ID:            1,
			What:          "Groceries",
			Amount:        50.00,
			Date:          "2026-02-03",
			PayerID:       1,
			Owers:         []api.Ower{{ID: 1, Weight: 1}},
			CategoryID:    1,
			PaymentModeID: 1,
		},
	}

	resolved := resolveBillNames(project, bills)

	cmd := NewListCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	printBillsJSON(cmd, resolved)

	var result []resolvedBill
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON output: %v\n%s", err, buf.String())
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 bill, got %d", len(result))
	}
	if result[0].Name != "Groceries" {
		t.Errorf("Wrong name: %s", result[0].Name)
	}
	if result[0].Amount != 50.00 {
		t.Errorf("Wrong amount: %f", result[0].Amount)
	}
	if result[0].PaidBy != "Alice" {
		t.Errorf("Wrong paid_by: %s", result[0].PaidBy)
	}
	if result[0].Category != "Food" {
		t.Errorf("Wrong category: %s", result[0].Category)
	}
	if result[0].PaymentMethod != "Cash" {
		t.Errorf("Wrong payment_method: %s", result[0].PaymentMethod)
	}
}

func TestPrintBillsJSONEmpty(t *testing.T) {
	resetListFlags()

	cmd := NewListCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	printBillsJSON(cmd, nil)

	var result []resolvedBill
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON output: %v\n%s", err, buf.String())
	}
	if len(result) != 0 {
		t.Errorf("Expected empty array, got %d items", len(result))
	}
}

func resetListFlags() {
	ProjectID = ""
	listPaidBy = ""
	listPaidFor = nil
	listAmount = ""
	listName = ""
	listPaymentMethod = ""
	listCategory = ""
	listLimit = 0
	listDate = ""
	listToday = false
	listThisMonth = false
	listThisWeek = false
	listRecent = ""
	listFormat = "table"
}
