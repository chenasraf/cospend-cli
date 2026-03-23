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
	cmd.AddCommand(newConfigListCommand())

	return cmd
}

func newConfigSetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long: `Set a configuration value.

Supported keys:
  domain             Nextcloud instance URL
  user               Nextcloud username
  default-project    Default project ID (used when -p is not specified)

Examples:
  cospend config set domain https://cloud.example.com
  cospend config set user alice
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
  domain             Nextcloud instance URL
  user               Nextcloud username
  default-project    Default project ID (used when -p is not specified)

Examples:
  cospend config get domain
  cospend config get user
  cospend config get default-project`,
		Args: cobra.ExactArgs(1),
		RunE: runConfigGet,
	}
}

func newConfigListCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "Show current configuration",
		Long: `Show the config file path and all configuration values.
Passwords are masked for security.

Examples:
  cospend config list`,
		Args: cobra.NoArgs,
		RunE: runConfigList,
	}
}

func maskPassword(password string) string {
	if password == "" {
		return "(not set)"
	}
	return "********"
}

func runConfigList(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	configPath := config.GetConfigPath()
	if configPath == "" {
		return fmt.Errorf("no config file found (run 'cospend init' first)")
	}

	cfg, err := config.LoadFromFile(configPath)
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	out := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(out, "Config file: %s\n", configPath)
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintf(out, "  domain:          %s\n", cfg.Domain)
	_, _ = fmt.Fprintf(out, "  user:            %s\n", cfg.User)
	_, _ = fmt.Fprintf(out, "  password:        %s\n", maskPassword(cfg.Password))
	if cfg.DefaultProject != "" {
		_, _ = fmt.Fprintf(out, "  default-project: %s\n", cfg.DefaultProject)
	}

	return nil
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
	case "domain":
		cfg.Domain = config.NormalizeURL(value)
	case "user":
		cfg.User = value
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

	var value string
	switch key {
	case "domain":
		value = cfg.Domain
	case "user":
		value = cfg.User
	case "default-project":
		value = cfg.DefaultProject
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}

	if value == "" {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "(not set)")
	} else {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), value)
	}

	return nil
}
