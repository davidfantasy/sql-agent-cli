package cli

import (
	"github.com/davidfantasy/sql-agent-cli/internal/config"
	"github.com/spf13/cobra"
)

func newDisconnectCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "disconnect <name>",
		Short: "Remove a named connection",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return config.DeleteConnection(args[0])
		},
	}
}
