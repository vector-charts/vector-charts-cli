package output

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Str renders a JSON-decoded value for table cells.
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

// Time formats a timestamp (RFC3339 or epoch milliseconds) as a relative human string.
func Time(v any) string {
	t, ok := parseTime(v)
	if !ok {
		return ""
	}
	return formatRelative(t, time.Now())
}

// EpochMS formats a timestamp as YYYY-MM-DD (UTC). Prefer Time for table output.
func EpochMS(v any) string {
	t, ok := parseTime(v)
	if !ok {
		if v == nil {
			return ""
		}
		if s, isStr := v.(string); isStr && s != "" {
			return s
		}
		return Str(v)
	}
	return t.UTC().Format("2006-01-02")
}

func parseTime(v any) (time.Time, bool) {
	if v == nil {
		return time.Time{}, false
	}
	switch t := v.(type) {
	case float64:
		if t <= 0 {
			return time.Time{}, false
		}
		return time.UnixMilli(int64(t)), true
	case int64:
		if t <= 0 {
			return time.Time{}, false
		}
		return time.UnixMilli(t), true
	case string:
		if t == "" {
			return time.Time{}, false
		}
		if ms, err := strconv.ParseInt(t, 10, 64); err == nil {
			if ms <= 0 {
				return time.Time{}, false
			}
			return time.UnixMilli(ms), true
		}
		if parsed, err := time.Parse(time.RFC3339, t); err == nil {
			return parsed, true
		}
		if parsed, err := time.Parse(time.RFC3339Nano, t); err == nil {
			return parsed, true
		}
		return time.Time{}, false
	default:
		return time.Time{}, false
	}
}

func formatRelative(t, now time.Time) string {
	d := now.Sub(t)
	future := d < 0
	if future {
		d = -d
	}
	label := formatAgo(d)
	if future {
		return "in " + label
	}
	return label + " ago"
}

func formatAgo(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		sec := int(d.Seconds())
		if sec < 1 {
			sec = 1
		}
		return fmt.Sprintf("%ds", sec)
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		days := int(d.Hours() / 24)
		if days < 1 {
			days = 1
		}
		return fmt.Sprintf("%dd", days)
	}
}

// Duration formats a positive duration as a short label (e.g. "3m", "2h").
func Duration(d time.Duration) string {
	return formatAgo(d)
}

// EstimateRemaining extrapolates time left from percent-complete and elapsed time.
// Returns "" when percent is too low/high to estimate.
func EstimateRemaining(percent float64, startedAt any) string {
	if percent <= 0 || percent >= 100 {
		return ""
	}
	started, ok := parseTime(startedAt)
	if !ok {
		return ""
	}
	elapsed := time.Since(started)
	if elapsed < time.Second {
		return ""
	}
	remaining := time.Duration(float64(elapsed) * (100 - percent) / percent)
	if remaining < time.Second {
		return ""
	}
	return "~" + formatAgo(remaining) + " remaining"
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
