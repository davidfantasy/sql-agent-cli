package cli

import (
	"fmt"
	"strings"

	"github.com/david/sql-agent-cli/internal/output"
	"github.com/spf13/cobra"
)

func newCountCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "count <name> <target>",
		Short: "Get an exact count for a table or query target",
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

			target := args[1]
			query := fmt.Sprintf("SELECT COUNT(*) AS count FROM %s", target)
			trimmed := strings.TrimSpace(strings.ToUpper(target))
			if strings.HasPrefix(trimmed, "SELECT ") || strings.HasPrefix(trimmed, "WITH ") {
				query = fmt.Sprintf("SELECT COUNT(*) AS count FROM (%s) AS count_target", target)
			}

			result, err := driver.Query(query)
			if err != nil {
				return err
			}
			return emitJSON(cmd.OutOrStdout(), output.Envelope{OK: true, Data: map[string]any{"columns": result.Columns, "rows": result.Rows}})
		},
	}
}
