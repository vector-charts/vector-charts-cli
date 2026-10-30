package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestResolveDefaults(t *testing.T) {
	sole := &File{Profiles: map[string]Profile{
		"local": {URL: "http://localhost:9909"},
	}}
	name, p, err := sole.Resolve("")
	if err != nil || name != "local" || p.URL != "http://localhost:9909" {
		t.Fatalf("sole profile: name=%q p=%+v err=%v", name, p, err)
	}

	multi := &File{
		Profiles: map[string]Profile{
			"a": {URL: "http://a.example"},
			"b": {URL: "http://b.example"},
		},
	}
	if _, _, err = multi.Resolve(""); err == nil {
		t.Fatal("expected error for missing default with multiple profiles")
	}

	multi.Profiles["default"] = Profile{URL: "http://default.example"}
	name, p, err = multi.Resolve("")
	if err != nil || name != "default" || p.URL != "http://default.example" {
		t.Fatalf("default profile: name=%q p=%+v err=%v", name, p, err)
	}

	multi.CurrentProfile = "b"
	name, p, err = multi.Resolve("")
	if err != nil || name != "b" || p.URL != "http://b.example" {
		t.Fatalf("current_profile: name=%q p=%+v err=%v", name, p, err)
	}
}

func TestLastSeenRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("VECTOR_CHARTS_HOME", dir)

	seen := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC).Format(time.RFC3339)
	f := &File{
		CurrentProfile: "local",
		Profiles: map[string]Profile{
			"local": {
				URL:      "http://localhost:9909",
				LastSeen: seen,
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
	if loaded.Profiles["local"].LastSeen != seen {
		t.Fatalf("last_seen=%q want %q", loaded.Profiles["local"].LastSeen, seen)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.MarkSeen("local")
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Profiles["local"].LastSeen == "" || loaded.Profiles["local"].LastSeen == seen {
		t.Fatalf("MarkSeen did not update last_seen: %q", loaded.Profiles["local"].LastSeen)
	}
}

func TestFormatReachability(t *testing.T) {
	if got := FormatReachability(true, ""); got != "online" {
		t.Fatalf("online: %q", got)
	}
	if got := FormatReachability(false, ""); got != "offline" {
		t.Fatalf("never seen: %q", got)
	}
	past := time.Now().UTC().Add(-90 * time.Minute).Format(time.RFC3339)
	got := FormatReachability(false, past)
	if !strings.HasPrefix(got, "offline, last seen ") || !strings.HasSuffix(got, " ago") {
		t.Fatalf("offline with last seen: %q", got)
	}
	if !strings.Contains(got, "1h") {
		t.Fatalf("expected ~1h ago, got %q", got)
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
