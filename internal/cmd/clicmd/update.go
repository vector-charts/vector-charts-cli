package clicmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/vector-charts/vector-charts-cli/internal/output"
)

const (
	githubRepo     = "vector-charts/vector-charts-cli"
	installCommand = `curl -fsSL https://raw.githubusercontent.com/vector-charts/vector-charts-cli/main/scripts/install.sh | bash`
)

func newCheckUpdateCmd(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "check-update",
		Short: "Check GitHub for a newer vc release",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			latest, err := fetchLatestReleaseTag()
			if err != nil {
				return err
			}
			current := normalizeVersion(version)
			remote := normalizeVersion(latest)
			update := compareVersions(current, remote) < 0

			payload := map[string]any{
				"current":         current,
				"latest":          remote,
				"updateAvailable": update,
			}
			if update {
				payload["install"] = installCommand
			}

			return output.PrintJSONOr(payload, func() error {
				if !update {
					output.Printf("vc is up to date (%s)\n", current)
					return nil
				}
				output.Printf("Update available: %s -> %s\n\n", current, remote)
				output.Printf("Install with:\n%s\n", installCommand)
				return nil
			})
		},
	}
}

func fetchLatestReleaseTag() (string, error) {
	url := "https://api.github.com/repos/" + githubRepo + "/releases/latest"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "vector-charts-cli")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("check update: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("check update: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("check update: GitHub API HTTP %d", resp.StatusCode)
	}
	var parsed struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("check update: parse release: %w", err)
	}
	if strings.TrimSpace(parsed.TagName) == "" {
		return "", fmt.Errorf("check update: latest release has no tag")
	}
	return parsed.TagName, nil
}

func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	return v
}

// compareVersions returns -1 if a < b, 0 if equal, 1 if a > b.
func compareVersions(a, b string) int {
	pa := versionParts(a)
	pb := versionParts(b)
	n := len(pa)
	if len(pb) > n {
		n = len(pb)
	}
	for i := 0; i < n; i++ {
		var x, y int
		if i < len(pa) {
			x = pa[i]
		}
		if i < len(pb) {
			y = pb[i]
		}
		if x < y {
			return -1
		}
		if x > y {
			return 1
		}
	}
	return 0
}

func versionParts(v string) []int {
	parts := strings.Split(normalizeVersion(v), ".")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			out = append(out, 0)
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			out = append(out, 0)
			continue
		}
		out = append(out, n)
	}
	return out
}
