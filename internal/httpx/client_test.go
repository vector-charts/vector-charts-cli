package httpx

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/vector-charts/vector-charts-cli/internal/config"
)

func TestUserAgent(t *testing.T) {
	Version = "1.2.3"
	want := fmt.Sprintf("Vector-Charts-CLI/1.2.3 (%s/%s)", runtime.GOOS, runtime.GOARCH)
	if got := UserAgent(); got != want {
		t.Fatalf("UserAgent()=%q want %q", got, want)
	}
}

func TestDoJSONSetsUserAgent(t *testing.T) {
	Version = "9.9.9"
	want := UserAgent()
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := New(config.Profile{URL: srv.URL}, config.Creds{})
	c.PortalURL = srv.URL
	if err := c.doJSON(http.MethodGet, srv.URL+"/health", "", nil, &map[string]any{}); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("User-Agent=%q want %q", got, want)
	}
}
