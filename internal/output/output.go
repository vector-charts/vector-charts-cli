package output

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"

	"golang.org/x/term"
)

var Format = "table"

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func SetFormat(f string) error {
	switch strings.ToLower(f) {
	case "table", "json", "plaintext":
		Format = strings.ToLower(f)
		return nil
	default:
		return fmt.Errorf("unsupported output format %q (use table, json, or plaintext)", f)
	}
}

func JSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func PrintJSONOr(v any, tableFn func() error) error {
	if Format == "json" {
		return JSON(v)
	}
	return tableFn()
}

// ColorEnabled follows https://no-color.org/ and common CLI conventions:
// disabled for plaintext output, when NO_COLOR is set, CLICOLOR=0, TERM=dumb,
// or stdout is not a TTY.
func ColorEnabled() bool {
	if Format == "plaintext" {
		return false
	}
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	if os.Getenv("CLICOLOR") == "0" {
		return false
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func Green(s string) string {
	return colorize(s, "32")
}

func Red(s string) string {
	return colorize(s, "31")
}

func Yellow(s string) string {
	return colorize(s, "33")
}

func Cyan(s string) string {
	return colorize(s, "36")
}

// PaleGrey is bright-black text for status lines and table borders.
func PaleGrey(s string) string {
	return colorize(s, "90")
}

// Grey is a light-grey status color (ANSI white), clearer than PaleGrey for state labels.
func Grey(s string) string {
	return colorize(s, "37")
}

// ColorState colors status / enum labels for table output.
func ColorState(status string) string {
	if status == "" {
		return status
	}
	lower := strings.ToLower(strings.TrimSpace(status))
	switch {
	case strings.HasPrefix(lower, "online"):
		return Green(status)
	case strings.HasPrefix(lower, "offline"):
		return Red(status)
	case strings.Contains(lower, "pending"):
		return Grey(status)
	}
	switch {
	case strings.HasPrefix(lower, "ok"), lower == "succeeded", lower == "completed", lower == "success",
		lower == "ready", lower == "true", lower == "enabled", lower == "yes", lower == "subscribed", lower == "active":
		return Green(status)
	case strings.HasPrefix(lower, "failed"), lower == "error", lower == "denied", lower == "false",
		lower == "disabled", lower == "no":
		return Red(status)
	case lower == "warning", lower == "degraded":
		return Yellow(status)
	case lower == "processing", lower == "running", lower == "uploading", lower == "downloading", lower == "syncing":
		return Cyan(status)
	case lower == "uploaded", lower == "queued", lower == "cancelled", lower == "canceled",
		lower == "scheduled", lower == "idle", lower == "inactive", lower == "none", lower == "unknown":
		return Grey(status)
	default:
		return status
	}
}

func colorize(s, code string) string {
	if !ColorEnabled() || s == "" {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func stripANSI(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}

func visibleLen(s string) int {
	return utf8.RuneCountInString(stripANSI(s))
}

// Connecting prints a pale-grey status line before table output.
// Skipped for json/plaintext so machine-readable stdout stays clean.
func Connecting(profileName string) {
	if Format == "json" || Format == "plaintext" || profileName == "" {
		return
	}
	_, _ = fmt.Fprintln(os.Stdout, PaleGrey(fmt.Sprintf("connecting to server '%s'...", profileName)))
}

// Table prints a bordered grid with grey column dividers and a header rule.
// In plaintext mode, prints the same columns as comma-separated rows (no color).
// Data rows are not separated. Pass nil/empty headers for a headerless list.
func Table(headers []string, rows [][]string) error {
	if Format == "plaintext" {
		return plaintextTable(headers, rows)
	}
	return prettyTable(headers, rows, false)
}

// TableSeparated is like Table but draws a rule between each data row.
// In plaintext mode this matches Table (comma-separated rows).
func TableSeparated(headers []string, rows [][]string) error {
	if Format == "plaintext" {
		return plaintextTable(headers, rows)
	}
	return prettyTable(headers, rows, true)
}

func columnCount(headers []string, rows [][]string) int {
	n := len(headers)
	for _, row := range rows {
		if len(row) > n {
			n = len(row)
		}
	}
	return n
}

func cellLines(cell string) []string {
	if cell == "" {
		return []string{""}
	}
	return strings.Split(cell, "\n")
}

func cellWidth(cell string) int {
	max := 0
	for _, line := range cellLines(cell) {
		if w := visibleLen(line); w > max {
			max = w
		}
	}
	return max
}

func plaintextTable(headers []string, rows [][]string) error {
	cols := columnCount(headers, rows)
	if cols == 0 {
		return nil
	}
	writeRow := func(cells []string) {
		parts := make([]string, cols)
		for i := 0; i < cols; i++ {
			cell := ""
			if i < len(cells) {
				cell = cells[i]
			}
			parts[i] = csvField(stripANSI(cell))
		}
		_, _ = fmt.Fprintln(os.Stdout, strings.Join(parts, ","))
	}
	if len(headers) > 0 {
		padded := make([]string, cols)
		copy(padded, headers)
		writeRow(padded)
	}
	for _, row := range rows {
		cells := make([]string, cols)
		for i := 0; i < cols; i++ {
			if i < len(row) {
				cells[i] = flattenCell(row[i])
			}
		}
		writeRow(cells)
	}
	return nil
}

func flattenCell(cell string) string {
	cell = stripANSI(cell)
	cell = strings.ReplaceAll(cell, "\r\n", " ")
	cell = strings.ReplaceAll(cell, "\n", " ")
	cell = strings.ReplaceAll(cell, "\r", " ")
	return strings.TrimSpace(cell)
}

func csvField(s string) string {
	s = flattenCell(s)
	if strings.ContainsAny(s, ",\"") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}

func prettyTable(headers []string, rows [][]string, separateRows bool) error {
	cols := columnCount(headers, rows)
	if cols == 0 {
		return nil
	}
	widths := make([]int, cols)
	for i, h := range headers {
		widths[i] = visibleLen(h)
	}
	normalized := make([][]string, len(rows))
	for r, row := range rows {
		cells := make([]string, cols)
		for i := 0; i < cols; i++ {
			if i < len(row) {
				cells[i] = row[i]
			}
			if w := cellWidth(cells[i]); w > widths[i] {
				widths[i] = w
			}
		}
		normalized[r] = cells
	}

	hasHeader := len(headers) > 0
	_, _ = fmt.Fprintln(os.Stdout, borderLine(widths, "┌", "┬", "┐"))
	if hasHeader {
		padded := make([]string, cols)
		copy(padded, headers)
		printBorderedRow(padded, widths)
		_, _ = fmt.Fprintln(os.Stdout, borderLine(widths, "├", "┼", "┤"))
	}
	for i, row := range normalized {
		if separateRows && i > 0 {
			_, _ = fmt.Fprintln(os.Stdout, borderLine(widths, "├", "┼", "┤"))
		}
		printBorderedMultilineRow(row, widths)
	}
	_, _ = fmt.Fprintln(os.Stdout, borderLine(widths, "└", "┴", "┘"))
	return nil
}

func borderLine(widths []int, left, mid, right string) string {
	parts := make([]string, len(widths))
	for i, w := range widths {
		// +2 for the padding spaces inside each cell
		parts[i] = strings.Repeat("─", w+2)
	}
	return PaleGrey(left + strings.Join(parts, mid) + right)
}

func printBorderedRow(cells []string, widths []int) {
	parts := make([]string, len(widths))
	vbar := PaleGrey("│")
	for i, w := range widths {
		cell := ""
		if i < len(cells) {
			cell = cells[i]
		}
		pad := w - visibleLen(cell)
		if pad < 0 {
			pad = 0
		}
		parts[i] = " " + cell + strings.Repeat(" ", pad) + " "
	}
	_, _ = fmt.Fprintln(os.Stdout, vbar+strings.Join(parts, vbar)+vbar)
}

// Expand a logical row whose cells may contain newlines into one printed
// border line per content line. Shorter cells pad with blanks.
func printBorderedMultilineRow(cells []string, widths []int) {
	split := make([][]string, len(widths))
	maxLines := 1
	for i := range widths {
		cell := ""
		if i < len(cells) {
			cell = cells[i]
		}
		lines := cellLines(cell)
		split[i] = lines
		if len(lines) > maxLines {
			maxLines = len(lines)
		}
	}
	for line := 0; line < maxLines; line++ {
		row := make([]string, len(widths))
		for i := range widths {
			if line < len(split[i]) {
				row[i] = split[i][line]
			}
		}
		printBorderedRow(row, widths)
	}
}

func Println(a ...any) {
	_, _ = fmt.Fprintln(os.Stdout, a...)
}

func Printf(format string, a ...any) {
	_, _ = fmt.Fprintf(os.Stdout, format, a...)
}

func Errf(format string, a ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format, a...)
}
