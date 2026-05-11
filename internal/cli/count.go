package cli

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/davidfantasy/sql-agent-cli/internal/output"
	"github.com/davidfantasy/sql-agent-cli/internal/safety"
	"github.com/spf13/cobra"
)

var countTableTargetRegex = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)?$`)

func newCountCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "count <name> <target>",
		Short: "Get an exact count for a table or read-only query target",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := strings.TrimSpace(args[1])
			query, err := buildCountQuery(target)
			if err != nil {
				return err
			}

			resolved, err := resolveConnection(args[0])
			if err != nil {
				return err
			}
			driver, err := openDriver(resolved)
			if err != nil {
				return err
			}
			defer driver.Close()

			result, err := driver.Query(query)
			if err != nil {
				return err
			}
			return emitJSON(cmd.OutOrStdout(), output.Envelope{OK: true, Data: map[string]any{"columns": result.Columns, "rows": result.Rows}})
		},
	}
}

func buildCountQuery(target string) (string, error) {
	upper := strings.ToUpper(target)
	if strings.HasPrefix(upper, "SELECT ") || strings.HasPrefix(upper, "WITH ") {
		analysis, err := safety.Analyze(target)
		if err != nil {
			return "", err
		}
		if !analysis.IsReadOnly {
			return "", fmt.Errorf("count target must be a table name or a read-only SELECT query")
		}
		return fmt.Sprintf("SELECT COUNT(*) AS count FROM (%s) AS count_target", target), nil
	}

	if !countTableTargetRegex.MatchString(target) {
		return "", fmt.Errorf("count target must be a table name or a read-only SELECT query")
	}

	return fmt.Sprintf("SELECT COUNT(*) AS count FROM %s", target), nil
}
