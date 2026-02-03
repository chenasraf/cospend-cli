package cmd

import (
	"fmt"

	"github.com/chenasraf/cospend-cli/internal/api"
	"github.com/chenasraf/cospend-cli/internal/config"
	"github.com/spf13/cobra"
)

var showAllProjects bool

// NewProjectsCommand creates the projects command
func NewProjectsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "projects",
		Aliases: []string{"proj"},
		Short:   "List available Cospend projects",
		Long:    `List all Cospend projects you have access to.`,
		RunE:    runProjects,
	}

	cmd.Flags().BoolVarP(&showAllProjects, "all", "a", false, "Show all projects including archived")

	return cmd
}

func runProjects(cmd *cobra.Command, _ []string) error {
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

	// Fetch projects
	projects, err := client.GetProjects()
	if err != nil {
		return fmt.Errorf("fetching projects: %w", err)
	}

	// Filter out archived projects unless --all is set
	var filtered []api.ProjectSummary
	for _, proj := range projects {
		if showAllProjects || !proj.IsArchived() {
			filtered = append(filtered, proj)
		}
	}

	// Print table
	out := cmd.OutOrStdout()
	if len(filtered) == 0 {
		_, _ = fmt.Fprintln(out, "No projects found.")
		return nil
	}

	table := NewTable("ID", "NAME", "CURRENCY")
	for _, proj := range filtered {
		currency := proj.CurrName
		if currency == "" {
			currency = "-"
		}
		table.AddRow(proj.ID, proj.Name, currency)
	}

	table.Render(out)
	_, _ = fmt.Fprintf(out, "\nTotal: %d project(s)\n", len(filtered))

	return nil
}
