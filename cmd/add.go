package cmd

import (
	"fmt"
	"strconv"
	"time"

	"github.com/chenasraf/cospend-cli/internal/api"
	"github.com/chenasraf/cospend-cli/internal/cache"
	"github.com/chenasraf/cospend-cli/internal/config"
	"github.com/spf13/cobra"
)

var (
	category      string
	paidBy        string
	paidFor       []string
	convertTo     string
	paymentMethod string
	comment       string
)

// NewAddCommand creates the add command
func NewAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <name> <amount>",
		Short: "Add an expense to a Cospend project",
		Long: `Add an expense to a Cospend project.

Examples:
  cospend add "Groceries" 25.50 -p myproject
  cospend add "Dinner" 45.00 -p myproject -c restaurant -b alice -f bob -f charlie`,
		Args: cobra.ExactArgs(2),
		RunE: runAdd,
	}

	cmd.Flags().StringVarP(&category, "category", "c", "", "Category by ID or name")
	cmd.Flags().StringVarP(&paidBy, "by", "b", "", "Paying member username (defaults to authenticated user)")
	cmd.Flags().StringArrayVarP(&paidFor, "for", "f", nil, "Owed member username (repeatable; defaults to payer only)")
	cmd.Flags().StringVarP(&convertTo, "convert", "C", "", "Currency to convert to")
	cmd.Flags().StringVarP(&paymentMethod, "method", "m", "", "Payment method by ID or name")
	cmd.Flags().StringVarP(&comment, "comment", "o", "", "Additional details about the bill")

	return cmd
}

func runAdd(cmd *cobra.Command, args []string) error {
	if ProjectID == "" {
		return fmt.Errorf("project is required (use -p or --project)")
	}

	expenseName := args[0]
	amountStr := args[1]

	// Parse amount
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return fmt.Errorf("invalid amount: %s", amountStr)
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
			// Non-fatal: log warning but continue
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to cache project: %v\n", err)
		}
	}

	// Resolve payer
	payerUsername := paidBy
	if payerUsername == "" {
		payerUsername = cfg.User
	}
	payerID, err := cache.ResolveMember(project, payerUsername)
	if err != nil {
		return fmt.Errorf("resolving payer: %w", err)
	}

	// Resolve owed members
	var owedIDs []int
	if len(paidFor) == 0 {
		// Default to payer only
		owedIDs = []int{payerID}
	} else {
		for _, username := range paidFor {
			memberID, err := cache.ResolveMember(project, username)
			if err != nil {
				return fmt.Errorf("resolving owed member: %w", err)
			}
			owedIDs = append(owedIDs, memberID)
		}
	}

	// Build bill
	bill := api.Bill{
		What:    expenseName,
		Amount:  amount,
		PayerID: payerID,
		OwedTo:  owedIDs,
		Date:    time.Now().Format("2006-01-02"),
	}

	// Resolve optional category
	if category != "" {
		categoryID, err := cache.ResolveCategory(project, category)
		if err != nil {
			return fmt.Errorf("resolving category: %w", err)
		}
		bill.CategoryID = categoryID
	}

	// Resolve optional payment method
	if paymentMethod != "" {
		methodID, err := cache.ResolvePaymentMode(project, paymentMethod)
		if err != nil {
			return fmt.Errorf("resolving payment method: %w", err)
		}
		bill.PaymentModeID = methodID
	}

	// Resolve optional currency
	if convertTo != "" {
		currencyID, err := cache.ResolveCurrency(project, convertTo)
		if err != nil {
			return fmt.Errorf("resolving currency: %w", err)
		}
		bill.OriginalCurrencyID = currencyID
	}

	// Add optional comment
	if comment != "" {
		bill.Comment = comment
	}

	// Create the bill
	if err := client.CreateBill(ProjectID, bill); err != nil {
		return fmt.Errorf("creating bill: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Successfully added expense: %s (%.2f)\n", expenseName, amount)
	return nil
}
