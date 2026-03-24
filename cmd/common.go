package cmd

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Debug enables debug output when true
var Debug bool

// ProjectID is the project to operate on (shared across commands)
var ProjectID string

// confirm prompts the user with a [Y/n] question and returns true if confirmed.
// Defaults to yes (empty input = yes).
func confirm(in io.Reader, out io.Writer, prompt string) bool {
	_, _ = fmt.Fprintf(out, "%s [Y/n] ", prompt)
	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		return false
	}
	answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
	return answer == "" || answer == "y" || answer == "yes"
}
