package cmd

import (
	"fmt"
	"os"

	"github.com/chenasraf/cospend-cli/internal/cache"
	"github.com/chenasraf/cospend-cli/internal/config"
	"github.com/spf13/cobra"
)

// NewLogoutCommand creates the logout command
func NewLogoutCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove configuration and cached data",
		Long: `Remove the configuration file and clear all cached data.

This effectively logs you out by removing your stored credentials
and any cached project data.`,
		RunE: runLogout,
	}
}

func runLogout(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	out := cmd.OutOrStdout()
	removed := false

	// Remove config file
	configPath := config.GetConfigPath()
	if configPath != "" {
		if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing config file: %w", err)
		}
		_, _ = fmt.Fprintf(out, "Removed config: %s\n", configPath)
		removed = true
	}

	// Remove cache directory
	cacheDir := cache.GetCacheDir()
	if info, err := os.Stat(cacheDir); err == nil && info.IsDir() {
		if err := os.RemoveAll(cacheDir); err != nil {
			return fmt.Errorf("removing cache directory: %w", err)
		}
		_, _ = fmt.Fprintf(out, "Removed cache:  %s\n", cacheDir)
		removed = true
	}

	if !removed {
		_, _ = fmt.Fprintln(out, "Nothing to remove (no config or cache found).")
	} else {
		_, _ = fmt.Fprintln(out, "Logged out successfully.")
	}

	return nil
}
