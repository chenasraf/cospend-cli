package cmd

import (
	"bytes"
	"strings"
	"testing"
)

// isolateConfig points config lookup at an empty directory so a config file on
// the machine running the tests can't feed settings (such as the confirm_*
// flags, which have no env var override) into the command under test.
func isolateConfig(t *testing.T) {
	t.Helper()

	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("HOME", configHome)
}

func TestConfirm(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   bool
		prompt string
	}{
		{"empty input defaults to yes", "\n", true, "Do it?"},
		{"y confirms", "y\n", true, "Do it?"},
		{"Y confirms", "Y\n", true, "Do it?"},
		{"yes confirms", "yes\n", true, "Do it?"},
		{"n declines", "n\n", false, "Do it?"},
		{"no declines", "no\n", false, "Do it?"},
		{"random text declines", "maybe\n", false, "Do it?"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := strings.NewReader(tt.input)
			var out bytes.Buffer
			got := confirm(in, &out, tt.prompt)
			if got != tt.want {
				t.Errorf("confirm(%q) = %v, want %v", tt.input, got, tt.want)
			}
			if !bytes.Contains(out.Bytes(), []byte("[Y/n]")) {
				t.Errorf("Expected [Y/n] prompt, got: %s", out.String())
			}
		})
	}
}
