package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

var Format = "table"

func SetFormat(f string) error {
	switch strings.ToLower(f) {
	case "table", "json":
		Format = strings.ToLower(f)
		return nil
	default:
		return fmt.Errorf("unsupported output format %q (use table or json)", f)
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

func Table(headers []string, rows [][]string) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	return w.Flush()
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
