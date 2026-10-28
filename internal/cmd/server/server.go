package server

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/zydro-marine/vector-charts-cli/internal/cli"
	"github.com/zydro-marine/vector-charts-cli/internal/config"
	"github.com/zydro-marine/vector-charts-cli/internal/discovery"
	"github.com/zydro-marine/vector-charts-cli/internal/httpx"
	"github.com/zydro-marine/vector-charts-cli/internal/machineid"
	"github.com/zydro-marine/vector-charts-cli/internal/output"
	"golang.org/x/term"
)

func NewCommand(factory cli.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Manage connection profiles and authentication",
	}
	cmd.AddCommand(newListCmd(factory))
	cmd.AddCommand(newAddCmd())
	cmd.AddCommand(newRemoveCmd())
	cmd.AddCommand(newUseCmd())
	cmd.AddCommand(newShowCmd(factory))
	cmd.AddCommand(newLoginCmd(factory))
	cmd.AddCommand(newLogoutCmd(factory))
	cmd.AddCommand(newWhoamiCmd(factory))
	cmd.AddCommand(newStatusCmd(factory))
	cmd.AddCommand(newDiscoverCmd())
	return cmd
}

func newDiscoverCmd() *cobra.Command {
	var timeoutSec int
	var group string
	var port int
	cmd := &cobra.Command{
		Use:   "discover",
		Short: "Discover Vector Charts instances on the local network",
		Long: `Listen for UDP multicast discovery beacons ($VC-DISCOVER-1.0) and list
instances announcing themselves on the LAN (same mechanism as Zeus discovery).

OEM containers typically need host networking for multicast to reach the LAN.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			output.Errf("Listening for Vector Charts instances on %s:%d (%ds)…\n", group, port, timeoutSec)
			peers, err := discovery.Listen(group, port, time.Duration(timeoutSec)*time.Second)
			if err != nil {
				return err
			}
			return output.PrintJSONOr(peers, func() error {
				if len(peers) == 0 {
					output.Println("No instances discovered.")
					return nil
				}
				rows := make([][]string, 0, len(peers))
				for _, p := range peers {
					rows = append(rows, []string{
						p.InstanceName,
						p.DeploymentMode,
						p.SourceIP,
						fmt.Sprintf("%d", p.HTTPPort),
						p.Version,
						p.BaseURL(),
						p.InstanceID,
					})
				}
				return output.Table(
					[]string{"NAME", "MODE", "ADDRESS", "PORT", "VERSION", "URL", "ID"},
					rows,
				)
			})
		},
	}
	cmd.Flags().IntVar(&timeoutSec, "timeout", 3, "Seconds to listen for beacons")
	cmd.Flags().StringVar(&group, "group", discovery.DefaultGroup, "Multicast group")
	cmd.Flags().IntVar(&port, "port", discovery.DefaultPort, "UDP port")
	return cmd
}

func newAddCmd() *cobra.Command {
	var url string
	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a connection profile (portal URL only)",
		Long: `Add a named profile pointing at a portal URL.

Cloud vs OEM and the tile API URL are resolved automatically after login
from the portal whoami response (account.planInstance).`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			var err error
			url, err = config.ValidateBaseURL(url)
			if err != nil {
				return err
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if _, exists := cfg.Profiles[name]; exists {
				return fmt.Errorf("profile %q already exists", name)
			}
			cfg.Profiles[name] = config.Profile{URL: url}
			if cfg.CurrentProfile == "" {
				cfg.CurrentProfile = name
			}
			if err := config.Save(cfg); err != nil {
				return err
			}
			output.Printf("Added profile %q → %s\n", name, url)
			return nil
		},
	}
	cmd.Flags().StringVar(&url, "url", "", "Portal base URL (e.g. https://cloud.vectorcharts.com or http://localhost:9909)")
	return cmd
}

func newRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a connection profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if _, ok := cfg.Profiles[name]; !ok {
				return fmt.Errorf("unknown profile %q", name)
			}
			delete(cfg.Profiles, name)
			if cfg.CurrentProfile == name {
				cfg.CurrentProfile = ""
				for k := range cfg.Profiles {
					cfg.CurrentProfile = k
					break
				}
			}
			if err := config.Save(cfg); err != nil {
				return err
			}
			creds, err := config.LoadCredentials()
			if err == nil {
				delete(creds.Profiles, name)
				_ = config.SaveCredentials(creds)
			}
			output.Printf("Removed profile %q\n", name)
			return nil
		},
	}
}

func newUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use <name>",
		Short: "Set the current profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if _, ok := cfg.Profiles[name]; !ok {
				return fmt.Errorf("unknown profile %q", name)
			}
			cfg.CurrentProfile = name
			if err := config.Save(cfg); err != nil {
				return err
			}
			output.Printf("Using profile %q\n", name)
			return nil
		},
	}
}

func newShowCmd(factory cli.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "show [name]",
		Short: "Show a profile (default: current)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			name := ""
			if len(args) == 1 {
				name = args[0]
			}
			name, p, err := cfg.Resolve(name)
			if err != nil {
				return err
			}
			credsFile, _ := config.LoadCredentials()
			creds := credsFile.Profiles[name]
			payload := map[string]any{
				"name":          name,
				"url":           p.URL,
				"api_url":       p.APIURL,
				"type":          p.Type,
				"current":       name == cfg.CurrentProfile,
				"has_session":   creds.SessionToken != "",
				"has_api_token": creds.APIToken != "",
			}
			return output.PrintJSONOr(payload, func() error {
				output.Printf("name:          %s\n", name)
				output.Printf("url:           %s\n", p.URL)
				if p.APIURL != "" && p.APIURL != p.URL {
					output.Printf("api_url:       %s\n", p.APIURL)
				}
				if p.Type != "" {
					output.Printf("type:          %s\n", p.Type)
				}
				output.Printf("current:       %v\n", name == cfg.CurrentProfile)
				output.Printf("has_session:   %v\n", creds.SessionToken != "")
				output.Printf("has_api_token: %v\n", creds.APIToken != "")
				return nil
			})
		},
	}
}

type probeResult struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Portal    string `json:"portal"`
	API       string `json:"api"`
	PortalOK  bool   `json:"portal_ok"`
	APIOK     bool   `json:"api_ok"`
	Auth      string `json:"auth"`
	User      string `json:"user,omitempty"`
	APIMode   string `json:"api_mode,omitempty"`
	PortalErr string `json:"portal_error,omitempty"`
	APIErr    string `json:"api_error,omitempty"`
}

func probeProfile(name string, p config.Profile, creds config.Creds) probeResult {
	p = p.Normalized()
	client := httpx.New(p, creds)
	res := probeResult{
		Name:   name,
		Type:   p.Type,
		Portal: p.URL,
		API:    p.APIURL,
		Auth:   "none",
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if _, err := client.PortalHealth(); err != nil {
			res.PortalErr = err.Error()
			return
		}
		res.PortalOK = true
	}()
	go func() {
		defer wg.Done()
		if p.APIURL == "" {
			res.APIErr = "api url not resolved (login first)"
			return
		}
		h, err := client.ServerHealth()
		if err != nil {
			res.APIErr = err.Error()
			return
		}
		res.APIOK = true
		if m, ok := h["mode"].(string); ok {
			res.APIMode = m
			if res.Type == "" {
				res.Type = m
			}
		}
	}()
	wg.Wait()

	if creds.SessionToken != "" || creds.APIToken != "" {
		user, err := client.WhoAmI()
		if err != nil {
			res.Auth = "failed"
		} else {
			res.Auth = "ok"
			if u, ok := user["username"].(string); ok {
				res.User = u
			}
		}
	}
	return res
}

func newListCmd(factory cli.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List profiles and probe connectivity",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			credsFile, err := config.LoadCredentials()
			if err != nil {
				return err
			}
			names := make([]string, 0, len(cfg.Profiles))
			for n := range cfg.Profiles {
				names = append(names, n)
			}
			results := make([]probeResult, len(names))
			var wg sync.WaitGroup
			for i, n := range names {
				wg.Add(1)
				go func(i int, n string) {
					defer wg.Done()
					results[i] = probeProfile(n, cfg.Profiles[n], credsFile.Profiles[n])
				}(i, n)
			}
			wg.Wait()

			return output.PrintJSONOr(results, func() error {
				rows := make([][]string, 0, len(results))
				for _, r := range results {
					mark := r.Name
					if r.Name == cfg.CurrentProfile {
						mark = r.Name + "*"
					}
					rows = append(rows, []string{
						mark,
						r.Type,
						r.Portal,
						r.API,
						boolStr(r.PortalOK),
						boolStr(r.APIOK),
						r.Auth,
						r.User,
					})
				}
				return output.Table(
					[]string{"NAME", "TYPE", "PORTAL", "API", "PORTAL_OK", "API_OK", "AUTH", "USER"},
					rows,
				)
			})
		},
	}
}

func boolStr(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func newStatusCmd(factory cli.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Probe connectivity for the current profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, p, creds, _, err := factory(true)
			if err != nil {
				return err
			}
			res := probeProfile(name, p, creds)
			errOut := output.PrintJSONOr(res, func() error {
				output.Printf("profile:    %s\n", res.Name)
				output.Printf("type:       %s\n", res.Type)
				output.Printf("portal_ok:  %v\n", res.PortalOK)
				output.Printf("api_ok:     %v\n", res.APIOK)
				if res.APIMode != "" {
					output.Printf("api_mode:   %s\n", res.APIMode)
				}
				output.Printf("auth:       %s\n", res.Auth)
				if res.User != "" {
					output.Printf("user:       %s\n", res.User)
				}
				if res.PortalErr != "" {
					output.Printf("portal_err: %s\n", res.PortalErr)
				}
				if res.APIErr != "" {
					output.Printf("api_err:    %s\n", res.APIErr)
				}
				return nil
			})
			if !res.PortalOK || !res.APIOK {
				return fmt.Errorf("one or more health checks failed")
			}
			return errOut
		},
	}
}

func newWhoamiCmd(factory cli.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show the authenticated portal user",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _, _, client, err := factory(true)
			if err != nil {
				return err
			}
			user, err := client.WhoAmI()
			if err != nil {
				return err
			}
			return output.PrintJSONOr(user, func() error {
				output.Printf("userId:   %v\n", user["userId"])
				output.Printf("username: %v\n", user["username"])
				output.Printf("isAdmin:  %v\n", user["isAdmin"])
				if acct, ok := user["account"].(map[string]any); ok {
					output.Printf("account:  %v (%v)\n", acct["name"], acct["id"])
				}
				return nil
			})
		},
	}
}

func newLogoutCmd(factory cli.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Clear cached CLI credentials for the current profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _, creds, client, err := factory(true)
			if err != nil {
				return err
			}
			if err := cli.RequireNamedProfile(name); err != nil {
				return err
			}
			if creds.SessionToken != "" || creds.APIToken != "" {
				_ = client.Logout()
			}
			if err := cli.ClearCredentials(name); err != nil {
				return err
			}
			output.Printf("Logged out profile %q\n", name)
			return nil
		},
	}
}

func newLoginCmd(factory cli.Factory) *cobra.Command {
	var tokenFlag, username string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with a Personal Token (device browser flow, --token, or --username)",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, p, creds, client, err := factory(true)
			if err != nil {
				return err
			}
			if err := cli.RequireNamedProfile(name); err != nil {
				return err
			}

			machineID, err := machineid.ID()
			if err != nil {
				return fmt.Errorf("machine id: %w", err)
			}
			machineLabel := machineid.Label()

			if tokenFlag == "" && username == "" && creds.APIToken != "" && creds.MachineID == machineID {
				client.APIToken = creds.APIToken
				client.Session = ""
				user, err := client.WhoAmI()
				if err == nil {
					if err := cli.PersistLogin(name, p.URL, creds.APIToken, machineID, user); err != nil {
						return err
					}
					output.Printf("Already logged in to profile %q as %v (reusing Personal Token)\n", name, user["username"])
					return nil
				}
			}

			if tokenFlag != "" {
				client.Session = ""
				client.APIToken = tokenFlag
				user, err := client.WhoAmI()
				if err != nil {
					return fmt.Errorf("token rejected: %w", err)
				}
				if err := cli.PersistLogin(name, p.URL, tokenFlag, machineID, user); err != nil {
					return err
				}
				output.Printf("Logged in to profile %q as %v\n", name, user["username"])
				return nil
			}

			if username != "" {
				password, err := readPassword("Password: ")
				if err != nil {
					return err
				}
				sessionID, user, err := client.Login(username, password)
				if err != nil {
					return err
				}
				client.Session = sessionID
				client.APIToken = ""
				personalID, user, err := client.IssuePersonalToken(machineID, machineLabel)
				if err != nil {
					return fmt.Errorf("personal token: %w", err)
				}
				if err := cli.PersistLogin(name, p.URL, personalID, machineID, user); err != nil {
					return err
				}
				uname, _ := user["username"].(string)
				output.Printf("Logged in as %s on profile %q\n", uname, name)
				return nil
			}

			return deviceLogin(name, p.URL, client, machineID, machineLabel)
		},
	}
	cmd.Flags().StringVar(&tokenFlag, "token", "", "Personal Token (or legacy session / OEM API token)")
	cmd.Flags().StringVar(&username, "username", "", "Portal username/email for password login")
	return cmd
}

func deviceLogin(name, portalURL string, client *httpx.Client, machineID, machineLabel string) error {
	start, err := client.StartDeviceAuth(machineID, machineLabel)
	if err != nil {
		return fmt.Errorf("device auth not available (is the portal updated?): %w", err)
	}

	output.Printf("Open the following URL in a browser and confirm the code:\n\n")
	output.Printf("  %s\n\n", start.VerificationURIComplete)
	output.Printf("User code: %s\n\n", start.UserCode)
	output.Printf("Waiting for approval...\n")

	_ = openBrowser(start.VerificationURIComplete)

	deadline := time.Now().Add(time.Duration(start.ExpiresIn) * time.Second)
	interval := time.Duration(start.Interval) * time.Second
	for time.Now().Before(deadline) {
		time.Sleep(interval)
		status, tokenID, user, err := client.PollDeviceAuth(start.DeviceCode)
		if err != nil {
			if apiErr, ok := err.(*httpx.APIError); ok && apiErr.Status == 400 {
				return err
			}
			continue
		}
		if status == "pending" {
			continue
		}
		if status == "complete" && tokenID != "" {
			if err := cli.PersistLogin(name, portalURL, tokenID, machineID, user); err != nil {
				return err
			}
			uname, _ := user["username"].(string)
			output.Printf("Logged in as %s on profile %q\n", uname, name)
			return nil
		}
	}
	return fmt.Errorf("timed out waiting for device approval")
}

func readPassword(prompt string) (string, error) {
	output.Errf("%s", prompt)
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		var pw string
		_, err := fmt.Fscanln(os.Stdin, &pw)
		return pw, err
	}
	b, err := term.ReadPassword(fd)
	output.Errf("\n")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported OS")
	}
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}
