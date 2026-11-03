package uploads

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zydro-marine/vector-charts-cli/internal/cli"
	"github.com/zydro-marine/vector-charts-cli/internal/output"
)

// Portal uploads.allowedExtensions.
var allowedExtensions = map[string]struct{}{
	".000": {},
	".zip": {},
	".txt": {},
	".prm": {},
	".h5":  {},
	".vcm": {},
}

func NewCommand(factory cli.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "upload",
		Aliases: []string{"uploads"},
		Short:   "Create, list, status, sync, ftp-sync, or delete chart uploads",
	}
	cli.HideAliases(cmd)
	cmd.AddCommand(newCreateCmd(factory))
	cmd.AddCommand(newListCmd(factory))
	cmd.AddCommand(newStatusCmd(factory))
	cmd.AddCommand(newSyncCmd(factory))
	cmd.AddCommand(newFTPSyncCmd(factory))
	cmd.AddCommand(newDeleteCmd(factory))
	return cmd
}

func newCreateCmd(factory cli.Factory) *cobra.Command {
	return &cobra.Command{
		Use:     "create <file>",
		Aliases: []string{"add"},
		Short:   "Upload a chart file for ingestion",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _, _, client, err := factory(true)
			if err != nil {
				return err
			}
			out, err := client.UploadChart(args[0])
			if err != nil {
				return err
			}
			return output.PrintJSONOr(out, func() error {
				output.Printf("Upload queued: id=%v status=%s\n", out["id"], output.ColorState(output.Str(out["status"])))
				return nil
			})
		},
	}
}

func newListCmd(factory cli.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List uploads",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _, _, client, err := factory(true)
			if err != nil {
				return err
			}
			out, items, err := client.PortalListAll("/v1/charts/uploads", "uploads", 200)
			if err != nil {
				return err
			}
			return output.PrintJSONOr(out, func() error {
				rows := make([][]string, 0, len(items))
				for _, item := range items {
					m, _ := item.(map[string]any)
					rows = append(rows, []string{
						output.Str(m["id"]),
						output.Str(m["filename"]),
						output.ColorState(output.Str(m["status"])),
						output.Time(m["createdAt"]),
					})
				}
				return output.Table([]string{"ID", "FILENAME", "STATUS", "CREATED"}, rows)
			})
		},
	}
}

func newStatusCmd(factory cli.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "status <id>",
		Short: "Show status info for an upload",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _, _, client, err := factory(true)
			if err != nil {
				return err
			}
			var out map[string]any
			if err := client.PortalJSON("GET", "/v1/charts/uploads/"+args[0], nil, &out); err != nil {
				return err
			}
			upload := out
			if nested, ok := out["upload"].(map[string]any); ok {
				upload = nested
			}
			return output.PrintJSONOr(out, func() error {
				rows := [][]string{
					{"ID", output.Str(upload["id"])},
					{"FILENAME", output.Str(upload["filename"])},
					{"STATUS", output.ColorState(output.Str(upload["status"]))},
					{"SIZE", formatBytes(upload["fileSize"])},
					{"CHARTS", output.Str(upload["totalCharts"])},
					{"EXPIRED", output.Str(upload["expiredCharts"])},
					{"UPLOADER", output.Str(upload["uploaderEmail"])},
					{"CREATED", output.Time(upload["createdAt"])},
					{"UPDATED", output.Time(upload["updatedAt"])},
				}
				if err := output.Table([]string{"FIELD", "VALUE"}, rows); err != nil {
					return err
				}
				if logs := output.ExtractList(upload["logs"]); len(logs) > 0 {
					output.Printf("\nLogs:\n")
					for _, line := range logs {
						output.Printf("  %s\n", output.Str(line))
					}
				}
				return nil
			})
		},
	}
}

func newDeleteCmd(factory cli.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an upload",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _, _, client, err := factory(true)
			if err != nil {
				return err
			}
			if err := client.PortalJSON("DELETE", "/v1/charts/uploads/"+args[0], nil, nil); err != nil {
				return err
			}
			output.Printf("Deleted upload %s\n", args[0])
			return nil
		},
	}
}

func formatBytes(v any) string {
	var n float64
	switch t := v.(type) {
	case float64:
		n = t
	case int64:
		n = float64(t)
	case int:
		n = float64(t)
	default:
		return output.Str(v)
	}
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%.0f B", n)
	}
	div, exp := float64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", n/div, "KMGTPE"[exp])
}
