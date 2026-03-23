package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/chenasraf/cospend-cli/internal/api"
	"github.com/chenasraf/cospend-cli/internal/cache"
	"github.com/chenasraf/cospend-cli/internal/config"
	"github.com/chenasraf/cospend-cli/internal/format"
	"github.com/spf13/cobra"
)

var (
	editName          string
	editAmount        string
	editCategory      string
	editPaidBy        string
	editPaidFor       []string
	editPaymentMethod string
	editComment       string
	editDate          string
)

// NewEditCommand creates the edit command
func NewEditCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "edit <bill_id>",
		Aliases: []string{"update"},
		Short:   "Edit an existing expense in a Cospend project",
		Long: `Edit an existing expense in a Cospend project.

Only specified flags will be updated; other fields remain unchanged.

Examples:
  cospend edit 123 -p myproject -n "Updated name"
  cospend edit 123 -p myproject -a 30.00 -c restaurant
  cospend edit 123 -p myproject -b alice -f bob -f charlie`,
		Args: cobra.ExactArgs(1),
		RunE: runEdit,
	}

	cmd.Flags().StringVarP(&editName, "name", "n", "", "New name/description")
	cmd.Flags().StringVarP(&editAmount, "amount", "a", "", "New amount")
	cmd.Flags().StringVarP(&editCategory, "category", "c", "", "Category by ID or name")
	cmd.Flags().StringVarP(&editPaidBy, "by", "b", "", "Paying member username")
	cmd.Flags().StringArrayVarP(&editPaidFor, "for", "f", nil, "Owed member username (repeatable)")
	cmd.Flags().StringVarP(&editPaymentMethod, "method", "m", "", "Payment method by ID or name")
	cmd.Flags().StringVarP(&editComment, "comment", "o", "", "Comment")
	cmd.Flags().StringVarP(&editDate, "date", "d", "", "Date (YYYY-MM-DD, MM-DD, or relative like -1d, +2w)")

	return cmd
}

func runEdit(cmd *cobra.Command, args []string) error {
	if ProjectID == "" {
		return fmt.Errorf("project is required (use -p or --project)")
	}

	billID, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid bill ID: %s", args[0])
	}

	// Parameters validated, silence usage for subsequent errors
	cmd.SilenceUsage = true

	cfg, err := config.Load()
	if err != nil {
		return err
	}

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

	// Fetch all bills to find the existing one
	bills, err := client.GetBills(ProjectID)
	if err != nil {
		return fmt.Errorf("fetching bills: %w", err)
	}

	var existing *api.BillResponse
	for i := range bills {
		if bills[i].ID == billID {
			existing = &bills[i]
			break
		}
	}
	if existing == nil {
		return fmt.Errorf("bill #%d not found", billID)
	}

	// Build member name lookup
	memberNames := make(map[int]string)
	for _, m := range project.Members {
		memberNames[m.ID] = m.Name
	}

	// Start from existing values
	bill := api.Bill{
		What:          existing.What,
		Amount:        existing.Amount,
		PayerID:       existing.PayerID,
		Date:          existing.Date,
		Comment:       existing.Comment,
		PaymentModeID: existing.PaymentModeID,
		CategoryID:    existing.CategoryID,
	}
	for _, o := range existing.Owers {
		bill.OwedTo = append(bill.OwedTo, o.ID)
	}

	// Apply changes for flags that were explicitly set
	if cmd.Flags().Changed("name") {
		bill.What = editName
	}

	if cmd.Flags().Changed("amount") {
		amount, err := strconv.ParseFloat(editAmount, 64)
		if err != nil {
			return fmt.Errorf("invalid amount: %s", editAmount)
		}
		bill.Amount = amount
	}

	if cmd.Flags().Changed("by") {
		payerID, err := cache.ResolveMember(project, editPaidBy)
		if err != nil {
			return fmt.Errorf("resolving payer: %w", err)
		}
		bill.PayerID = payerID
	}

	if cmd.Flags().Changed("for") {
		var owedIDs []int
		for _, username := range editPaidFor {
			memberID, err := cache.ResolveMember(project, username)
			if err != nil {
				return fmt.Errorf("resolving owed member: %w", err)
			}
			owedIDs = append(owedIDs, memberID)
		}
		bill.OwedTo = owedIDs
	}

	if cmd.Flags().Changed("date") {
		parsed, err := parseDate(editDate)
		if err != nil {
			return err
		}
		bill.Date = parsed
	}

	if cmd.Flags().Changed("category") {
		categoryID, err := cache.ResolveCategory(project, editCategory)
		if err != nil {
			return fmt.Errorf("resolving category: %w", err)
		}
		bill.CategoryID = categoryID
	}

	if cmd.Flags().Changed("method") {
		methodID, err := cache.ResolvePaymentMode(project, editPaymentMethod)
		if err != nil {
			return fmt.Errorf("resolving payment method: %w", err)
		}
		bill.PaymentModeID = methodID
	}

	if cmd.Flags().Changed("comment") {
		bill.Comment = editComment
	}

	// Edit the bill
	if err := client.EditBill(ProjectID, billID, bill); err != nil {
		return fmt.Errorf("editing bill: %w", err)
	}

	// Fetch user info for locale-aware formatting
	locale := "en_US"
	userInfo, ok := cache.LoadUserInfo()
	if !ok {
		userInfo, err = client.GetUserInfo()
		if err == nil {
			_ = cache.SaveUserInfo(userInfo)
		}
	}
	if userInfo != nil && userInfo.Locale != "" {
		locale = userInfo.Locale
	} else if userInfo != nil && userInfo.Language != "" {
		locale = userInfo.Language
	}

	formatter := format.NewAmountFormatter(locale, project.CurrencyName)
	out := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(out, "Updated bill #%d\n", billID)
	_, _ = fmt.Fprintf(out, "  Name:     %s\n", bill.What)
	_, _ = fmt.Fprintf(out, "  Amount:   %s\n", formatter.Format(bill.Amount))
	_, _ = fmt.Fprintf(out, "  Date:     %s\n", bill.Date)
	_, _ = fmt.Fprintf(out, "  Paid by:  %s\n", memberNames[bill.PayerID])
	var owerNames []string
	for _, id := range bill.OwedTo {
		owerNames = append(owerNames, memberNames[id])
	}
	_, _ = fmt.Fprintf(out, "  Paid for: %s\n", strings.Join(owerNames, ", "))
	if bill.CategoryID != 0 {
		for _, c := range project.Categories {
			if c.ID == bill.CategoryID {
				_, _ = fmt.Fprintf(out, "  Category: %s\n", c.Name)
				break
			}
		}
	}
	if bill.PaymentModeID != 0 {
		for _, pm := range project.PaymentModes {
			if pm.ID == bill.PaymentModeID {
				_, _ = fmt.Fprintf(out, "  Method:   %s\n", pm.Name)
				break
			}
		}
	}
	if bill.Comment != "" {
		_, _ = fmt.Fprintf(out, "  Comment:  %s\n", bill.Comment)
	}

	return nil
}
