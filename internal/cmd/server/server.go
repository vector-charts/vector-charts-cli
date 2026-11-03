package server

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
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
	cmd.AddCommand(newInfoCmd(factory))
	cmd.AddCommand(newLoginCmd(factory))
	cmd.AddCommand(newLogoutCmd(factory))
	cmd.AddCommand(newWhoamiCmd(factory))
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
					})
				}
				return output.Table(
					[]string{"NAME", "MODE", "ADDRESS", "PORT", "VERSION", "URL"},
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

func looksLikeURL(s string) bool {
	return strings.Contains(s, "://")
}

func newAddCmd() *cobra.Command {
	var urlFlag string
	cmd := &cobra.Command{
		Use:   "add [name] [url]",
		Short: "Add a connection profile (portal URL only)",
		Long: `Add a named profile pointing at a portal URL.

Cloud vs OEM and the tile API URL are resolved automatically after login
from the portal whoami response (account.planInstance).

Examples:
  vc server add https://cloud.vectorcharts.com
  vc server add cloud https://cloud.vectorcharts.com
  vc server add oem --url http://localhost:9909

If name is omitted, the profile is stored as "default".`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := "default"
			rawURL := ""
			switch len(args) {
			case 1:
				if looksLikeURL(args[0]) {
					rawURL = args[0]
				} else {
					name = args[0]
				}
			case 2:
				name = args[0]
				rawURL = args[1]
			}
			if urlFlag != "" {
				if rawURL != "" && rawURL != urlFlag {
					return fmt.Errorf("conflicting URL: positional %q and --url %q", rawURL, urlFlag)
				}
				rawURL = urlFlag
			}
			if rawURL == "" {
				return fmt.Errorf("URL is required (positional or --url)")
			}
			validated, err := config.ValidateBaseURL(rawURL)
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
			cfg.Profiles[name] = config.Profile{URL: validated}
			if cfg.CurrentProfile == "" {
				cfg.CurrentProfile = name
			}
			if err := config.Save(cfg); err != nil {
				return err
			}
			output.Printf("Added profile %q → %s\n", name, validated)
			return nil
		},
	}
	cmd.Flags().StringVar(&urlFlag, "url", "", "Portal base URL (e.g. https://cloud.vectorcharts.com or http://localhost:9909)")
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

func newInfoCmd(factory cli.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "info [name]",
		Aliases: []string{"show", "status"},
		Short:   "Show profile details and probe connectivity (default: current)",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			var name string
			var p config.Profile
			var creds config.Creds
			if len(args) == 1 {
				name, p, err = cfg.Resolve(args[0])
				if err != nil {
					return err
				}
				credsFile, _ := config.LoadCredentials()
				creds = credsFile.Profiles[name]
			} else {
				name, p, creds, _, err = factory(false)
				if err != nil {
					return err
				}
			}

			res := probeProfile(name, p, creds)
			if res.PortalOK || res.APIOK {
				_ = config.TouchLastSeen(name)
			}
			payload := map[string]any{
				"name":          name,
				"url":           p.URL,
				"api_url":       p.APIURL,
				"type":          firstNonEmpty(res.Type, p.Type),
				"last_seen":     firstNonEmpty(res.LastSeen, p.LastSeen),
				"current":       name == cfg.CurrentProfile,
				"has_session":   creds.SessionToken != "",
				"has_api_token": creds.APIToken != "",
				"status":        res.Status,
				"portal_ok":     res.PortalOK,
				"api_ok":        res.APIOK,
				"auth":          res.Auth,
				"user":          res.User,
				"api_mode":      res.APIMode,
				"portal_error":  res.PortalErr,
				"api_error":     res.APIErr,
			}
			errOut := output.PrintJSONOr(payload, func() error {
				output.Printf("name:          %s\n", name)
				output.Printf("url:           %s\n", p.URL)
				if p.APIURL != "" && p.APIURL != p.URL {
					output.Printf("api_url:       %s\n", p.APIURL)
				}
				typ := firstNonEmpty(res.Type, p.Type)
				if typ != "" {
					output.Printf("type:          %s\n", typ)
				}
				lastSeen := firstNonEmpty(res.LastSeen, p.LastSeen)
				if lastSeen != "" {
					output.Printf("last_seen:     %s\n", lastSeen)
				}
				output.Printf("current:       %v\n", name == cfg.CurrentProfile)
				output.Printf("has_session:   %v\n", creds.SessionToken != "")
				output.Printf("has_api_token: %v\n", creds.APIToken != "")
				output.Printf("status:\n%s\n", indentLines(statusLines(res), "  "))
				return nil
			})
			if !res.PortalOK || !res.APIOK {
				return fmt.Errorf("one or more health checks failed")
			}
			return errOut
		},
	}
	cli.HideAliases(cmd)
	return cmd
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func indentLines(s, prefix string) string {
	if s == "" {
		return s
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}

type probeResult struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Portal    string `json:"portal"`
	API       string `json:"api"`
	PortalOK  bool   `json:"portal_ok"`
	APIOK     bool   `json:"api_ok"`
	Status    string `json:"status"`
	LastSeen  string `json:"last_seen,omitempty"`
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
		Name:     name,
		Type:     p.Type,
		Portal:   p.URL,
		API:      p.APIURL,
		LastSeen: p.LastSeen,
		Auth:     "none",
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

	online := res.PortalOK || res.APIOK
	if online {
		res.LastSeen = time.Now().UTC().Format(time.RFC3339)
	}
	res.Status = config.FormatReachability(online, res.LastSeen)

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

			for _, r := range results {
				if r.PortalOK || r.APIOK {
					cfg.MarkSeen(r.Name)
				}
			}
			_ = config.Save(cfg)

			return output.PrintJSONOr(results, func() error {
				rows := make([][]string, 0, len(results))
				for _, r := range results {
					mark := r.Name
					if r.Name == cfg.CurrentProfile {
						mark = r.Name + "*"
					}
					rows = append(rows, []string{
						mark,
						r.Portal,
						colorStatus(r.Status, r.PortalOK || r.APIOK),
					})
				}
				return output.Table(nil, rows)
			})
		},
	}
}

func colorStatus(status string, online bool) string {
	if online {
		return output.Green(status)
	}
	return output.Red(status)
}

func statusLines(res probeResult) string {
	lines := []string{colorStatus(res.Status, res.PortalOK || res.APIOK)}
	if res.Type != "" {
		lines = append(lines, output.PaleGrey(res.Type))
	}
	if res.PortalOK {
		lines = append(lines, output.PaleGrey("portal ok"))
	} else if res.PortalErr != "" {
		lines = append(lines, output.PaleGrey("portal: "+res.PortalErr))
	} else {
		lines = append(lines, output.PaleGrey("portal down"))
	}
	if res.APIOK {
		api := "api ok"
		if res.APIMode != "" && res.APIMode != res.Type {
			api += " (" + res.APIMode + ")"
		}
		lines = append(lines, output.PaleGrey(api))
	} else if res.APIErr != "" {
		lines = append(lines, output.PaleGrey("api: "+res.APIErr))
	} else {
		lines = append(lines, output.PaleGrey("api down"))
	}
	switch res.Auth {
	case "ok":
		if res.User != "" {
			lines = append(lines, output.PaleGrey("logged in as "+res.User))
		} else {
			lines = append(lines, output.PaleGrey("auth ok"))
		}
	case "failed":
		lines = append(lines, output.PaleGrey("auth failed"))
	}
	return strings.Join(lines, "\n")
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
			name, _, creds, client, err := factory(false)
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
		Short: "Authenticate with a Personal Access Token (device browser flow, --token, or --username)",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, p, creds, client, err := factory(false)
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

			// Always probe existing credentials first when present. Connection
			// failures abort; expired auth is reported but login continues so
			// re-login always works even when already logged in.
			if creds.APIToken != "" || creds.SessionToken != "" {
				if _, err := client.WhoAmI(); err != nil {
					if httpx.IsConnectionError(err) {
						return fmt.Errorf("failed to connect")
					}
					output.Errf("authentication expired, please login again\n")
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
				sessionID, _, err := client.Login(username, password)
				if err != nil {
					return err
				}
				client.Session = sessionID
				client.APIToken = ""
				personalID, user, err := client.IssuePersonalAccessToken(machineID, machineLabel)
				if err != nil {
					return fmt.Errorf("personal access token: %w", err)
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
	cmd.Flags().StringVar(&tokenFlag, "token", "", "Personal Access Token (or legacy session / OEM API token)")
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
		if status == "denied" {
			return fmt.Errorf("authorization denied")
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
