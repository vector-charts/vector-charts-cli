package clicmd

import (
	"os"

	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cli",
		Short: "CLI utility commands",
	}
	cmd.AddCommand(newCompletionCmd())
	return cmd
}

func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion",
		Short: "Generate the autocompletion script for the specified shell",
		Long: `Generate the autocompletion script for vc for the specified shell.
See each sub-command's help for details on how to use the generated script.`,
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "bash",
		Short: "Generate the autocompletion script for bash",
		Long: `Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.

To load completions in your current shell session:

	source <(vc cli completion bash)

To load completions for every new session, execute once:

#### Linux:

	vc cli completion bash > /etc/bash_completion.d/vc

#### macOS:

	vc cli completion bash > $(brew --prefix)/etc/bash_completion.d/vc

You will need to start a new shell for this setup to take effect.`,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenBashCompletion(os.Stdout)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "zsh",
		Short: "Generate the autocompletion script for zsh",
		Long: `Generate the autocompletion script for the zsh shell.

To load completions in your current shell session:

	source <(vc cli completion zsh)

To load completions for every new session, execute once:

	vc cli completion zsh > "${fpath[1]}/_vc"

You will need to start a new shell for this setup to take effect.`,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenZshCompletion(os.Stdout)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "fish",
		Short: "Generate the autocompletion script for fish",
		Long: `Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	vc cli completion fish | source

To load completions for every new session, execute once:

	vc cli completion fish > ~/.config/fish/completions/vc.fish

You will need to start a new shell for this setup to take effect.`,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenFishCompletion(os.Stdout, true)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "powershell",
		Short: "Generate the autocompletion script for powershell",
		Long: `Generate the autocompletion script for powershell.

To load completions in your current shell session:

	vc cli completion powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output of the above command
to your powershell profile.`,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		},
	})

	return cmd
}
