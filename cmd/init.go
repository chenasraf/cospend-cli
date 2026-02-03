package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/chenasraf/cospend-cli/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var configFormat string

// NewInitCommand creates the init command
func NewInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize configuration file",
		Long: `Initialize a configuration file with your Nextcloud credentials.

This command will interactively prompt for your Nextcloud domain, username,
and password, then save them to a config file.

Config file location:
  Linux:   ~/.config/cospend/cospend.{ext}
  macOS:   ~/Library/Application Support/cospend/cospend.{ext}
  Windows: %APPDATA%\cospend\cospend.{ext}`,
		RunE: runInit,
	}

	cmd.Flags().StringVarP(&configFormat, "format", "f", "json", "Config file format (json, yaml, toml)")

	return cmd
}

func runInit(cmd *cobra.Command, _ []string) error {
	// Validate format
	switch configFormat {
	case "json", "yaml", "yml", "toml":
		// valid
	default:
		return fmt.Errorf("unsupported format: %s (use json, yaml, or toml)", configFormat)
	}

	// Parameters validated, silence usage for subsequent errors
	cmd.SilenceUsage = true

	// Check if config already exists
	if existingPath := config.GetConfigPath(); existingPath != "" {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Config file already exists: %s\n", existingPath)
		overwrite, err := promptYesNo(cmd, "Overwrite?")
		if err != nil {
			return err
		}
		if !overwrite {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
			return nil
		}
		// Remove existing config
		if err := os.Remove(existingPath); err != nil {
			return fmt.Errorf("removing existing config: %w", err)
		}
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Setting up Cospend CLI configuration...")
	_, _ = fmt.Fprintln(cmd.OutOrStdout())

	// Prompt for domain
	domain, err := promptString(cmd, "Nextcloud domain (e.g., https://cloud.example.com)")
	if err != nil {
		return err
	}
	domain = strings.TrimRight(domain, "/")

	// Prompt for username
	user, err := promptString(cmd, "Username")
	if err != nil {
		return err
	}

	// Prompt for password (hidden input)
	password, err := promptPassword(cmd, "Password (or app token)")
	if err != nil {
		return err
	}

	cfg := &config.Config{
		Domain:   domain,
		User:     user,
		Password: password,
	}

	path, err := config.Save(cfg, configFormat)
	if err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout())
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Configuration saved to: %s\n", path)
	_, _ = fmt.Fprintln(cmd.OutOrStdout())
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "You can now use cospend commands without setting environment variables.")

	return nil
}

func promptString(cmd *cobra.Command, prompt string) (string, error) {
	reader := bufio.NewReader(cmd.InOrStdin())
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s: ", prompt)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

func promptPassword(cmd *cobra.Command, prompt string) (string, error) {
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s: ", prompt)

	// Try to read password with hidden input
	if f, ok := cmd.InOrStdin().(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		password, err := term.ReadPassword(int(f.Fd()))
		_, _ = fmt.Fprintln(cmd.OutOrStdout()) // Print newline after hidden input
		if err != nil {
			return "", err
		}
		return string(password), nil
	}

	// Fallback to regular input (for non-terminal/testing)
	reader := bufio.NewReader(cmd.InOrStdin())
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

func promptYesNo(cmd *cobra.Command, prompt string) (bool, error) {
	reader := bufio.NewReader(cmd.InOrStdin())
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s [y/N]: ", prompt)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes", nil
}
