package cmd

import (
	"fmt"
	"strconv"

	"github.com/chenasraf/cospend-cli/internal/api"
	"github.com/chenasraf/cospend-cli/internal/cache"
	"github.com/chenasraf/cospend-cli/internal/config"
	"github.com/spf13/cobra"
)

var infoCached bool

// NewInfoCommand creates the info command
func NewInfoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show account and configuration info",
		Long:  `Show the configured Nextcloud server, authenticated user, and user locale/language. When --project is set, also show project details.`,
		RunE:  runInfo,
	}

	cmd.Flags().BoolVar(&infoCached, "cached", false, "Use cached data instead of fetching fresh from API")

	return cmd
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

	var userInfo *api.UserInfo
	if infoCached {
		userInfo, _ = cache.LoadUserInfo()
	}
	if userInfo == nil {
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

	if ProjectID != "" {
		var project *api.Project
		if infoCached {
			project, _ = cache.Load(ProjectID)
		}
		if project == nil {
			project, err = client.GetProject(ProjectID)
			if err != nil {
				return fmt.Errorf("fetching project: %w", err)
			}
			if err := cache.Save(ProjectID, project); err != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to cache project: %v\n", err)
			}
		}

		_, _ = fmt.Fprintf(out, "\nProject:  %s\n", project.Name)
		_, _ = fmt.Fprintf(out, "Currency: %s\n", project.CurrencyName)

		_, _ = fmt.Fprintln(out)
		membersTable := NewTable("ID", "Name", "UserID")
		for _, m := range project.Members {
			membersTable.AddRow(strconv.Itoa(m.ID), m.Name, m.UserID)
		}
		_, _ = fmt.Fprintln(out, "Members:")
		membersTable.Render(out)

		if len(project.Categories) > 0 {
			_, _ = fmt.Fprintln(out)
			catTable := NewTable("ID", "Icon", "Name")
			for _, c := range project.Categories {
				catTable.AddRow(strconv.Itoa(c.ID), c.Icon, c.Name)
			}
			_, _ = fmt.Fprintln(out, "Categories:")
			catTable.Render(out)
		}

		if len(project.PaymentModes) > 0 {
			_, _ = fmt.Fprintln(out)
			pmTable := NewTable("ID", "Icon", "Name")
			for _, pm := range project.PaymentModes {
				pmTable.AddRow(strconv.Itoa(pm.ID), pm.Icon, pm.Name)
			}
			_, _ = fmt.Fprintln(out, "Payment Modes:")
			pmTable.Render(out)
		}

		if len(project.Currencies) > 0 {
			_, _ = fmt.Fprintln(out)
			currTable := NewTable("ID", "Name", "Exchange Rate")
			for _, cur := range project.Currencies {
				currTable.AddRow(strconv.Itoa(cur.ID), cur.Name, strconv.FormatFloat(cur.ExchangeRate, 'f', -1, 64))
			}
			_, _ = fmt.Fprintln(out, "Currencies:")
			currTable.Render(out)
		}
	}

	return nil
}
