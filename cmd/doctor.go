package cmd

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/chenasraf/cospend-cli/internal/api"
	"github.com/chenasraf/cospend-cli/internal/config"
	"github.com/spf13/cobra"
)

// NewDoctorCommand creates the doctor command
func NewDoctorCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check configuration, connectivity, and authentication",
		Long: `Run diagnostic checks to verify that your cospend-cli setup is working correctly.

Checks performed:
  - Config file exists and is readable
  - Required fields (domain, user, password) are set
  - Nextcloud server is reachable
  - Authentication credentials are valid
  - Default project (if configured) is accessible`,
		RunE: runDoctor,
	}
}

type checkResult struct {
	name   string
	ok     bool
	detail string
}

func runDoctor(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	out := cmd.OutOrStdout()
	var results []checkResult
	var cfg *config.Config
	var canConnect bool

	// Check 1: Config file
	configPath := config.GetConfigPath()
	if configPath == "" {
		results = append(results, checkResult{"Config file", false, "not found (run 'cospend init')"})
	} else {
		var err error
		cfg, err = config.LoadFromFile(configPath)
		if err != nil {
			results = append(results, checkResult{"Config file", false, fmt.Sprintf("error reading %s: %v", configPath, err)})
		} else {
			results = append(results, checkResult{"Config file", true, configPath})
		}
	}

	// Check 2: Required fields
	if cfg != nil {
		missing := []string{}
		if cfg.Domain == "" {
			missing = append(missing, "domain")
		}
		if cfg.User == "" {
			missing = append(missing, "user")
		}
		if cfg.Password == "" {
			missing = append(missing, "password")
		}
		if len(missing) > 0 {
			results = append(results, checkResult{"Required fields", false, fmt.Sprintf("missing: %s", joinWords(missing))})
		} else {
			results = append(results, checkResult{"Required fields", true, "domain, user, password all set"})
		}
	}

	// Check 3: Server connectivity
	if cfg != nil && cfg.Domain != "" {
		baseURL := config.NormalizeURL(cfg.Domain)
		httpClient := &http.Client{Timeout: 10 * time.Second}
		resp, err := httpClient.Get(baseURL + "/status.php")
		if err != nil {
			results = append(results, checkResult{"Server reachable", false, err.Error()})
		} else {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				results = append(results, checkResult{"Server reachable", true, baseURL})
				canConnect = true
			} else {
				results = append(results, checkResult{"Server reachable", false, fmt.Sprintf("HTTP %d from %s/status.php", resp.StatusCode, baseURL)})
			}
		}
	}

	// Check 4: Authentication
	if cfg != nil && cfg.Domain != "" && cfg.User != "" && cfg.Password != "" && canConnect {
		client := api.NewClient(cfg)
		userInfo, err := client.GetUserInfo()
		if err != nil {
			results = append(results, checkResult{"Authentication", false, fmt.Sprintf("failed: %v", err)})
		} else {
			detail := fmt.Sprintf("logged in as %s", cfg.User)
			if userInfo.Locale != "" {
				detail += fmt.Sprintf(" (locale: %s)", userInfo.Locale)
			}
			results = append(results, checkResult{"Authentication", true, detail})
		}
	}

	// Check 5: Default project
	if cfg != nil && cfg.DefaultProject != "" && canConnect {
		client := api.NewClient(cfg)
		project, err := client.GetProject(cfg.DefaultProject)
		if err != nil {
			results = append(results, checkResult{"Default project", false, fmt.Sprintf("%s: %v", cfg.DefaultProject, err)})
		} else {
			results = append(results, checkResult{"Default project", true, fmt.Sprintf("%s (%s)", cfg.DefaultProject, project.Name)})
		}
	} else if cfg != nil && cfg.DefaultProject == "" {
		results = append(results, checkResult{"Default project", true, "not configured (optional)"})
	}

	// Render results
	allOK := true
	for _, r := range results {
		if r.ok {
			_, _ = fmt.Fprintf(out, "  [ok] %-17s %s\n", r.name, r.detail)
		} else {
			_, _ = fmt.Fprintf(out, "  [!!] %-17s %s\n", r.name, r.detail)
			allOK = false
		}
	}

	_, _ = fmt.Fprintln(out)
	if allOK {
		_, _ = fmt.Fprintln(out, "All checks passed.")
	} else {
		_, _ = fmt.Fprintln(out, "Some checks failed. See above for details.")
	}

	return nil
}

func joinWords(words []string) string {
	switch len(words) {
	case 0:
		return ""
	case 1:
		return words[0]
	default:
		return strings.Join(words[:len(words)-1], ", ") + " and " + words[len(words)-1]
	}
}
