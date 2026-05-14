// Package output renders command results as table or JSON.
//
// Use Format to dispatch on the global -o/--output flag. JSON output goes
// to stdout (errors included) so it's pipeable to `jq`.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/mattn/go-runewidth"
)

// Format names. Use these constants to avoid stringly-typed mistakes.
const (
	FormatTable = "table"
	FormatJSON  = "json"
)

// EmptyPlaceholder is shown for nil / empty cell values in tables.
const EmptyPlaceholder = "-"

// TimeFormat is the table-mode timestamp format (local timezone).
const TimeFormat = "2006-01-02 15:04"

// Column describes a single table column.
type Column struct {
	Header string
	Width  int // display width (runewidth); 0 = auto from header
}

// Render writes a table to w. rows[i][j] is the cell value for row i, column j.
func Render(w io.Writer, cols []Column, rows [][]string) {
	// Resolve widths.
	widths := make([]int, len(cols))
	for i, c := range cols {
		widths[i] = c.Width
		if widths[i] == 0 {
			widths[i] = runewidth.StringWidth(c.Header)
		}
	}

	writeRow(w, widths, headerCells(cols))
	writeDivider(w, widths)
	for _, r := range rows {
		writeRow(w, widths, padRow(r, len(cols)))
	}
}

func headerCells(cols []Column) []string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = c.Header
	}
	return out
}

func padRow(r []string, n int) []string {
	if len(r) == n {
		return r
	}
	out := make([]string, n)
	copy(out, r)
	for i := range out {
		if out[i] == "" {
			out[i] = ""
		}
	}
	return out
}

func writeRow(w io.Writer, widths []int, cells []string) {
	var sb strings.Builder
	for i, cell := range cells {
		if cell == "" {
			cell = EmptyPlaceholder
		}
		cell = Truncate(cell, widths[i])
		sb.WriteString(cell)
		// pad to width
		pad := widths[i] - runewidth.StringWidth(cell)
		if pad > 0 {
			sb.WriteString(strings.Repeat(" ", pad))
		}
		if i < len(cells)-1 {
			sb.WriteString("  ")
		}
	}
	sb.WriteString("\n")
	_, _ = io.WriteString(w, sb.String())
}

func writeDivider(w io.Writer, widths []int) {
	var sb strings.Builder
	total := 0
	for _, x := range widths {
		total += x
	}
	total += 2 * (len(widths) - 1)
	sb.WriteString(strings.Repeat("─", total))
	sb.WriteString("\n")
	_, _ = io.WriteString(w, sb.String())
}

// ellipsis is the truncation marker. We use "…" but the width may be 1 or 2
// depending on the user's locale (go-runewidth treats it as ambiguous).
const ellipsis = '…'

// Truncate trims s so its runewidth is at most max, appending "…" if truncated.
// Returns s as-is if already short enough or if max < 1.
//
// When max is too small to even fit the ellipsis, the function falls back to
// ASCII "." or "" so the cell never exceeds max columns.
func Truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if runewidth.StringWidth(s) <= max {
		return s
	}
	ellipsisW := runewidth.RuneWidth(ellipsis)
	mark := string(ellipsis)
	// Locale may make "…" 2-wide; fall back to ASCII "." if max is tight.
	if ellipsisW > max {
		mark = "."
		ellipsisW = 1
	}
	target := max - ellipsisW
	if target < 0 {
		return mark[:max]
	}
	w := 0
	var b strings.Builder
	for _, r := range s {
		rw := runewidth.RuneWidth(r)
		if w+rw > target {
			break
		}
		b.WriteRune(r)
		w += rw
	}
	b.WriteString(mark)
	return b.String()
}

// JSON writes v to w as a JSON document, no trailing newline beyond what
// encoding/json adds. Errors writing v are non-fatal (we still attempt to
// emit at least a JSON error envelope).
func JSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// ErrorEnvelope is the JSON shape for command errors.
type ErrorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// PrintErrorJSON writes {"error":{"message":..., "code":...}} to w (stdout).
// httpOrExitCode is the HTTP status if known, otherwise the exit code.
func PrintErrorJSON(w io.Writer, message string, httpOrExitCode int) {
	_ = JSON(w, ErrorEnvelope{Error: errorBody{Message: message, Code: httpOrExitCode}})
}

// FormatUnixMillis renders a millisecond UNIX timestamp in local time.
// Zero / negative timestamps yield EmptyPlaceholder.
func FormatUnixMillis(ms int64) string {
	if ms <= 0 {
		return EmptyPlaceholder
	}
	return time.UnixMilli(ms).Local().Format(TimeFormat)
}

// Validate returns an error if format is not a recognized value.
func Validate(format string) error {
	switch format {
	case FormatTable, FormatJSON:
		return nil
	default:
		return fmt.Errorf("unknown output format %q (must be table or json)", format)
	}
}
