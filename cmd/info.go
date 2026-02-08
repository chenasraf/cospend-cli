package cmd

import (
	"fmt"

	"github.com/chenasraf/cospend-cli/internal/api"
	"github.com/chenasraf/cospend-cli/internal/cache"
	"github.com/chenasraf/cospend-cli/internal/config"
	"github.com/spf13/cobra"
)

// NewInfoCommand creates the info command
func NewInfoCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Show account and configuration info",
		Long:  `Show the configured Nextcloud server, authenticated user, and user locale/language.`,
		RunE:  runInfo,
	}
}

func runInfo(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	client := api.NewClient(cfg)
	client.Debug = Debug
	client.DebugWriter = cmd.ErrOrStderr()

	userInfo, ok := cache.LoadUserInfo()
	if !ok {
		userInfo, err = client.GetUserInfo()
		if err != nil {
			return fmt.Errorf("fetching user info: %w", err)
		}
		_ = cache.SaveUserInfo(userInfo)
	}

	server := config.NormalizeURL(cfg.Domain)

	out := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(out, "Server:   %s\n", server)
	_, _ = fmt.Fprintf(out, "User:     %s\n", cfg.User)
	_, _ = fmt.Fprintf(out, "Locale:   %s\n", userInfo.Locale)
	_, _ = fmt.Fprintf(out, "Language: %s\n", userInfo.Language)

	return nil
}
