package main

import (
	_ "embed"
	"os"
	"strings"

	"github.com/chenasraf/cospend-cli/cmd"
	"github.com/chenasraf/cospend-cli/internal/config"
	"github.com/spf13/cobra"
)

//go:embed version.txt
var version string

func main() {
	rootCmd := &cobra.Command{
		Use:              "cospend",
		Short:            "A CLI tool for Nextcloud Cospend",
		Long:             `cospend is a command-line interface for adding expenses to Nextcloud Cospend projects.`,
		Version:          strings.TrimSpace(version),
		TraverseChildren: true,
		PersistentPreRun: func(c *cobra.Command, args []string) {
			// Apply default project from config if -p not explicitly set
			if cmd.ProjectID == "" {
				if raw := config.LoadRaw(); raw.DefaultProject != "" {
					cmd.ProjectID = raw.DefaultProject
				}
			}
		},
	}

	rootCmd.AddCommand(cmd.NewAddCommand())
	rootCmd.AddCommand(cmd.NewInitCommand())
	rootCmd.AddCommand(cmd.NewListCommand())
	rootCmd.AddCommand(cmd.NewDeleteCommand())
	rootCmd.AddCommand(cmd.NewEditCommand())
	rootCmd.AddCommand(cmd.NewProjectsCommand())
	rootCmd.AddCommand(cmd.NewInfoCommand())
	rootCmd.AddCommand(cmd.NewConfigCommand())

	rootCmd.PersistentFlags().BoolVarP(&cmd.Debug, "debug", "D", false, "Enable debug output")
	rootCmd.PersistentFlags().StringVarP(&cmd.ProjectID, "project", "p", "", "Project ID")
	rootCmd.Flags().BoolP("version", "v", false, "Print version information")
	rootCmd.SetVersionTemplate("{{.Version}}\n")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
