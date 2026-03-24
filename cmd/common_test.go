package cmd

import (
	"bytes"
	"strings"
	"testing"
)

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
