package cli

import "github.com/spf13/cobra"

// HideAliases keeps aliases working for invocation and completion, but omits
// them from help/usage so plural compatibility names stay undocumented.
func HideAliases(cmd *cobra.Command) {
	help := cmd.HelpFunc()
	cmd.SetHelpFunc(func(c *cobra.Command, args []string) {
		aliases := c.Aliases
		c.Aliases = nil
		defer func() { c.Aliases = aliases }()
		help(c, args)
	})
	usage := cmd.UsageFunc()
	cmd.SetUsageFunc(func(c *cobra.Command) error {
		aliases := c.Aliases
		c.Aliases = nil
		defer func() { c.Aliases = aliases }()
		return usage(c)
	})
}
