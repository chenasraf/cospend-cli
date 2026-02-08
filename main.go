package main

import (
	_ "embed"
	"os"
	"strings"

	"github.com/chenasraf/cospend-cli/cmd"
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
	}

	rootCmd.AddCommand(cmd.NewAddCommand())
	rootCmd.AddCommand(cmd.NewInitCommand())
	rootCmd.AddCommand(cmd.NewListCommand())
	rootCmd.AddCommand(cmd.NewDeleteCommand())
	rootCmd.AddCommand(cmd.NewProjectsCommand())
	rootCmd.AddCommand(cmd.NewInfoCommand())

	rootCmd.PersistentFlags().BoolVarP(&cmd.Debug, "debug", "d", false, "Enable debug output")
	rootCmd.PersistentFlags().StringVarP(&cmd.ProjectID, "project", "p", "", "Project ID")
	rootCmd.Flags().BoolP("version", "v", false, "Print version information")
	rootCmd.SetVersionTemplate("{{.Version}}\n")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
