package charts

import (
	"github.com/spf13/cobra"
	"github.com/zydro-marine/vector-charts-cli/internal/cli"
	"github.com/zydro-marine/vector-charts-cli/internal/output"
)

func NewCommand(factory cli.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "chart",
		Aliases: []string{"charts"},
		Short:   "Manage custom chart cells",
	}
	cli.HideAliases(cmd)
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List custom cells",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _, _, client, err := factory(true)
			if err != nil {
				return err
			}
			out, items, err := client.PortalListAll("/v1/charts/listCustomCells", "cells", 200)
			if err != nil {
				return err
			}
			return output.PrintJSONOr(out, func() error {
				rows := make([][]string, 0, len(items))
				for _, item := range items {
					m, _ := item.(map[string]any)
					rows = append(rows, []string{
						output.Str(m["id"]),
						output.Str(m["cellName"]),
						output.Boolish(output.FirstNonEmpty(m["enabled"], m["active"])),
						output.Str(m["sourceType"]),
					})
				}
				return output.Table([]string{"ID", "CELL", "ENABLED", "SOURCE"}, rows)
			})
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "enable <id>",
		Short: "Enable a custom cell",
		Args:  cobra.ExactArgs(1),
		RunE:  setCellEnabled(factory, true),
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "disable <id>",
		Short: "Disable a custom cell",
		Args:  cobra.ExactArgs(1),
		RunE:  setCellEnabled(factory, false),
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "status <id>",
		Short: "Get processing status for a custom cell",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _, _, client, err := factory(true)
			if err != nil {
				return err
			}
			var out map[string]any
			if err := client.PortalJSON("GET", "/v1/charts/customCells/"+args[0]+"/status", nil, &out); err != nil {
				return err
			}
			return output.PrintJSONOr(out, func() error {
				status := output.Str(out["status"])
				return output.Table([]string{"STATUS"}, [][]string{{output.ColorState(status)}})
			})
		},
	})
	return cmd
}

func setCellEnabled(factory cli.Factory, enabled bool) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		_, _, _, client, err := factory(true)
		if err != nil {
			return err
		}
		id := args[0]
		var out map[string]any
		if err := client.PortalJSON("POST", "/v1/charts/cells/"+id+"/enabled", map[string]any{
			"enabled": enabled,
		}, &out); err != nil {
			return err
		}
		return output.PrintJSONOr(out, func() error {
			action := "Disabled"
			if enabled {
				action = "Enabled"
			}
			output.Printf("%s cell %s\n", action, id)
			return nil
		})
	}
}
