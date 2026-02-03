package cmd

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/chenasraf/cospend-cli/internal/api"
	"github.com/chenasraf/cospend-cli/internal/cache"
	"github.com/chenasraf/cospend-cli/internal/config"
	"github.com/spf13/cobra"
)

var (
	listPaidBy        string
	listPaidFor       []string
	listAmount        string
	listName          string
	listPaymentMethod string
	listCategory      string
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
  cospend list -p myproject --amount "<=100" -n dinner`,
		RunE: runList,
	}

	cmd.Flags().StringVarP(&listPaidBy, "by", "b", "", "Filter by paying member username")
	cmd.Flags().StringArrayVarP(&listPaidFor, "for", "f", nil, "Filter by owed member username (repeatable)")
	cmd.Flags().StringVarP(&listAmount, "amount", "a", "", "Filter by amount (e.g., 50, >30, <=100, =25)")
	cmd.Flags().StringVarP(&listName, "name", "n", "", "Filter by name (case-insensitive, contains)")
	cmd.Flags().StringVarP(&listPaymentMethod, "method", "m", "", "Filter by payment method")
	cmd.Flags().StringVarP(&listCategory, "category", "c", "", "Filter by category")

	return cmd
}

func runList(cmd *cobra.Command, _ []string) error {
	if ProjectID == "" {
		return fmt.Errorf("project is required (use -p or --project)")
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

	// Build filters
	filters, err := buildFilters(project)
	if err != nil {
		return err
	}

	// Apply filters
	filteredBills := applyFilters(bills, filters)

	// Print table
	printBillsTable(cmd, project, filteredBills)

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

func printBillsTable(cmd *cobra.Command, project *api.Project, bills []api.BillResponse) {
	if len(bills) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No bills found.")
		return
	}

	// Sort by date (newest first), then by timestamp for same-date entries
	sort.Slice(bills, func(i, j int) bool {
		if bills[i].Date != bills[j].Date {
			return bills[i].Date > bills[j].Date
		}
		return bills[i].Timestamp > bills[j].Timestamp
	})

	// Build lookup maps for names
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

	table := NewTable("ID", "DATE", "NAME", "AMOUNT", "PAID BY", "PAID FOR", "CATEGORY", "METHOD")

	for _, bill := range bills {
		// Get payer name
		payerName := memberNames[bill.PayerID]
		if payerName == "" {
			payerName = fmt.Sprintf("#%d", bill.PayerID)
		}

		// Get owed member names
		var owerNames []string
		for _, ower := range bill.Owers {
			name := memberNames[ower.ID]
			if name == "" {
				name = fmt.Sprintf("#%d", ower.ID)
			}
			owerNames = append(owerNames, name)
		}
		owersStr := strings.Join(owerNames, ", ")

		// Get category name
		catName := categoryNames[bill.CategoryID]
		if catName == "" && bill.CategoryID != 0 {
			catName = fmt.Sprintf("#%d", bill.CategoryID)
		}
		if catName == "" {
			catName = "-"
		}

		// Get payment method name
		methodName := paymentModeNames[bill.PaymentModeID]
		if methodName == "" && bill.PaymentModeID != 0 {
			methodName = fmt.Sprintf("#%d", bill.PaymentModeID)
		}
		if methodName == "" {
			methodName = "-"
		}

		// Truncate name if too long
		name := bill.What
		if len(name) > 30 {
			name = name[:27] + "..."
		}

		table.AddRow(
			fmt.Sprintf("%d", bill.ID),
			bill.Date,
			name,
			fmt.Sprintf("%.2f", bill.Amount),
			payerName,
			owersStr,
			catName,
			methodName,
		)
	}

	out := cmd.OutOrStdout()
	table.Render(out)
	_, _ = fmt.Fprintf(out, "\nTotal: %d bill(s)\n", len(bills))
}
