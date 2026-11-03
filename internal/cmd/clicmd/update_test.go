package clicmd

import "testing"

func TestNormalizeVersion(t *testing.T) {
	cases := map[string]string{
		"v1.2.3":     "1.2.3",
		"1.2.3":      "1.2.3",
		"V0.1.0":     "0.1.0",
		"1.2.3-beta": "1.2.3",
		" 1.0.0 ":    "1.0.0",
	}
	for in, want := range cases {
		if got := normalizeVersion(in); got != want {
			t.Fatalf("normalizeVersion(%q)=%q want %q", in, got, want)
		}
	}
}

func TestCompareVersions(t *testing.T) {
	if compareVersions("0.1.0", "0.1.0") != 0 {
		t.Fatal("equal")
	}
	if compareVersions("0.1.0", "0.2.0") >= 0 {
		t.Fatal("0.1.0 < 0.2.0")
	}
	if compareVersions("1.0.0", "0.9.9") <= 0 {
		t.Fatal("1.0.0 > 0.9.9")
	}
	if compareVersions("v1.2.3", "1.2.4") >= 0 {
		t.Fatal("v1.2.3 < 1.2.4")
	}
}
