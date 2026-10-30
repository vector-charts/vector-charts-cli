package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	TypeCloud = "cloud"
	TypeOEM   = "oem"
)

// Profile is a named portal connection. Users only set URL (the portal).
// Type and APIURL are filled in after login from whoami / planInstance.
type Profile struct {
	URL       string `yaml:"url"`
	PortalURL string `yaml:"portal_url,omitempty"` // legacy alias for url
	APIURL    string `yaml:"api_url,omitempty"`
	Type      string `yaml:"type,omitempty"`
	LastSeen  string `yaml:"last_seen,omitempty"` // RFC3339 UTC; updated on successful contact
}

type File struct {
	CurrentProfile string             `yaml:"current_profile"`
	Profiles       map[string]Profile `yaml:"profiles"`
}

type Creds struct {
	SessionToken string `yaml:"session_token,omitempty"`
	APIToken     string `yaml:"api_token,omitempty"`
	MachineID    string `yaml:"machine_id,omitempty"`
}

type CredentialsFile struct {
	Profiles map[string]Creds `yaml:"profiles"`
}

func Dir() string {
	if v := os.Getenv("VECTOR_CHARTS_HOME"); v != "" {
		return expand(v)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".vector-charts")
	}
	return filepath.Join(home, ".vector-charts")
}

func ConfigPath() string {
	return filepath.Join(Dir(), "config.yml")
}

func CredentialsPath() string {
	return filepath.Join(Dir(), "credentials.yml")
}

func expand(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func EnsureDir() error {
	return os.MkdirAll(Dir(), 0o700)
}

func Load() (*File, error) {
	path := ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &File{Profiles: map[string]Profile{}}, nil
		}
		return nil, err
	}
	var f File
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if f.Profiles == nil {
		f.Profiles = map[string]Profile{}
	}
	for name, p := range f.Profiles {
		f.Profiles[name] = p.Normalized()
	}
	return &f, nil
}

func Save(f *File) error {
	if err := EnsureDir(); err != nil {
		return err
	}
	cleaned := &File{
		CurrentProfile: f.CurrentProfile,
		Profiles:       map[string]Profile{},
	}
	for name, p := range f.Profiles {
		p = p.Normalized()
		// Persist canonical fields only (drop legacy portal_url).
		cleaned.Profiles[name] = Profile{
			URL:      p.URL,
			APIURL:   p.APIURL,
			Type:     p.Type,
			LastSeen: p.LastSeen,
		}
	}
	data, err := yaml.Marshal(cleaned)
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigPath(), data, 0o600)
}

func LoadCredentials() (*CredentialsFile, error) {
	path := CredentialsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &CredentialsFile{Profiles: map[string]Creds{}}, nil
		}
		return nil, err
	}
	var f CredentialsFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if f.Profiles == nil {
		f.Profiles = map[string]Creds{}
	}
	return &f, nil
}

func SaveCredentials(f *CredentialsFile) error {
	if err := EnsureDir(); err != nil {
		return err
	}
	data, err := yaml.Marshal(f)
	if err != nil {
		return err
	}
	return os.WriteFile(CredentialsPath(), data, 0o600)
}

func (f *File) soleProfile() (string, Profile, bool) {
	if len(f.Profiles) != 1 {
		return "", Profile{}, false
	}
	for name, p := range f.Profiles {
		return name, p.Normalized(), true
	}
	return "", Profile{}, false
}

// Resolve picks a profile. Empty name means: the sole profile if there is
// only one, else current_profile, else "default".
func (f *File) Resolve(name string) (string, Profile, error) {
	if name == "" {
		if n, p, ok := f.soleProfile(); ok {
			return n, p, nil
		}
		if f.CurrentProfile != "" {
			name = f.CurrentProfile
		} else {
			name = "default"
		}
	}
	p, ok := f.Profiles[name]
	if !ok {
		if len(f.Profiles) == 0 {
			return "", Profile{}, errors.New("no profiles configured; run: vc server add [name] <url>")
		}
		return "", Profile{}, fmt.Errorf("unknown profile %q", name)
	}
	return name, p.Normalized(), nil
}

func (f *File) UpdateProfile(name string, update func(*Profile)) error {
	p, ok := f.Profiles[name]
	if !ok {
		return fmt.Errorf("unknown profile %q", name)
	}
	p = p.Normalized()
	update(&p)
	f.Profiles[name] = p
	return Save(f)
}

func (p Profile) Normalized() Profile {
	url := NormalizeURL(p.URL)
	if url == "" {
		url = NormalizeURL(p.PortalURL)
	}
	api := NormalizeURL(p.APIURL)
	if api == "" {
		api = url
	}
	return Profile{
		URL:      url,
		APIURL:   api,
		Type:     p.Type,
		LastSeen: p.LastSeen,
	}
}

// MarkSeen records that the profile's server was reachable just now.
func (f *File) MarkSeen(name string) {
	p, ok := f.Profiles[name]
	if !ok {
		return
	}
	p = p.Normalized()
	p.LastSeen = time.Now().UTC().Format(time.RFC3339)
	f.Profiles[name] = p
}

// TouchLastSeen loads config, marks the profile seen, and saves.
func TouchLastSeen(name string) error {
	cfg, err := Load()
	if err != nil {
		return err
	}
	if _, ok := cfg.Profiles[name]; !ok {
		return fmt.Errorf("unknown profile %q", name)
	}
	cfg.MarkSeen(name)
	return Save(cfg)
}

// FormatReachability returns "online" or "offline, last seen <duration> ago".
func FormatReachability(online bool, lastSeenRFC3339 string) string {
	if online {
		return "online"
	}
	if lastSeenRFC3339 == "" {
		return "offline"
	}
	t, err := time.Parse(time.RFC3339, lastSeenRFC3339)
	if err != nil {
		return "offline"
	}
	return "offline, last seen " + formatAgo(time.Since(t)) + " ago"
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
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

func NormalizeURL(u string) string {
	return strings.TrimRight(strings.TrimSpace(u), "/")
}

// ValidateBaseURL checks a portal/API base URL. Catches the common mistake of
// writing http://host/9904 instead of http://host:9904.
func ValidateBaseURL(raw string) (string, error) {
	raw = NormalizeURL(raw)
	if raw == "" {
		return "", fmt.Errorf("URL is required")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid URL %q: %w", raw, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("URL must start with http:// or https:// (got %q)", raw)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("invalid URL %q: missing host", raw)
	}
	// http://localhost/9904 → path looks like a port
	path := strings.Trim(parsed.Path, "/")
	if path != "" && !strings.Contains(path, "/") {
		if _, err := strconv.Atoi(path); err == nil && parsed.Port() == "" {
			return "", fmt.Errorf(
				"invalid URL %q: use a colon for the port, e.g. %s://%s:%s (not a slash)",
				raw, parsed.Scheme, parsed.Hostname(), path,
			)
		}
		return "", fmt.Errorf("base URL must not include a path (got %q)", raw)
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("base URL must not include query or fragment (got %q)", raw)
	}
	return NormalizeURL(fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)), nil
}

// ApplyWhoAmI sets type + api_url from a portal whoami/login user payload.
// Cloud responses include account.planInstance (hostname). OEM omits it → same-origin API.
func ApplyWhoAmI(portalURL string, user map[string]any) (typ string, apiURL string) {
	portalURL = NormalizeURL(portalURL)
	acct, _ := user["account"].(map[string]any)
	planInstance, _ := acct["planInstance"].(string)
	planInstance = strings.TrimSpace(planInstance)

	switch {
	case planInstance == "" || planInstance == "oem" || planInstance == "api-dev":
		return TypeOEM, portalURL
	case strings.Contains(planInstance, "://"):
		return TypeCloud, NormalizeURL(planInstance)
	default:
		return TypeCloud, "https://" + planInstance
	}
}
