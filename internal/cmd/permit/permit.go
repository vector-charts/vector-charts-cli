package permit

import (
	"github.com/spf13/cobra"
	"github.com/vector-charts/vector-charts-cli/internal/cli"
	"github.com/vector-charts/vector-charts-cli/internal/output"
)

func NewCommand(factory cli.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "permit",
		Short: "Manage S-63 permits",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "show-s63-user-permit",
		Short: "Fetch the account S-63 user permit",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _, _, client, err := factory(true)
			if err != nil {
				return err
			}
			var out any
			if err := client.PortalJSON("GET", "/v1/charts/getPermit", nil, &out); err != nil {
				return err
			}
			return output.PrintJSONOr(out, func() error {
				permitID := ""
				switch v := out.(type) {
				case string:
					permitID = v
				case map[string]any:
					permitID = output.Str(output.FirstNonEmpty(v["permitId"], v["permit"], v["id"]))
				default:
					permitID = output.Str(out)
				}
				return output.Table([]string{"PERMIT"}, [][]string{{permitID}})
			})
		},
	})
	return cmd
}
