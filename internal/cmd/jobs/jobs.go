package jobs

import (
	"fmt"
	"strconv"

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
	var limit, offset int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List jobs",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _, _, client, err := factory(true)
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/v1/jobs?limit=%d&offset=%d", limit, offset)
			if activeOnly {
				path += "&activeOnly=true"
			}
			var out any
			if err := client.PortalJSON("GET", path, nil, &out); err != nil {
				return err
			}
			return output.PrintJSONOr(out, func() error {
				items := extractList(out, "jobs")
				rows := make([][]string, 0, len(items))
				for _, item := range items {
					m, _ := item.(map[string]any)
					rows = append(rows, []string{
						str(m["id"]),
						str(m["jobType"]),
						str(m["jobState"]),
						str(m["createdAt"]),
					})
				}
				return output.Table([]string{"ID", "TYPE", "STATE", "CREATED"}, rows)
			})
		},
	}
	cmd.Flags().BoolVar(&activeOnly, "active-only", false, "Only active jobs")
	cmd.Flags().IntVar(&limit, "limit", 25, "Page size")
	cmd.Flags().IntVar(&offset, "offset", 0, "Page offset")
	return cmd
}

func newStatusCmd(factory cli.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show active job count and editability",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _, _, client, err := factory(true)
			if err != nil {
				return err
			}
			var out map[string]any
			if err := client.PortalJSON("GET", "/v1/jobs/getJobStatus", nil, &out); err != nil {
				return err
			}
			return output.PrintJSONOr(out, func() error {
				output.Printf("activeJobs: %v\n", out["activeJobs"])
				output.Printf("isEditable: %v\n", out["isEditable"])
				return nil
			})
		},
	}
}

func extractList(v any, keys ...string) []any {
	switch t := v.(type) {
	case []any:
		return t
	case map[string]any:
		for _, k := range keys {
			if arr, ok := t[k].([]any); ok {
				return arr
			}
		}
	}
	return nil
}

func str(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
	}
	return fmt.Sprint(v)
}
