package cli

import (
	"fmt"
	"time"

	"github.com/vector-charts/vector-charts-cli/internal/config"
	"github.com/vector-charts/vector-charts-cli/internal/httpx"
)

// Factory resolves the active profile/client.
// When requireAuth is true, WhoAmI runs first; connection/auth failures are classified.
type Factory func(requireAuth bool) (name string, profile config.Profile, creds config.Creds, client *httpx.Client, err error)

func SaveSession(name, sessionToken string) error {
	creds, err := config.LoadCredentials()
	if err != nil {
		return err
	}
	c := creds.Profiles[name]
	c.SessionToken = sessionToken
	creds.Profiles[name] = c
	return config.SaveCredentials(creds)
}

func ClearSession(name string) error {
	creds, err := config.LoadCredentials()
	if err != nil {
		return err
	}
	c := creds.Profiles[name]
	c.SessionToken = ""
	creds.Profiles[name] = c
	return config.SaveCredentials(creds)
}

func ClearCredentials(name string) error {
	creds, err := config.LoadCredentials()
	if err != nil {
		return err
	}
	creds.Profiles[name] = config.Creds{}
	return config.SaveCredentials(creds)
}

func RequireNamedProfile(name string) error {
	if name == "" {
		return fmt.Errorf("refusing to persist credentials without a named profile; use --profile or vc server use")
	}
	return nil
}

// PersistLogin stores a personal API token (preferred CLI credential) and resolves api_url/type.

func PersistLogin(name, portalURL, personalAccessToken, machineID string, user map[string]any) error {
	if err := RequireNamedProfile(name); err != nil {
		return err
	}
	typ, apiURL := config.ApplyWhoAmI(portalURL, user)
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := cfg.UpdateProfile(name, func(p *config.Profile) {
		p.Type = typ
		p.APIURL = apiURL
		p.LastSeen = time.Now().UTC().Format(time.RFC3339)
	}); err != nil {
		return err
	}
	creds, err := config.LoadCredentials()
	if err != nil {
		return err
	}
	c := creds.Profiles[name]
	c.APIToken = personalAccessToken
	c.MachineID = machineID
	c.SessionToken = ""
	creds.Profiles[name] = c
	return config.SaveCredentials(creds)
}
