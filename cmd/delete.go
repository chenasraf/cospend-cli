package cmd

import (
	"fmt"
	"strconv"

	"github.com/chenasraf/cospend-cli/internal/api"
	"github.com/chenasraf/cospend-cli/internal/config"
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

	// Delete the bill
	if err := client.DeleteBill(ProjectID, billID); err != nil {
		return fmt.Errorf("deleting bill: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Successfully deleted bill #%d\n", billID)
	return nil
}
