package charts

import (
	"github.com/spf13/cobra"
	"github.com/zydro-marine/vector-charts-cli/internal/cli"
	"github.com/zydro-marine/vector-charts-cli/internal/output"
)

func NewCommand(factory cli.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "charts",
		Aliases: []string{"chart"},
		Short:   "Manage chart uploads and custom cells",
	}
	cmd.AddCommand(newUploadsCmd(factory))
	cmd.AddCommand(newCellsCmd(factory))
	cmd.AddCommand(newPermitCmd(factory))
	return cmd
}

func newUploadsCmd(factory cli.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "uploads",
		Aliases: []string{"upload"},
		Short:   "Create, list, get, or delete chart uploads",
	}
	cmd.AddCommand(&cobra.Command{
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
				output.Printf("Upload queued: id=%v status=%v\n", out["id"], out["status"])
				return nil
			})
		},
	})
	cmd.AddCommand(&cobra.Command{
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
						output.Str(m["status"]),
						output.Time(m["createdAt"]),
					})
				}
				return output.Table([]string{"ID", "FILENAME", "STATUS", "CREATED"}, rows)
			})
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "get <id>",
		Short: "Get one upload",
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
			return output.PrintJSONOr(out, func() error {
				return output.JSON(out)
			})
		},
	})
	cmd.AddCommand(&cobra.Command{
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
	})
	return cmd
}

func newCellsCmd(factory cli.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cells",
		Short: "Manage custom chart cells",
	}
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
				return output.JSON(out)
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

func newPermitCmd(factory cli.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "permit",
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
	}
}
