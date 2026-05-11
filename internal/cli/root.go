package cli

import "github.com/spf13/cobra"

// NewRootCommand wires the top-level command tree.
func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sql-agent",
		Short: "AI-friendly SQL database CLI",
	}

	cmd.AddCommand(
		newConnectCommand(),
		newDisconnectCommand(),
		newQueryCommand(),
		newSchemaCommand(),
		newCountCommand(),
	)

	return cmd
}
