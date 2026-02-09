package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/mattn/go-runewidth"
)

// Table border characters
const (
	borderHorizontal  = "─"
	borderVertical    = "│"
	borderTopLeft     = "┌"
	borderTopRight    = "┐"
	borderBottomLeft  = "└"
	borderBottomRight = "┘"
	borderTopMid      = "┬"
	borderBottomMid   = "┴"
	borderLeftMid     = "├"
	borderRightMid    = "┤"
	borderCross       = "┼"
)

// Table handles formatted table output
type Table struct {
	headers   []string
	rows      [][]string
	colWidths []int
}

// NewTable creates a new table with the given headers
func NewTable(headers ...string) *Table {
	colWidths := make([]int, len(headers))
	for i, h := range headers {
		colWidths[i] = runewidth.StringWidth(h)
	}
	return &Table{
		headers:   headers,
		colWidths: colWidths,
	}
}

// AddRow adds a row to the table
func (t *Table) AddRow(values ...string) {
	// Pad with empty strings if needed
	for len(values) < len(t.headers) {
		values = append(values, "")
	}
	// Truncate if too many
	if len(values) > len(t.headers) {
		values = values[:len(t.headers)]
	}

	t.rows = append(t.rows, values)

	// Update column widths
	for i, v := range values {
		if w := runewidth.StringWidth(v); w > t.colWidths[i] {
			t.colWidths[i] = w
		}
	}
}

// Render writes the table to the given writer
func (t *Table) Render(w io.Writer) {
	t.printBorder(w, borderTopLeft, borderTopMid, borderTopRight)
	t.printRow(w, t.headers)
	t.printBorder(w, borderLeftMid, borderCross, borderRightMid)

	for _, row := range t.rows {
		t.printRow(w, row)
	}

	t.printBorder(w, borderBottomLeft, borderBottomMid, borderBottomRight)
}

func (t *Table) printBorder(w io.Writer, left, mid, right string) {
	_, _ = fmt.Fprint(w, left)
	for i, width := range t.colWidths {
		_, _ = fmt.Fprint(w, strings.Repeat(borderHorizontal, width+2))
		if i < len(t.colWidths)-1 {
			_, _ = fmt.Fprint(w, mid)
		}
	}
	_, _ = fmt.Fprintln(w, right)
}

func (t *Table) printRow(w io.Writer, values []string) {
	_, _ = fmt.Fprint(w, borderVertical)
	for i, val := range values {
		padded := runewidth.FillRight(val, t.colWidths[i])
		_, _ = fmt.Fprintf(w, " %s %s", padded, borderVertical)
	}
	_, _ = fmt.Fprintln(w)
}
