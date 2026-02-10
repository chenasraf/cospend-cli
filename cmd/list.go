package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/chenasraf/cospend-cli/internal/api"
	"github.com/chenasraf/cospend-cli/internal/cache"
	"github.com/chenasraf/cospend-cli/internal/config"
	"github.com/chenasraf/cospend-cli/internal/format"
	"github.com/spf13/cobra"
)

var (
	listPaidBy        string
	listPaidFor       []string
	listAmount        string
	listName          string
	listPaymentMethod string
	listCategory      string
	listLimit         int
	listDate          string
	listToday         bool
	listThisMonth     bool
	listThisWeek      bool
	listRecent        string
	listFormat        string
)

// amountFilter holds parsed amount filter criteria
type amountFilter struct {
	operator string
	value    float64
}

// NewListCommand creates the list command
func NewListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List expenses in a Cospend project",
		Long: `List expenses in a Cospend project with optional filters.

Examples:
  cospend list -p myproject
  cospend list -p myproject -b alice
  cospend list -p myproject -c groceries
  cospend list -p myproject --amount ">50"
  cospend list -p myproject --amount "<=100" -n dinner
  cospend list -p myproject --today
  cospend list -p myproject --date ">=2026-01-01"
  cospend list -p myproject --date "<=01-15"
  cospend list -p myproject --this-month
  cospend list -p myproject --this-week
  cospend list -p myproject --recent 7d
  cospend list -p myproject --recent 2w`,
		RunE: runList,
	}

	cmd.Flags().StringVarP(&listPaidBy, "by", "b", "", "Filter by paying member username")
	cmd.Flags().StringArrayVarP(&listPaidFor, "for", "f", nil, "Filter by owed member username (repeatable)")
	cmd.Flags().StringVarP(&listAmount, "amount", "a", "", "Filter by amount (e.g., 50, >30, <=100, =25)")
	cmd.Flags().StringVarP(&listName, "name", "n", "", "Filter by name (case-insensitive, contains)")
	cmd.Flags().StringVarP(&listPaymentMethod, "method", "m", "", "Filter by payment method")
	cmd.Flags().StringVarP(&listCategory, "category", "c", "", "Filter by category")
	cmd.Flags().IntVarP(&listLimit, "limit", "l", 0, "Limit number of results (0 = no limit)")
	cmd.Flags().StringVar(&listDate, "date", "", "Filter by date (e.g., 2026-01-15, >=2026-01-01, <=01-15)")
	cmd.Flags().BoolVar(&listToday, "today", false, "Filter bills from today")
	cmd.Flags().BoolVar(&listThisMonth, "this-month", false, "Filter bills from the current month")
	cmd.Flags().BoolVar(&listThisWeek, "this-week", false, "Filter bills from the current calendar week")
	cmd.Flags().StringVar(&listRecent, "recent", "", "Filter recent bills (e.g., 7d, 2w, 1m)")
	cmd.Flags().StringVar(&listFormat, "format", "table", "Output format: table, csv, json")

	return cmd
}

func runList(cmd *cobra.Command, _ []string) error {
	if ProjectID == "" {
		return fmt.Errorf("project is required (use -p or --project)")
	}

	switch listFormat {
	case "table", "csv", "json":
	default:
		return fmt.Errorf("unsupported format: %s (expected table, csv, or json)", listFormat)
	}

	// Parameters validated, silence usage for subsequent errors
	cmd.SilenceUsage = true

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Get API client
	client := api.NewClient(cfg)
	client.Debug = Debug
	client.DebugWriter = cmd.ErrOrStderr()

	// Get project (from cache or API)
	project, ok := cache.Load(ProjectID)
	if !ok {
		project, err = client.GetProject(ProjectID)
		if err != nil {
			return fmt.Errorf("fetching project: %w", err)
		}
		if err := cache.Save(ProjectID, project); err != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to cache project: %v\n", err)
		}
	}

	// Fetch bills
	bills, err := client.GetBills(ProjectID)
	if err != nil {
		return fmt.Errorf("fetching bills: %w", err)
	}

	// Fetch user info for locale (with cache, graceful fallback)
	locale := "en_US"
	userInfo, ok := cache.LoadUserInfo()
	if !ok {
		userInfo, err = client.GetUserInfo()
		if err != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to fetch user info: %v\n", err)
		} else {
			if err := cache.SaveUserInfo(userInfo); err != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to cache user info: %v\n", err)
			}
		}
	}
	if userInfo != nil && userInfo.Locale != "" {
		locale = userInfo.Locale
	} else if userInfo != nil && userInfo.Language != "" {
		locale = userInfo.Language
	}

	// Build filters
	filters, err := buildFilters(project)
	if err != nil {
		return err
	}

	// Apply filters
	filteredBills := applyFilters(bills, filters)

	// Output results
	formatter := format.NewAmountFormatter(locale, project.CurrencyName)
	resolved := resolveBillNames(project, filteredBills)

	switch listFormat {
	case "csv":
		printBillsCSV(cmd, resolved)
	case "json":
		printBillsJSON(cmd, resolved)
	default:
		printBillsTable(cmd, resolved, formatter)
	}

	return nil
}

// billFilter is a function that returns true if a bill should be included
type billFilter func(bill api.BillResponse) bool

func buildFilters(project *api.Project) ([]billFilter, error) {
	var filters []billFilter

	// Filter by payer
	if listPaidBy != "" {
		payerID, err := cache.ResolveMember(project, listPaidBy)
		if err != nil {
			return nil, fmt.Errorf("resolving payer filter: %w", err)
		}
		filters = append(filters, func(bill api.BillResponse) bool {
			return bill.PayerID == payerID
		})
	}

	// Filter by owed members
	if len(listPaidFor) > 0 {
		var owedIDs []int
		for _, username := range listPaidFor {
			memberID, err := cache.ResolveMember(project, username)
			if err != nil {
				return nil, fmt.Errorf("resolving owed member filter: %w", err)
			}
			owedIDs = append(owedIDs, memberID)
		}
		filters = append(filters, func(bill api.BillResponse) bool {
			for _, requiredID := range owedIDs {
				found := false
				for _, ower := range bill.Owers {
					if ower.ID == requiredID {
						found = true
						break
					}
				}
				if !found {
					return false
				}
			}
			return true
		})
	}

	// Filter by amount
	if listAmount != "" {
		af, err := parseAmountFilter(listAmount)
		if err != nil {
			return nil, fmt.Errorf("parsing amount filter: %w", err)
		}
		filters = append(filters, func(bill api.BillResponse) bool {
			return matchAmount(bill.Amount, af)
		})
	}

	// Filter by name (case-insensitive contains)
	if listName != "" {
		lowerName := strings.ToLower(listName)
		filters = append(filters, func(bill api.BillResponse) bool {
			return strings.Contains(strings.ToLower(bill.What), lowerName)
		})
	}

	// Filter by payment method
	if listPaymentMethod != "" {
		methodID, err := cache.ResolvePaymentMode(project, listPaymentMethod)
		if err != nil {
			return nil, fmt.Errorf("resolving payment method filter: %w", err)
		}
		filters = append(filters, func(bill api.BillResponse) bool {
			return bill.PaymentModeID == methodID
		})
	}

	// Filter by category
	if listCategory != "" {
		categoryID, err := cache.ResolveCategory(project, listCategory)
		if err != nil {
			return nil, fmt.Errorf("resolving category filter: %w", err)
		}
		filters = append(filters, func(bill api.BillResponse) bool {
			return bill.CategoryID == categoryID
		})
	}

	// Filter by today
	if listToday {
		today := time.Now().Format("2006-01-02")
		filters = append(filters, func(bill api.BillResponse) bool {
			return bill.Date == today
		})
	}

	// Filter by date
	if listDate != "" {
		df, err := parseDateFilter(listDate)
		if err != nil {
			return nil, fmt.Errorf("parsing date filter: %w", err)
		}
		filters = append(filters, func(bill api.BillResponse) bool {
			return matchDate(bill.Date, df)
		})
	}

	// Filter by this month
	if listThisMonth {
		now := time.Now()
		prefix := now.Format("2006-01")
		filters = append(filters, func(bill api.BillResponse) bool {
			return strings.HasPrefix(bill.Date, prefix)
		})
	}

	// Filter by this week
	if listThisWeek {
		now := time.Now()
		weekday := now.Weekday()
		if weekday == time.Sunday {
			weekday = 7
		}
		startOfWeek := now.AddDate(0, 0, -int(weekday-time.Monday))
		endOfWeek := startOfWeek.AddDate(0, 0, 6)
		startStr := startOfWeek.Format("2006-01-02")
		endStr := endOfWeek.Format("2006-01-02")
		filters = append(filters, func(bill api.BillResponse) bool {
			return bill.Date >= startStr && bill.Date <= endStr
		})
	}

	// Filter by recent duration
	if listRecent != "" {
		cutoff, err := parseRecent(listRecent)
		if err != nil {
			return nil, fmt.Errorf("parsing recent filter: %w", err)
		}
		cutoffStr := cutoff.Format("2006-01-02")
		filters = append(filters, func(bill api.BillResponse) bool {
			return bill.Date >= cutoffStr
		})
	}

	return filters, nil
}

func applyFilters(bills []api.BillResponse, filters []billFilter) []api.BillResponse {
	if len(filters) == 0 {
		return bills
	}

	var result []api.BillResponse
	for _, bill := range bills {
		include := true
		for _, filter := range filters {
			if !filter(bill) {
				include = false
				break
			}
		}
		if include {
			result = append(result, bill)
		}
	}
	return result
}

func parseAmountFilter(s string) (amountFilter, error) {
	s = strings.TrimSpace(s)

	// Match operators: >=, <=, >, <, =, or just a number
	re := regexp.MustCompile(`^(>=|<=|>|<|=)?(.+)$`)
	matches := re.FindStringSubmatch(s)
	if matches == nil {
		return amountFilter{}, fmt.Errorf("invalid amount filter format: %s", s)
	}

	operator := matches[1]
	if operator == "" {
		operator = "="
	}

	value, err := strconv.ParseFloat(strings.TrimSpace(matches[2]), 64)
	if err != nil {
		return amountFilter{}, fmt.Errorf("invalid amount value: %s", matches[2])
	}

	return amountFilter{operator: operator, value: value}, nil
}

func matchAmount(amount float64, af amountFilter) bool {
	switch af.operator {
	case "=":
		return amount == af.value
	case ">":
		return amount > af.value
	case "<":
		return amount < af.value
	case ">=":
		return amount >= af.value
	case "<=":
		return amount <= af.value
	default:
		return false
	}
}

// dateFilter holds parsed date filter criteria
type dateFilter struct {
	operator string
	date     string // YYYY-MM-DD format for string comparison
}

func parseDateFilter(s string) (dateFilter, error) {
	s = strings.TrimSpace(s)

	re := regexp.MustCompile(`^(>=|<=|>|<|=)?(.+)$`)
	matches := re.FindStringSubmatch(s)
	if matches == nil {
		return dateFilter{}, fmt.Errorf("invalid date filter format: %s", s)
	}

	operator := matches[1]
	if operator == "" {
		operator = "="
	}

	dateStr := strings.TrimSpace(matches[2])

	// Try full date format YYYY-MM-DD
	if _, err := time.Parse("2006-01-02", dateStr); err == nil {
		return dateFilter{operator: operator, date: dateStr}, nil
	}

	// Try short format MM-DD (assume current year)
	if t, err := time.Parse("01-02", dateStr); err == nil {
		dateStr = fmt.Sprintf("%d-%s", time.Now().Year(), t.Format("01-02"))
		return dateFilter{operator: operator, date: dateStr}, nil
	}

	return dateFilter{}, fmt.Errorf("invalid date format: %s (expected YYYY-MM-DD or MM-DD)", dateStr)
}

func matchDate(billDate string, df dateFilter) bool {
	switch df.operator {
	case "=":
		return billDate == df.date
	case ">":
		return billDate > df.date
	case "<":
		return billDate < df.date
	case ">=":
		return billDate >= df.date
	case "<=":
		return billDate <= df.date
	default:
		return false
	}
}

func parseRecent(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return time.Time{}, fmt.Errorf("invalid recent format: %s (expected e.g. 7d, 2w, 1m)", s)
	}

	unit := s[len(s)-1]
	valueStr := s[:len(s)-1]
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid recent value: %s", valueStr)
	}

	now := time.Now()
	switch unit {
	case 'd':
		return now.AddDate(0, 0, -value), nil
	case 'w':
		return now.AddDate(0, 0, -value*7), nil
	case 'm':
		return now.AddDate(0, -value, 0), nil
	default:
		return time.Time{}, fmt.Errorf("invalid recent unit: %c (expected d, w, or m)", unit)
	}
}

// resolvedBill holds a bill with human-readable names resolved from IDs
type resolvedBill struct {
	ID            int      `json:"id"`
	Date          string   `json:"date"`
	Name          string   `json:"name"`
	Amount        float64  `json:"amount"`
	PaidBy        string   `json:"paid_by"`
	PaidFor       []string `json:"paid_for"`
	Category      string   `json:"category"`
	PaymentMethod string   `json:"payment_method"`
}

func resolveBillNames(project *api.Project, bills []api.BillResponse) []resolvedBill {
	// Sort by date (newest first), then by timestamp for same-date entries
	sort.Slice(bills, func(i, j int) bool {
		if bills[i].Date != bills[j].Date {
			return bills[i].Date > bills[j].Date
		}
		return bills[i].Timestamp > bills[j].Timestamp
	})

	// Apply limit if set
	if listLimit > 0 && len(bills) > listLimit {
		bills = bills[:listLimit]
	}

	// Build lookup maps
	memberNames := make(map[int]string)
	for _, m := range project.Members {
		memberNames[m.ID] = m.Name
	}
	categoryNames := make(map[int]string)
	for _, c := range project.Categories {
		categoryNames[c.ID] = c.Name
	}
	paymentModeNames := make(map[int]string)
	for _, pm := range project.PaymentModes {
		paymentModeNames[pm.ID] = pm.Name
	}

	var result []resolvedBill
	for _, bill := range bills {
		payerName := memberNames[bill.PayerID]
		if payerName == "" {
			payerName = fmt.Sprintf("#%d", bill.PayerID)
		}

		var owerNames []string
		for _, ower := range bill.Owers {
			name := memberNames[ower.ID]
			if name == "" {
				name = fmt.Sprintf("#%d", ower.ID)
			}
			owerNames = append(owerNames, name)
		}

		catName := categoryNames[bill.CategoryID]
		if catName == "" && bill.CategoryID != 0 {
			catName = fmt.Sprintf("#%d", bill.CategoryID)
		}

		methodName := paymentModeNames[bill.PaymentModeID]
		if methodName == "" && bill.PaymentModeID != 0 {
			methodName = fmt.Sprintf("#%d", bill.PaymentModeID)
		}

		name := strings.Map(func(r rune) rune {
			if r == '\n' || r == '\r' || r == '\t' {
				return ' '
			}
			return r
		}, strings.TrimSpace(bill.What))

		result = append(result, resolvedBill{
			ID:            bill.ID,
			Date:          bill.Date,
			Name:          name,
			Amount:        bill.Amount,
			PaidBy:        payerName,
			PaidFor:       owerNames,
			Category:      catName,
			PaymentMethod: methodName,
		})
	}
	return result
}

func printBillsTable(cmd *cobra.Command, bills []resolvedBill, formatter *format.AmountFormatter) {
	if len(bills) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No bills found.")
		return
	}

	table := NewTable("ID", "DATE", "NAME", "AMOUNT", "PAID BY", "PAID FOR", "CATEGORY", "METHOD")

	var totalAmount float64
	for _, bill := range bills {
		totalAmount += bill.Amount

		catName := bill.Category
		if catName == "" {
			catName = "-"
		}
		methodName := bill.PaymentMethod
		if methodName == "" {
			methodName = "-"
		}

		name := bill.Name
		if len(name) > 30 {
			name = name[:27] + "..."
		}

		table.AddRow(
			fmt.Sprintf("%d", bill.ID),
			bill.Date,
			name,
			formatter.Format(bill.Amount),
			bill.PaidBy,
			strings.Join(bill.PaidFor, ", "),
			catName,
			methodName,
		)
	}

	out := cmd.OutOrStdout()
	table.Render(out)
	_, _ = fmt.Fprintf(out, "\nTotal: %d bill(s), %s\n", len(bills), formatter.Format(totalAmount))
}

func printBillsCSV(cmd *cobra.Command, bills []resolvedBill) {
	out := cmd.OutOrStdout()
	w := csv.NewWriter(out)

	_ = w.Write([]string{"ID", "Date", "Name", "Amount", "Paid By", "Paid For", "Category", "Payment Method"})
	for _, bill := range bills {
		_ = w.Write([]string{
			strconv.Itoa(bill.ID),
			bill.Date,
			bill.Name,
			strconv.FormatFloat(bill.Amount, 'f', 2, 64),
			bill.PaidBy,
			strings.Join(bill.PaidFor, ", "),
			bill.Category,
			bill.PaymentMethod,
		})
	}
	w.Flush()
}

func printBillsJSON(cmd *cobra.Command, bills []resolvedBill) {
	out := cmd.OutOrStdout()
	if bills == nil {
		bills = []resolvedBill{}
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	_ = enc.Encode(bills)
}
