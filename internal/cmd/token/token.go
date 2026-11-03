package token

import (
	"github.com/spf13/cobra"
	"github.com/vector-charts/vector-charts-cli/internal/cli"
	"github.com/vector-charts/vector-charts-cli/internal/output"
)

func NewCommand(factory cli.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "token",
		Aliases: []string{"tokens"},
		Short:   "Manage portal API tokens",
	}
	cli.HideAliases(cmd)
	cmd.AddCommand(newListCmd(factory))
	cmd.AddCommand(newCreateCmd(factory))
	cmd.AddCommand(newUpdateCmd(factory))
	cmd.AddCommand(newDeleteCmd(factory))
	return cmd
}

func newListCmd(factory cli.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List API tokens",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _, _, client, err := factory(true)
			if err != nil {
				return err
			}
			var out any
			if err := client.PortalJSON("GET", "/v1/apiTokens", nil, &out); err != nil {
				return err
			}
			return output.PrintJSONOr(out, func() error {
				tokens := output.ExtractList(out, "tokens", "data")
				rows := make([][]string, 0, len(tokens))
				for _, t := range tokens {
					m, _ := t.(map[string]any)
					rows = append(rows, []string{
						output.Str(m["id"]),
						output.Str(m["name"]),
						output.Str(m["userEmail"]),
						output.Time(m["createdAt"]),
					})
				}
				return output.Table([]string{"ID", "NAME", "USER", "CREATED"}, rows)
			})
		},
	}
}

func newCreateCmd(factory cli.Factory) *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an API token",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _, _, client, err := factory(true)
			if err != nil {
				return err
			}
			if name == "" {
				name = "CLI Token"
			}
			var out map[string]any
			if err := client.PortalJSON("POST", "/v1/apiTokens", map[string]any{
				"name":         name,
				"restrictions": map[string]any{},
			}, &out); err != nil {
				return err
			}
			return output.PrintJSONOr(out, func() error {
				output.Printf("Created token %s (%s)\n", output.Str(out["id"]), output.Str(out["name"]))
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "CLI Token", "Token name")
	return cmd
}

func newUpdateCmd(factory cli.Factory) *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update an API token name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _, _, client, err := factory(true)
			if err != nil {
				return err
			}
			body := map[string]any{}
			if name != "" {
				body["name"] = name
			}
			var out map[string]any
			if err := client.PortalJSON("POST", "/v1/apiTokens/"+args[0], body, &out); err != nil {
				return err
			}
			return output.PrintJSONOr(out, func() error {
				output.Printf("Updated token %s\n", args[0])
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New token name")
	return cmd
}

func newDeleteCmd(factory cli.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an API token",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _, _, client, err := factory(true)
			if err != nil {
				return err
			}
			if err := client.PortalJSON("DELETE", "/v1/apiTokens/"+args[0], nil, nil); err != nil {
				return err
			}
			output.Printf("Deleted token %s\n", args[0])
			return nil
		},
	}
}
