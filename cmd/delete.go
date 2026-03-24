package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/chenasraf/cospend-cli/internal/api"
	"github.com/chenasraf/cospend-cli/internal/cache"
	"github.com/chenasraf/cospend-cli/internal/config"
	"github.com/chenasraf/cospend-cli/internal/format"
	"github.com/spf13/cobra"
)

// NewDeleteCommand creates the delete command
func NewDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <bill_id>",
		Aliases: []string{"rm"},
		Short:   "Delete an expense from a Cospend project",
		Long: `Delete an expense from a Cospend project by its bill ID.

Use 'cospend list' to find the bill ID you want to delete.

Examples:
  cospend delete 123 -p myproject`,
		Args: cobra.ExactArgs(1),
		RunE: runDelete,
	}

	return cmd
}

func runDelete(cmd *cobra.Command, args []string) error {
	if ProjectID == "" {
		return fmt.Errorf("project is required (use -p or --project)")
	}

	billIDStr := args[0]

	// Parse bill ID
	billID, err := strconv.Atoi(billIDStr)
	if err != nil {
		return fmt.Errorf("invalid bill ID: %s", billIDStr)
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

	// Confirm if configured
	if cfg.ConfirmDelete {
		// Fetch bill details for preview
		bills, err := client.GetBills(ProjectID)
		if err != nil {
			return fmt.Errorf("fetching bills: %w", err)
		}

		var bill *api.BillResponse
		for i := range bills {
			if bills[i].ID == billID {
				bill = &bills[i]
				break
			}
		}

		out := cmd.OutOrStdout()
		if bill != nil {
			// Fetch project for member names and currency
			project, ok := cache.Load(ProjectID)
			if !ok {
				project, err = client.GetProject(ProjectID)
				if err != nil {
					return fmt.Errorf("fetching project: %w", err)
				}
			}
			memberNames := make(map[int]string)
			for _, m := range project.Members {
				memberNames[m.ID] = m.Name
			}

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
			_, _ = fmt.Fprintf(out, "Bill #%d:\n", billID)
			_, _ = fmt.Fprintf(out, "  Name:     %s\n", bill.What)
			_, _ = fmt.Fprintf(out, "  Amount:   %s\n", formatter.Format(bill.Amount))
			_, _ = fmt.Fprintf(out, "  Date:     %s\n", bill.Date)
			_, _ = fmt.Fprintf(out, "  Paid by:  %s\n", memberNames[bill.PayerID])
		}

		if !confirm(os.Stdin, out, fmt.Sprintf("Delete bill #%d?", billID)) {
			_, _ = fmt.Fprintln(out, "Cancelled.")
			return nil
		}
	}

	// Delete the bill
	if err := client.DeleteBill(ProjectID, billID); err != nil {
		return fmt.Errorf("deleting bill: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Successfully deleted bill #%d\n", billID)
	return nil
}
