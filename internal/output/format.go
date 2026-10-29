package output

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Str renders a JSON-decoded value for table/plaintext cells.
func Str(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return fmt.Sprint(v)
	}
}

// EpochMS formats an epoch-milliseconds timestamp (or RFC3339 string) as YYYY-MM-DD.
func EpochMS(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case float64:
		if t <= 0 {
			return ""
		}
		return time.UnixMilli(int64(t)).UTC().Format("2006-01-02")
	case int64:
		if t <= 0 {
			return ""
		}
		return time.UnixMilli(t).UTC().Format("2006-01-02")
	case string:
		if t == "" {
			return ""
		}
		if ms, err := strconv.ParseInt(t, 10, 64); err == nil {
			return time.UnixMilli(ms).UTC().Format("2006-01-02")
		}
		if parsed, err := time.Parse(time.RFC3339, t); err == nil {
			return parsed.UTC().Format("2006-01-02")
		}
		return t
	default:
		return Str(v)
	}
}

// Boolish prefers the first non-nil bool-like value; missing values are "false".
func Boolish(vals ...any) string {
	if s := OptionalBool(vals...); s != "" {
		return s
	}
	return "false"
}

// OptionalBool is like Boolish but returns "" when every value is nil.
func OptionalBool(vals ...any) string {
	for _, v := range vals {
		if v == nil {
			continue
		}
		switch t := v.(type) {
		case bool:
			return Str(t)
		case string:
			if t != "" {
				return t
			}
		default:
			return Str(v)
		}
	}
	return ""
}

// Price formats a store price object: {amount, currency, interval}.
func Price(v any) string {
	m, ok := v.(map[string]any)
	if !ok || m == nil {
		return ""
	}
	amount, ok := m["amount"].(float64)
	if !ok {
		return ""
	}
	s := fmt.Sprintf("$%.2f", amount/100)
	if interval, _ := m["interval"].(string); interval != "" {
		s += "/" + interval
	}
	return s
}

// ExtractList pulls the first matching array key (or the value itself if already a list).
func ExtractList(v any, keys ...string) []any {
	switch t := v.(type) {
	case []any:
		return t
	case map[string]any:
		for _, k := range keys {
			if arr, ok := t[k].([]any); ok {
				return arr
			}
		}
	}
	return nil
}

// FirstNonEmpty returns the first non-nil value.
func FirstNonEmpty(vals ...any) any {
	for _, v := range vals {
		if v != nil {
			return v
		}
	}
	return nil
}

// CellsLabel matches the portal license list: single cell name, or "N cells".
func CellsLabel(permit map[string]any) string {
	if permit == nil {
		return ""
	}
	if cells, ok := permit["cells"].([]any); ok {
		if len(cells) == 1 {
			return Str(cells[0])
		}
		if len(cells) > 1 {
			if n, ok := permit["cellCount"].(float64); ok && n > 0 {
				return fmt.Sprintf("%d cells", int(n))
			}
			return fmt.Sprintf("%d cells", len(cells))
		}
	}
	if n, ok := permit["cellCount"].(float64); ok && n > 0 {
		if n == 1 {
			return Str(permit["name"])
		}
		return fmt.Sprintf("%d cells", int(n))
	}
	return Str(permit["name"])
}

// JoinStrings joins non-empty stringified values.
func JoinStrings(sep string, vals ...any) string {
	parts := make([]string, 0, len(vals))
	for _, v := range vals {
		s := Str(v)
		if s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, sep)
}
