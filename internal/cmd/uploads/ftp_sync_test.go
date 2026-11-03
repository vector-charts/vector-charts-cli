package uploads

import "testing"

func TestParseFTPTarget(t *testing.T) {
	cases := []struct {
		in   string
		addr string
		path string
	}{
		{"ftp.example.com/ENC", "ftp.example.com:21", "ENC"},
		{"ftp.example.com/charts/ENC", "ftp.example.com:21", "charts/ENC"},
		{"ftp://ftp.example.com:2121/data/enc", "ftp.example.com:2121", "data/enc"},
		{"ftp.example.com", "ftp.example.com:21", ""},
		{"ftp.example.com/", "ftp.example.com:21", ""},
	}
	for _, tc := range cases {
		got, err := parseFTPTarget(tc.in)
		if err != nil {
			t.Fatalf("parseFTPTarget(%q): %v", tc.in, err)
		}
		if got.Addr != tc.addr || got.Path != tc.path {
			t.Fatalf("parseFTPTarget(%q) = {%q,%q}, want {%q,%q}", tc.in, got.Addr, got.Path, tc.addr, tc.path)
		}
	}
	if _, err := parseFTPTarget("sftp://host/path"); err == nil {
		t.Fatal("expected error for sftp scheme")
	}
}
