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
	out := captureStdout(t, func() {
		if err := Table([]string{"CODE", "NAME"}, [][]string{
			{"au", "Australia"},
			{"uk", "United Kingdom"},
		}); err != nil {
			t.Fatal(err)
		}
	})
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines, got %d: %q", len(lines), out)
	}
	if !strings.Contains(lines[0], "CODE") || !strings.Contains(lines[0], "NAME") {
		t.Fatalf("bad header: %q", lines[0])
	}
	if !strings.Contains(lines[1], "─") {
		t.Fatalf("expected rule line, got %q", lines[1])
	}
	if !strings.Contains(lines[2], "au") || !strings.Contains(lines[3], "uk") {
		t.Fatalf("bad rows: %q", out)
	}
}

func TestPlaintextTable(t *testing.T) {
	Format = "plaintext"
	out := captureStdout(t, func() {
		if err := Table([]string{"CODE", "NAME"}, [][]string{{"au", "Australia"}}); err != nil {
			t.Fatal(err)
		}
	})
	want := "CODE\tNAME\nau\tAustralia\n"
	if out != want {
		t.Fatalf("got %q want %q", out, want)
	}
}

func TestSetFormat(t *testing.T) {
	for _, f := range []string{"table", "json", "plaintext", "JSON", "Table"} {
		if err := SetFormat(f); err != nil {
			t.Fatalf("SetFormat(%q): %v", f, err)
		}
	}
	if err := SetFormat("yaml"); err == nil {
		t.Fatal("expected error for yaml")
	}
}
