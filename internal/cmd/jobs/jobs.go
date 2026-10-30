package jobs

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zydro-marine/vector-charts-cli/internal/cli"
	"github.com/zydro-marine/vector-charts-cli/internal/output"
)

func NewCommand(factory cli.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "jobs",
		Short: "Inspect background jobs",
	}
	cmd.AddCommand(newListCmd(factory))
	cmd.AddCommand(newStatusCmd(factory))
	return cmd
}

func newListCmd(factory cli.Factory) *cobra.Command {
	var activeOnly bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List jobs",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _, _, client, err := factory(true)
			if err != nil {
				return err
			}
			path := "/v1/jobs"
			if activeOnly {
				path += "?activeOnly=true"
			}
			out, items, err := client.PortalListAll(path, "jobs", 200)
			if err != nil {
				return err
			}
			return output.PrintJSONOr(out, func() error {
				rows := make([][]string, 0, len(items))
				for _, item := range items {
					m, _ := item.(map[string]any)
					rows = append(rows, []string{
						output.Str(m["id"]),
						output.Str(m["jobType"]),
						output.Str(m["jobState"]),
						output.Time(m["createdAt"]),
					})
				}
				return output.Table([]string{"ID", "TYPE", "STATE", "CREATED"}, rows)
			})
		},
	}
	cmd.Flags().BoolVar(&activeOnly, "active-only", false, "Only active jobs")
	return cmd
}

func shortJobID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

func jobPercent(m map[string]any) float64 {
	switch v := m["percentComplete"].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0
	}
}

func colorJobLine(state, line string) string {
	switch strings.ToLower(state) {
	case "running", "active":
		return output.Green(line)
	case "pending":
		return output.PaleGrey(line)
	case "failed", "error":
		return output.Red(line)
	default:
		return line
	}
}

func formatJobLine(m map[string]any) string {
	name := output.Str(m["jobType"])
	if name == "" {
		name = "job"
	}
	id := shortJobID(output.Str(m["id"]))
	state := output.Str(m["jobState"])
	percent := fmt.Sprintf("%.0f%%", jobPercent(m))
	started := output.Time(m["createdAt"])
	if started == "" {
		started = "-"
	}

	var line string
	if id != "" {
		line = fmt.Sprintf("%s (%s): %s, %s, %s", name, id, state, percent, started)
	} else {
		line = fmt.Sprintf("%s: %s, %s, %s", name, state, percent, started)
	}
	return colorJobLine(state, line)
}

func newStatusCmd(factory cli.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show active jobs with progress",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _, _, client, err := factory(true)
			if err != nil {
				return err
			}

			var summary map[string]any
			if err := client.PortalJSON("GET", "/v1/jobs/getJobStatus", nil, &summary); err != nil {
				return err
			}

			_, items, err := client.PortalListAll("/v1/jobs?activeOnly=true", "jobs", 200)
			if err != nil {
				return err
			}

			payload := map[string]any{
				"activeJobs":      summary["activeJobs"],
				"percentComplete": summary["percentComplete"],
				"isEditable":      summary["isEditable"],
				"jobs":            items,
			}

			return output.PrintJSONOr(payload, func() error {
				active, _ := summary["activeJobs"].(float64)
				pct, _ := summary["percentComplete"].(float64)

				summaryLine := fmt.Sprintf("%d active, %.0f%% complete", int(active), pct)
				jobLines := "no active jobs"
				if len(items) > 0 {
					lines := make([]string, 0, len(items))
					for _, item := range items {
						m, _ := item.(map[string]any)
						lines = append(lines, formatJobLine(m))
					}
					jobLines = strings.Join(lines, "\n")
				}

				return output.TableSeparated(nil, [][]string{
					{"jobs"},
					{summaryLine},
					{jobLines},
				})
			})
		},
	}
}
