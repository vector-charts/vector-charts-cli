package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vector-charts/vector-charts-cli/internal/cli"
	"github.com/vector-charts/vector-charts-cli/internal/cmd/charts"
	"github.com/vector-charts/vector-charts-cli/internal/cmd/clicmd"
	"github.com/vector-charts/vector-charts-cli/internal/cmd/jobs"
	"github.com/vector-charts/vector-charts-cli/internal/cmd/permit"
	"github.com/vector-charts/vector-charts-cli/internal/cmd/permits"
	"github.com/vector-charts/vector-charts-cli/internal/cmd/server"
	"github.com/vector-charts/vector-charts-cli/internal/cmd/store"
	"github.com/vector-charts/vector-charts-cli/internal/cmd/token"
	"github.com/vector-charts/vector-charts-cli/internal/cmd/uploads"
	"github.com/vector-charts/vector-charts-cli/internal/config"
	"github.com/vector-charts/vector-charts-cli/internal/httpx"
	"github.com/vector-charts/vector-charts-cli/internal/output"
)

var (
	Version   = "0.1.0"
	profile   string
	outputFmt string
)

func NewRoot() *cobra.Command {
	root := &cobra.Command{
		Use:           "vc",
		Short:         "Vector Charts command-line interface",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return output.SetFormat(outputFmt)
		},
	}
	root.CompletionOptions.DisableDefaultCmd = true
	// Disable the standalone "help" command; --help remains available.
	root.SetHelpCommand(&cobra.Command{Use: "__help_disabled", Hidden: true})

	root.PersistentFlags().StringVar(&profile, "profile", "", "Profile name (default: sole profile, current_profile, or \"default\")")
	root.PersistentFlags().StringVarP(&outputFmt, "output", "o", "table", "Output format: table|json")

	root.AddCommand(newVersionCmd())
	root.AddCommand(server.NewCommand(ResolveClient))
	root.AddCommand(token.NewCommand(ResolveClient))
	root.AddCommand(charts.NewCommand(ResolveClient))
	root.AddCommand(uploads.NewCommand(ResolveClient))
	root.AddCommand(permit.NewCommand(ResolveClient))
	root.AddCommand(permits.NewCommand(ResolveClient))
	root.AddCommand(store.NewCommand(ResolveClient))
	root.AddCommand(jobs.NewCommand(ResolveClient))
	root.AddCommand(clicmd.NewCommand())

	return root
}

func ResolveClient(requireAuth bool) (string, config.Profile, config.Creds, *httpx.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return "", config.Profile{}, config.Creds{}, nil, err
	}
	credsFile, err := config.LoadCredentials()
	if err != nil {
		return "", config.Profile{}, config.Creds{}, nil, err
	}

	name, p, err := cfg.Resolve(profile)
	if err != nil {
		return "", config.Profile{}, config.Creds{}, nil, err
	}
	p = p.Normalized()
	creds := credsFile.Profiles[name]
	output.Connecting(name)
	client := httpx.New(p, creds)

	if requireAuth {
		if creds.APIToken == "" && creds.SessionToken == "" {
			return "", config.Profile{}, config.Creds{}, nil, fmt.Errorf("authentication expired, please login again")
		}
		if _, err := client.WhoAmI(); err != nil {
			return "", config.Profile{}, config.Creds{}, nil, httpx.ClassifyWhoAmIError(err)
		}
	}

	return name, p, creds, client, nil
}

var _ cli.Factory = ResolveClient

func Execute() {
	root := NewRoot()
	if err := root.Execute(); err != nil {
		output.Errf("error: %v\n", err)
		os.Exit(1)
	}
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print CLI version",
		Run: func(cmd *cobra.Command, args []string) {
			if output.Format == "json" {
				_ = output.JSON(map[string]string{"version": Version})
				return
			}
			output.Println(Version)
		},
	}
}
