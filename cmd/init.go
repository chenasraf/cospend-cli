package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

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
	var overwritePath string
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
		overwritePath = existingPath
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Setting up Cospend CLI configuration...")
	_, _ = fmt.Fprintln(cmd.OutOrStdout())

	// Prompt for domain
	domain, err := promptString(cmd, "Nextcloud domain (e.g., cloud.example.com)")
	if err != nil {
		return err
	}
	domain = strings.TrimRight(domain, "/")

	// Auto-prepend https:// if no scheme provided
	domainLower := strings.ToLower(domain)
	if !strings.HasPrefix(domainLower, "http://") && !strings.HasPrefix(domainLower, "https://") {
		domain = "https://" + domain
	}

	// Choose login method
	_, _ = fmt.Fprintln(cmd.OutOrStdout())
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Choose login method:")

	options := []selectOption{
		{label: "Browser login (recommended)", description: "Opens browser for secure authentication"},
		{label: "Password/App token", description: "Enter credentials manually"},
	}

	selected, err := promptSelect(cmd, options)
	if err != nil {
		return err
	}

	var cfg *config.Config

	switch selected {
	case 0:
		cfg, err = loginFlowAuth(cmd, domain)
		if err != nil {
			return err
		}
	case 1:
		cfg, err = passwordAuth(cmd, domain)
		if err != nil {
			return err
		}
	}

	var path string
	if overwritePath != "" {
		path, err = config.SaveToPath(cfg, overwritePath)
	} else {
		path, err = config.Save(cfg, configFormat)
	}
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

// selectOption represents an option in a select prompt
type selectOption struct {
	label       string
	description string
}

// promptSelect displays an interactive select menu and returns the selected index
func promptSelect(cmd *cobra.Command, options []selectOption) (int, error) {
	// Check if we're in a terminal
	f, ok := cmd.InOrStdin().(*os.File)
	if !ok || !term.IsTerminal(int(f.Fd())) {
		// Fallback to simple numbered input for non-terminal
		return promptSelectFallback(cmd, options)
	}

	selected := 0
	out := cmd.OutOrStdout()

	// Save terminal state and set raw mode
	oldState, err := term.MakeRaw(int(f.Fd()))
	if err != nil {
		return promptSelectFallback(cmd, options)
	}
	defer func() { _ = term.Restore(int(f.Fd()), oldState) }()

	// Hide cursor
	_, _ = fmt.Fprint(out, "\033[?25l")
	defer func() { _, _ = fmt.Fprint(out, "\033[?25h") }() // Show cursor on exit

	renderOptions := func() {
		for i, opt := range options {
			if i == selected {
				_, _ = fmt.Fprintf(out, "\r\033[K  \033[36m>\033[0m \033[1m%s\033[0m - %s\n", opt.label, opt.description)
			} else {
				_, _ = fmt.Fprintf(out, "\r\033[K    %s - %s\n", opt.label, opt.description)
			}
		}
	}

	// Move cursor up helper
	moveUp := func(n int) {
		if n > 0 {
			_, _ = fmt.Fprintf(out, "\033[%dA", n)
		}
	}

	renderOptions()

	buf := make([]byte, 3)
	for {
		moveUp(len(options))
		renderOptions()

		n, err := f.Read(buf)
		if err != nil {
			return 0, err
		}

		// Handle input
		if n == 1 {
			switch buf[0] {
			case 13, 10: // Enter
				_, _ = fmt.Fprintln(out)
				return selected, nil
			case 3: // Ctrl+C
				_, _ = fmt.Fprintln(out)
				return 0, fmt.Errorf("cancelled")
			case 'j', 'J': // vim down
				selected = (selected + 1) % len(options)
			case 'k', 'K': // vim up
				selected = (selected - 1 + len(options)) % len(options)
			}
		} else if n == 3 && buf[0] == 27 && buf[1] == 91 {
			// Arrow keys: ESC [ A/B
			switch buf[2] {
			case 65: // Up
				selected = (selected - 1 + len(options)) % len(options)
			case 66: // Down
				selected = (selected + 1) % len(options)
			}
		}
	}
}

// promptSelectFallback is a simple numbered fallback for non-terminals
func promptSelectFallback(cmd *cobra.Command, options []selectOption) (int, error) {
	out := cmd.OutOrStdout()
	for i, opt := range options {
		_, _ = fmt.Fprintf(out, "  %d. %s - %s\n", i+1, opt.label, opt.description)
	}
	_, _ = fmt.Fprintln(out)

	choice, err := promptString(cmd, "Enter choice [1]")
	if err != nil {
		return 0, err
	}
	if choice == "" {
		return 0, nil
	}

	idx := 0
	if _, err := fmt.Sscanf(choice, "%d", &idx); err != nil || idx < 1 || idx > len(options) {
		return 0, fmt.Errorf("invalid choice: %s", choice)
	}
	return idx - 1, nil
}

// passwordAuth handles traditional password/app token authentication
func passwordAuth(cmd *cobra.Command, domain string) (*config.Config, error) {
	// Prompt for username
	user, err := promptString(cmd, "Username")
	if err != nil {
		return nil, err
	}

	// Prompt for password (hidden input)
	password, err := promptPassword(cmd, "Password (or app token)")
	if err != nil {
		return nil, err
	}

	return &config.Config{
		Domain:   domain,
		User:     user,
		Password: password,
	}, nil
}

// loginFlowResponse represents the initial login flow response
type loginFlowResponse struct {
	Poll struct {
		Token    string `json:"token"`
		Endpoint string `json:"endpoint"`
	} `json:"poll"`
	Login string `json:"login"`
}

// loginFlowResult represents the successful poll response
type loginFlowResult struct {
	Server      string `json:"server"`
	LoginName   string `json:"loginName"`
	AppPassword string `json:"appPassword"`
}

const userAgent = "Cospend CLI"

// loginFlowAuth handles Nextcloud Login Flow v2 authentication
func loginFlowAuth(cmd *cobra.Command, domain string) (*config.Config, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	// Step 1: Initiate login flow
	loginURL := domain + "/index.php/login/v2"
	req, err := http.NewRequest("POST", loginURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("initiating login flow: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("login flow initiation failed with status %d", resp.StatusCode)
	}

	var flowResp loginFlowResponse
	if err := json.NewDecoder(resp.Body).Decode(&flowResp); err != nil {
		return nil, fmt.Errorf("parsing login flow response: %w", err)
	}

	// Step 2: Open browser for user to authenticate
	_, _ = fmt.Fprintln(cmd.OutOrStdout())
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Opening browser for authentication...")
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "If the browser doesn't open, visit this URL manually:")
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), flowResp.Login)
	_, _ = fmt.Fprintln(cmd.OutOrStdout())

	if err := openBrowser(flowResp.Login); err != nil {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: couldn't open browser: %v\n", err)
	}

	// Step 3: Poll for authentication result
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Waiting for authentication...")

	result, err := pollForLogin(flowResp.Poll.Endpoint, flowResp.Poll.Token)
	if err != nil {
		return nil, err
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Authentication successful!")

	// Use the server from the response (in case of redirects) or fall back to original domain
	serverDomain := result.Server
	if serverDomain == "" {
		serverDomain = domain
	}

	return &config.Config{
		Domain:   serverDomain,
		User:     result.LoginName,
		Password: result.AppPassword,
	}, nil
}

// pollForLogin polls the login endpoint until authentication completes or times out
func pollForLogin(endpoint, token string) (*loginFlowResult, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	deadline := time.Now().Add(20 * time.Minute) // Token valid for 20 minutes

	for time.Now().Before(deadline) {
		data := url.Values{}
		data.Set("token", token)

		req, err := http.NewRequest("POST", endpoint, strings.NewReader(data.Encode()))
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("User-Agent", userAgent)

		resp, err := client.Do(req)
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			var result loginFlowResult
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				_ = resp.Body.Close()
				return nil, fmt.Errorf("parsing login result: %w", err)
			}
			_ = resp.Body.Close()
			return &result, nil
		}

		_ = resp.Body.Close()

		// 404 means still waiting for user to authenticate
		if resp.StatusCode == http.StatusNotFound {
			time.Sleep(2 * time.Second)
			continue
		}

		return nil, fmt.Errorf("unexpected status during polling: %d", resp.StatusCode)
	}

	return nil, fmt.Errorf("authentication timed out (20 minutes)")
}

// openBrowser is a function variable to allow mocking in tests
var openBrowser = openBrowserDefault

// openBrowserDefault opens the given URL in the default browser
func openBrowserDefault(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}
