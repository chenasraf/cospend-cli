package format

import (
	"testing"
)

func TestNewAmountFormatterISO(t *testing.T) {
	f := NewAmountFormatter("en_US", "USD")
	if !f.hasUnit {
		t.Error("Expected hasUnit=true for ISO code USD")
	}
}

func TestNewAmountFormatterSymbol(t *testing.T) {
	f := NewAmountFormatter("en_US", "€")
	if !f.hasUnit {
		t.Error("Expected hasUnit=true for symbol €")
	}
}

func TestNewAmountFormatterUnknown(t *testing.T) {
	f := NewAmountFormatter("en_US", "XYZ123")
	if f.hasUnit {
		t.Error("Expected hasUnit=false for unknown currency")
	}
}

func TestNewAmountFormatterEmpty(t *testing.T) {
	f := NewAmountFormatter("en_US", "")
	if f.hasUnit {
		t.Error("Expected hasUnit=false for empty currency")
	}
}

func TestFormatWithCurrency(t *testing.T) {
	tests := []struct {
		name     string
		locale   string
		currency string
		amount   float64
		contains string
	}{
		{"USD en_US", "en_US", "USD", 1234.50, "$"},
		{"EUR de_DE", "de_DE", "EUR", 1234.50, "€"},
		{"ILS he_IL", "he_IL", "ILS", 1234.50, "₪"},
		{"symbol lookup", "en_US", "₪", 50.00, "₪"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewAmountFormatter(tt.locale, tt.currency)
			result := f.Format(tt.amount)
			if result == "" {
				t.Error("Format returned empty string")
			}
			if tt.contains != "" && !containsStr(result, tt.contains) {
				t.Errorf("Format(%v) = %q, want it to contain %q", tt.amount, result, tt.contains)
			}
		})
	}
}

func TestFormatWithoutCurrency(t *testing.T) {
	f := NewAmountFormatter("en_US", "")
	result := f.Format(1234.50)
	if result != "1,234.50" {
		t.Errorf("Format(1234.50) = %q, want %q", result, "1,234.50")
	}
}

func TestFormatFallbackLocale(t *testing.T) {
	// Invalid locale should fall back to en-US
	f := NewAmountFormatter("invalid!!!", "")
	result := f.Format(1234.50)
	if result != "1,234.50" {
		t.Errorf("Format(1234.50) with invalid locale = %q, want %q", result, "1,234.50")
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
