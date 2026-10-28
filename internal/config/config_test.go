package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("VECTOR_CHARTS_HOME", dir)

	f := &File{
		CurrentProfile: "local",
		Profiles: map[string]Profile{
			"local": {
				URL: "http://localhost:9909",
			},
		},
	}
	if err := Save(f); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.CurrentProfile != "local" {
		t.Fatalf("current=%q", loaded.CurrentProfile)
	}
	p := loaded.Profiles["local"]
	if p.URL != "http://localhost:9909" {
		t.Fatalf("profile=%+v", p)
	}
	if p.APIURL != "http://localhost:9909" {
		t.Fatalf("api default=%q", p.APIURL)
	}

	creds := &CredentialsFile{Profiles: map[string]Creds{
		"local": {SessionToken: "abc"},
	}}
	if err := SaveCredentials(creds); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, "credentials.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("credentials too open: %v", info.Mode())
	}
}

func TestApplyWhoAmI(t *testing.T) {
	typ, api := ApplyWhoAmI("http://localhost:9909", map[string]any{
		"account": map[string]any{"id": 1, "name": "oem"},
	})
	if typ != TypeOEM || api != "http://localhost:9909" {
		t.Fatalf("oem: typ=%s api=%s", typ, api)
	}

	typ, api = ApplyWhoAmI("https://cloud.vectorcharts.com", map[string]any{
		"account": map[string]any{
			"planInstance": "api.vectorcharts.com",
		},
	})
	if typ != TypeCloud || api != "https://api.vectorcharts.com" {
		t.Fatalf("cloud: typ=%s api=%s", typ, api)
	}

	typ, api = ApplyWhoAmI("https://cloud.vectorcharts.com", map[string]any{
		"account": map[string]any{
			"planInstance": "api-enterprise-1.example.com",
		},
	})
	if typ != TypeCloud || api != "https://api-enterprise-1.example.com" {
		t.Fatalf("dedicated: typ=%s api=%s", typ, api)
	}
}

func TestValidateBaseURLPortSlash(t *testing.T) {
	_, err := ValidateBaseURL("http://localhost/9904")
	if err == nil || !strings.Contains(err.Error(), "colon") {
		t.Fatalf("expected colon hint, got %v", err)
	}
	got, err := ValidateBaseURL("http://localhost:9904/")
	if err != nil {
		t.Fatal(err)
	}
	if got != "http://localhost:9904" {
		t.Fatalf("got %q", got)
	}
}

func TestLegacyPortalURL(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("VECTOR_CHARTS_HOME", dir)
	path := ConfigPath()
	if err := os.WriteFile(path, []byte(`
current_profile: old
profiles:
  old:
    type: oem
    portal_url: http://localhost:9909
    api_url: http://localhost:9909
`), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	p := loaded.Profiles["old"]
	if p.URL != "http://localhost:9909" {
		t.Fatalf("legacy portal_url not migrated: %+v", p)
	}
}
