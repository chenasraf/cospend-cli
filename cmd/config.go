package cmd

import (
	"fmt"

	"github.com/chenasraf/cospend-cli/internal/config"
	"github.com/spf13/cobra"
)

// NewConfigCommand creates the config command with subcommands
func NewConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage CLI configuration",
		Long:  `View and modify cospend-cli configuration settings.`,
	}

	cmd.AddCommand(newConfigSetCommand())
	cmd.AddCommand(newConfigGetCommand())

	return cmd
}

func newConfigSetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long: `Set a configuration value.

Supported keys:
  default-project    Default project ID (used when -p is not specified)

Examples:
  cospend config set default-project myproject`,
		Args: cobra.ExactArgs(2),
		RunE: runConfigSet,
	}
}

func newConfigGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Long: `Get a configuration value.

Supported keys:
  default-project    Default project ID (used when -p is not specified)

Examples:
  cospend config get default-project`,
		Args: cobra.ExactArgs(1),
		RunE: runConfigGet,
	}
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	cmd.SilenceUsage = true

	configPath := config.GetConfigPath()
	if configPath == "" {
		return fmt.Errorf("no config file found (run 'cospend init' first)")
	}

	cfg, err := config.LoadFromFile(configPath)
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	switch key {
	case "default-project":
		cfg.DefaultProject = value
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}

	if _, err := config.SaveToPath(cfg, configPath); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Set %s = %s\n", key, value)
	return nil
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	key := args[0]

	cmd.SilenceUsage = true

	configPath := config.GetConfigPath()
	if configPath == "" {
		return fmt.Errorf("no config file found (run 'cospend init' first)")
	}

	cfg, err := config.LoadFromFile(configPath)
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	switch key {
	case "default-project":
		if cfg.DefaultProject == "" {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "(not set)")
		} else {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), cfg.DefaultProject)
		}
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}

	return nil
}
