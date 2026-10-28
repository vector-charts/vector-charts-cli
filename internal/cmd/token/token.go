package token

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zydro-marine/vector-charts-cli/internal/cli"
	"github.com/zydro-marine/vector-charts-cli/internal/output"
)

func NewCommand(factory cli.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token",
		Short: "Manage portal API tokens",
	}
	cmd.AddCommand(newListCmd(factory))
	cmd.AddCommand(newCreateCmd(factory))
	cmd.AddCommand(newUpdateCmd(factory))
	cmd.AddCommand(newDeleteCmd(factory))
	cmd.AddCommand(newUseCmd(factory))
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
				tokens, _ := asSlice(out)
				rows := make([][]string, 0, len(tokens))
				for _, t := range tokens {
					m, _ := t.(map[string]any)
					rows = append(rows, []string{
						str(m["id"]),
						str(m["name"]),
						str(m["userEmail"]),
						str(m["createdAt"]),
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
				output.Printf("Created token %s (%s)\n", str(out["id"]), str(out["name"]))
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

func newUseCmd(factory cli.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "use <id>",
		Short: "Store an existing API token id on the current profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _, _, _, err := factory(true)
			if err != nil {
				return err
			}
			if err := cli.RequireNamedProfile(name); err != nil {
				return err
			}
			if err := cli.SaveAPIToken(name, args[0]); err != nil {
				return err
			}
			output.Printf("Profile %q now uses API token %s\n", name, args[0])
			return nil
		},
	}
}

func asSlice(v any) ([]any, bool) {
	switch t := v.(type) {
	case []any:
		return t, true
	case map[string]any:
		if arr, ok := t["tokens"].([]any); ok {
			return arr, true
		}
		if arr, ok := t["data"].([]any); ok {
			return arr, true
		}
	}
	return nil, false
}

func str(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}
