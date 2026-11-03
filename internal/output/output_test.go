package output

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = old }()

	fn()
	_ = w.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestPrettyTable(t *testing.T) {
	Format = "table"
	t.Setenv("NO_COLOR", "1")
	out := captureStdout(t, func() {
		if err := Table([]string{"CODE", "NAME"}, [][]string{
			{"au", "Australia"},
			{"uk", "United Kingdom"},
		}); err != nil {
			t.Fatal(err)
		}
	})
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 6 {
		t.Fatalf("expected 6 lines, got %d: %q", len(lines), out)
	}
	if !strings.HasPrefix(lines[0], "┌") || !strings.Contains(lines[0], "┬") || !strings.HasSuffix(lines[0], "┐") {
		t.Fatalf("bad top border: %q", lines[0])
	}
	if !strings.Contains(lines[1], "CODE") || !strings.Contains(lines[1], "NAME") {
		t.Fatalf("bad header: %q", lines[1])
	}
	if !strings.HasPrefix(lines[2], "├") || !strings.Contains(lines[2], "┼") {
		t.Fatalf("expected header rule, got %q", lines[2])
	}
	if !strings.Contains(lines[3], "au") || !strings.Contains(lines[4], "uk") {
		t.Fatalf("bad rows: %q", out)
	}
	if strings.Contains(lines[3], "┼") {
		t.Fatalf("unexpected row separator between data rows: %q", out)
	}
	if !strings.HasPrefix(lines[5], "└") {
		t.Fatalf("bad bottom border: %q", lines[5])
	}
}

func TestHeaderlessPrettyTable(t *testing.T) {
	Format = "table"
	t.Setenv("NO_COLOR", "1")
	out := captureStdout(t, func() {
		if err := Table(nil, [][]string{
			{"default*", "http://localhost:9909", "online"},
			{"cloud", "https://example.com", "offline, last seen 2h ago"},
		}); err != nil {
			t.Fatal(err)
		}
	})
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines (top, 2 rows, bottom), got %d: %q", len(lines), out)
	}
	if !strings.HasPrefix(lines[0], "┌") {
		t.Fatalf("expected top border: %q", lines[0])
	}
	if !strings.Contains(lines[1], "default*") || !strings.Contains(lines[1], "│") || !strings.Contains(lines[1], "online") {
		t.Fatalf("bad row: %q", lines[1])
	}
	if strings.Contains(lines[1], "┼") || strings.HasPrefix(lines[2], "├") {
		t.Fatalf("unexpected mid-row rule: %q", out)
	}
	if !strings.HasPrefix(lines[3], "└") {
		t.Fatalf("expected bottom border: %q", lines[3])
	}
}

func TestMultilinePrettyTable(t *testing.T) {
	Format = "table"
	t.Setenv("NO_COLOR", "1")
	out := captureStdout(t, func() {
		if err := Table(nil, [][]string{
			{"local-oem", "online\noem\nportal ok"},
		}); err != nil {
			t.Fatal(err)
		}
	})
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 5 {
		t.Fatalf("expected 5 lines (top, 3 content, bottom), got %d: %q", len(lines), out)
	}
	if !strings.Contains(lines[1], "local-oem") || !strings.Contains(lines[1], "online") {
		t.Fatalf("bad first content line: %q", lines[1])
	}
	if strings.Contains(lines[2], "local-oem") {
		t.Fatalf("name should only appear on first content line: %q", out)
	}
	if !strings.Contains(lines[2], "oem") || !strings.Contains(lines[3], "portal ok") {
		t.Fatalf("bad status lines: %q", out)
	}
}

func TestConnecting(t *testing.T) {
	Format = "table"
	t.Setenv("NO_COLOR", "1")
	out := captureStdout(t, func() {
		Connecting("local-oem")
	})
	want := "connecting to server 'local-oem'...\n"
	if out != want {
		t.Fatalf("got %q want %q", out, want)
	}

	Format = "json"
	out = captureStdout(t, func() {
		Connecting("local-oem")
	})
	if out != "" {
		t.Fatalf("expected no connecting line for json, got %q", out)
	}
}

func TestColorDisabledByNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if ColorEnabled() {
		t.Fatal("expected ColorEnabled false with NO_COLOR")
	}
	if Green("online") != "online" || Red("offline") != "offline" || Yellow("warn") != "warn" || Cyan("run") != "run" || PaleGrey("x") != "x" {
		t.Fatal("expected no ANSI when colors disabled")
	}
}

func TestSetFormat(t *testing.T) {
	for _, f := range []string{"table", "json", "JSON", "Table", "plaintext", "Plaintext"} {
		if err := SetFormat(f); err != nil {
			t.Fatalf("SetFormat(%q): %v", f, err)
		}
	}
	if err := SetFormat("yaml"); err == nil {
		t.Fatal("expected error for yaml")
	}
}

func TestPlaintextTable(t *testing.T) {
	if err := SetFormat("plaintext"); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = SetFormat("table") }()
	out := captureStdout(t, func() {
		if err := Table([]string{"CODE", "NAME"}, [][]string{
			{"au", "Australia"},
			{"uk", "United Kingdom"},
			{"x", "a,b"},
		}); err != nil {
			t.Fatal(err)
		}
	})
	want := "CODE,NAME\nau,Australia\nuk,United Kingdom\nx,\"a,b\"\n"
	if out != want {
		t.Fatalf("got %q want %q", out, want)
	}
	if ColorEnabled() {
		t.Fatal("expected ColorEnabled false in plaintext mode")
	}
}

func TestColorState(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	cases := map[string]string{
		"online":                     "online",
		"offline, last seen 2h ago":  "offline, last seen 2h ago",
		"completed":                  "completed",
		"failed":                     "failed",
		"running":                    "running",
		"true":                       "true",
		"false":                      "false",
		"ok (cloud)":                 "ok (cloud)",
		"failed: boom":               "failed: boom",
		"queued":                     "queued",
	}
	for in, want := range cases {
		if got := ColorState(in); got != want {
			t.Fatalf("ColorState(%q)=%q want %q", in, got, want)
		}
	}
}
