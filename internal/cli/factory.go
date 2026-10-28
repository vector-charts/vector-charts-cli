package cli

import (
	"fmt"

	"github.com/zydro-marine/vector-charts-cli/internal/config"
	"github.com/zydro-marine/vector-charts-cli/internal/httpx"
)

type Factory func(requireProfile bool) (name string, profile config.Profile, creds config.Creds, client *httpx.Client, err error)

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

func SaveAPIToken(name, apiToken string) error {
	creds, err := config.LoadCredentials()
	if err != nil {
		return err
	}
	c := creds.Profiles[name]
	c.APIToken = apiToken
	creds.Profiles[name] = c
	return config.SaveCredentials(creds)
}

func RequireNamedProfile(name string) error {
	if name == "" || name == "(override)" {
		return fmt.Errorf("refusing to persist credentials without a named profile; use --profile or vc server use")
	}
	return nil
}

// PersistLogin stores a personal API token (preferred CLI credential) and resolves api_url/type.

func PersistLogin(name, portalURL, personalToken, machineID string, user map[string]any) error {
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
	}); err != nil {
		return err
	}
	creds, err := config.LoadCredentials()
	if err != nil {
		return err
	}
	c := creds.Profiles[name]
	c.APIToken = personalToken
	c.MachineID = machineID
	c.SessionToken = ""
	creds.Profiles[name] = c
	return config.SaveCredentials(creds)
}
