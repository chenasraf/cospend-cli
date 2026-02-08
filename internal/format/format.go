package format

import (
	"strings"

	"github.com/chenasraf/cospend-cli/internal/cache"
	"golang.org/x/text/currency"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// AmountFormatter formats monetary amounts with locale-aware formatting.
type AmountFormatter struct {
	printer *message.Printer
	unit    currency.Unit
	hasUnit bool
}

// NewAmountFormatter creates a formatter for the given locale and currency name.
// locale is a string like "en_US" or "he_IL". currencyName is the project's
// currency name, which may be an ISO code (e.g. "EUR") or a symbol (e.g. "₪").
func NewAmountFormatter(locale, currencyName string) *AmountFormatter {
	// Parse locale to language tag
	tag, err := language.Parse(strings.ReplaceAll(locale, "_", "-"))
	if err != nil {
		tag = language.AmericanEnglish
	}

	f := &AmountFormatter{
		printer: message.NewPrinter(tag),
	}

	// Try to resolve currency
	if currencyName != "" {
		// Try as ISO code first
		unit, err := currency.ParseISO(strings.ToUpper(currencyName))
		if err == nil {
			f.unit = unit
			f.hasUnit = true
			return f
		}

		// Try reverse-mapping from symbol
		if iso := cache.SymbolToISO(currencyName); iso != "" {
			unit, err = currency.ParseISO(iso)
			if err == nil {
				f.unit = unit
				f.hasUnit = true
				return f
			}
		}
	}

	return f
}

// Format formats a monetary amount using locale-aware formatting.
func (f *AmountFormatter) Format(amount float64) string {
	if f.hasUnit {
		return f.printer.Sprintf("%v", currency.Symbol(f.unit.Amount(amount)))
	}
	return f.printer.Sprintf("%.2f", amount)
}
