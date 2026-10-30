package permits

import (
	"github.com/spf13/cobra"
	"github.com/zydro-marine/vector-charts-cli/internal/cli"
	"github.com/zydro-marine/vector-charts-cli/internal/output"
)

func NewCommand(factory cli.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "permits",
		Aliases: []string{"licenses"},
		Short:   "List account chart licenses and S-63 cell permits",
	}
	cmd.AddCommand(newListCmd(factory))
	return cmd
}

func newListCmd(factory cli.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List store licenses and S-63 cell permits",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _, _, client, err := factory(true)
			if err != nil {
				return err
			}
			var out any
			if err := client.PortalJSON("GET", "/v1/permits", nil, &out); err != nil {
				return err
			}
			return output.PrintJSONOr(out, func() error {
				items := output.ExtractList(out, "permits", "data")
				rows := make([][]string, 0, len(items))
				for _, item := range items {
					m, _ := item.(map[string]any)
					rows = append(rows, []string{
						output.Str(m["id"]),
						output.CellsLabel(m),
						output.Str(m["type"]),
						output.Time(m["expiresAt"]),
						output.Boolish(m["autoRenew"]),
					})
				}
				return output.Table([]string{"ID", "CELLS", "TYPE", "EXPIRES", "AUTO-RENEW"}, rows)
			})
		},
	}
}
