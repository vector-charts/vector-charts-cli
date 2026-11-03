package clicmd

import (
	"os"

	"github.com/spf13/cobra"
)

func NewCommand(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cli",
		Short: "CLI utility commands",
	}
	cmd.AddCommand(newCheckUpdateCmd(version))
	cmd.AddCommand(newCompletionCmd())
	return cmd
}

func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion",
		Short: "Generate the autocompletion script for the specified shell",
	}

	cmd.AddCommand(&cobra.Command{
		Use:               "bash",
		Short:             "Generate the autocompletion script for bash",
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenBashCompletion(os.Stdout)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:               "zsh",
		Short:             "Generate the autocompletion script for zsh",
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenZshCompletion(os.Stdout)
		},
	})

	return cmd
}
