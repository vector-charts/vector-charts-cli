package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"
)

var Format = "table"

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

// Pretty table: aligned columns with a header rule.
// Plaintext: tab-separated header + rows (script-friendly).
func Table(headers []string, rows [][]string) error {
	if Format == "plaintext" {
		return plainTable(headers, rows)
	}
	return prettyTable(headers, rows)
}

func plainTable(headers []string, rows [][]string) error {
	fmt.Fprintln(os.Stdout, strings.Join(headers, "\t"))
	for _, row := range rows {
		cells := make([]string, len(headers))
		for i := range headers {
			if i < len(row) {
				cells[i] = row[i]
			}
		}
		fmt.Fprintln(os.Stdout, strings.Join(cells, "\t"))
	}
	return nil
}

func prettyTable(headers []string, rows [][]string) error {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = utf8.RuneCountInString(h)
	}
	normalized := make([][]string, len(rows))
	for r, row := range rows {
		cells := make([]string, len(headers))
		for i := range headers {
			if i < len(row) {
				cells[i] = row[i]
			}
			w := utf8.RuneCountInString(cells[i])
			if w > widths[i] {
				widths[i] = w
			}
		}
		normalized[r] = cells
	}

	printPrettyRow(headers, widths)
	printPrettyRule(widths)
	for _, row := range normalized {
		printPrettyRow(row, widths)
	}
	return nil
}

func printPrettyRow(cells []string, widths []int) {
	parts := make([]string, len(widths))
	for i, w := range widths {
		cell := ""
		if i < len(cells) {
			cell = cells[i]
		}
		pad := w - utf8.RuneCountInString(cell)
		if pad < 0 {
			pad = 0
		}
		parts[i] = cell + strings.Repeat(" ", pad)
	}
	fmt.Fprintln(os.Stdout, strings.Join(parts, "  "))
}

func printPrettyRule(widths []int) {
	parts := make([]string, len(widths))
	for i, w := range widths {
		parts[i] = strings.Repeat("─", w)
	}
	fmt.Fprintln(os.Stdout, strings.Join(parts, "  "))
}

func Println(a ...any) {
	fmt.Fprintln(os.Stdout, a...)
}

func Printf(format string, a ...any) {
	fmt.Fprintf(os.Stdout, format, a...)
}

func Errf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format, a...)
}
