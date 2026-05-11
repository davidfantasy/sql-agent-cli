package cli

import (
	"github.com/davidfantasy/sql-agent-cli/internal/output"
	"github.com/spf13/cobra"
)

func newSchemaCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Inspect database schema",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list <name>",
			Short: "List tables and views",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				resolved, err := resolveConnection(args[0])
				if err != nil {
					return err
				}
				driver, err := openDriver(resolved)
				if err != nil {
					return err
				}
				defer driver.Close()

				tables, err := driver.ListSchema()
				if err != nil {
					return err
				}
				return emitJSON(cmd.OutOrStdout(), output.Envelope{OK: true, Data: map[string]any{"tables": tables}})
			},
		},
		&cobra.Command{
			Use:   "describe <name> <table>",
			Short: "Describe one table",
			Args:  cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				resolved, err := resolveConnection(args[0])
				if err != nil {
					return err
				}
				driver, err := openDriver(resolved)
				if err != nil {
					return err
				}
				defer driver.Close()

				description, err := driver.DescribeTable(args[1])
				if err != nil {
					return err
				}
				return emitJSON(cmd.OutOrStdout(), output.Envelope{OK: true, Data: description})
			},
		},
	)

	return cmd
}
