package cmd

import "github.com/spf13/cobra"

// Commands renamed in 2.0, such as assets to devices, keep their old names as
// Cobra aliases so existing scripts still run. Help shows only the current
// names, and shell completion never offers aliases.
func hideAliasesInHelp(root *cobra.Command) {
	usage := root.UsageFunc()
	root.SetUsageFunc(func(c *cobra.Command) error {
		aliases := c.Aliases
		c.Aliases = nil
		defer func() { c.Aliases = aliases }()
		return usage(c)
	})
}

func init() {
	hideAliasesInHelp(rootCmd)
}
