package store

import (
	"github.com/spf13/cobra"
	"github.com/zydro-marine/vector-charts-cli/internal/cli"
	"github.com/zydro-marine/vector-charts-cli/internal/output"
)

func NewCommand(factory cli.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "store",
		Aliases: []string{"subscription", "subscriptions"},
		Short:   "Manage chart store subscriptions",
	}
	cli.HideAliases(cmd)
	cmd.AddCommand(newListCmd(factory))
	cmd.AddCommand(newEnableCmd(factory))
	cmd.AddCommand(newDisableCmd(factory))
	return cmd
}

func newListCmd(factory cli.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List chart store products and subscription state",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _, _, client, err := factory(true)
			if err != nil {
				return err
			}
			var out any
			if err := client.PortalJSON("GET", "/v1/store/products", nil, &out); err != nil {
				return err
			}
			return output.PrintJSONOr(out, func() error {
				items := output.ExtractList(out, "products", "data")
				rows := make([][]string, 0, len(items))
				for _, item := range items {
					m, _ := item.(map[string]any)
					rows = append(rows, []string{
						output.Str(m["code"]),
						output.Str(m["displayName"]),
						productEnabled(m),
						output.Time(m["expiresAt"]),
						output.OptionalBool(m["willRenew"]),
						output.Price(m["price"]),
					})
				}
				return output.Table(
					[]string{"CODE", "NAME", "ENABLED", "EXPIRES", "WILL-RENEW", "PRICE"},
					rows,
				)
			})
		},
	}
}

func newEnableCmd(factory cli.Factory) *cobra.Command {
	return &cobra.Command{
		Use:     "enable <product-code>",
		Aliases: []string{"subscribe"},
		Short:   "Enable (subscribe to) a chart store product",
		Args:    cobra.ExactArgs(1),
		RunE:    setSubscription(factory, true),
	}
}

func newDisableCmd(factory cli.Factory) *cobra.Command {
	return &cobra.Command{
		Use:     "disable <product-code>",
		Aliases: []string{"unsubscribe", "cancel"},
		Short:   "Disable (unsubscribe from) a chart store product",
		Args:    cobra.ExactArgs(1),
		RunE:    setSubscription(factory, false),
	}
}

func setSubscription(factory cli.Factory, enable bool) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		_, _, _, client, err := factory(true)
		if err != nil {
			return err
		}
		code := args[0]
		path := "/v1/store/cancel"
		action := "Disabled"
		if enable {
			path = "/v1/store/checkout"
			action = "Enabled"
		}
		var out map[string]any
		if err := client.PortalJSON("POST", path, map[string]any{
			"productCode": code,
		}, &out); err != nil {
			return err
		}
		return output.PrintJSONOr(out, func() error {
			output.Printf("%s store subscription %s\n", action, code)
			return nil
		})
	}
}

// Cloud products use "subscribed"; OEM products use "enabled".
func productEnabled(m map[string]any) string {
	if m == nil {
		return "false"
	}
	if v, ok := m["enabled"]; ok && v != nil {
		return output.Boolish(v)
	}
	if v, ok := m["subscribed"]; ok && v != nil {
		return output.Boolish(v)
	}
	return "false"
}
