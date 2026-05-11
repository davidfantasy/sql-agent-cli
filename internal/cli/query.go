package cli

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/davidfantasy/sql-agent-cli/internal/db"
	"github.com/davidfantasy/sql-agent-cli/internal/output"
	"github.com/davidfantasy/sql-agent-cli/internal/safety"
	"github.com/spf13/cobra"
)

func newQueryCommand() *cobra.Command {
	var confirm bool

	cmd := &cobra.Command{
		Use:   "query <name> <sql>",
		Short: "Execute one SQL statement",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			analysis, err := safety.Analyze(args[1])
			if err != nil {
				return err
			}

			if analysis.RequiresConfirm && !confirm {
				marshalErr := emitJSON(cmd.OutOrStdout(), output.Envelope{
					OK:      false,
					Data:    nil,
					Warning: nil,
					Error: output.ErrorBody{
						Code:    "destructive_query_requires_confirmation",
						Message: "statement is blocked until a human reviews the warning and explicitly approves retrying with --confirm",
						Details: map[string]any{
							"statement_type": analysis.StatementType,
							"action":         "show this warning to the human before retrying with --confirm",
						},
					},
					DurationMS: 0,
				})
				if marshalErr != nil {
					return marshalErr
				}
				return errors.New("destructive query requires confirmation")
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

			page, _ := cmd.Flags().GetInt("page")
			if page < 1 {
				page = 1
			}
			pageSize, _ := cmd.Flags().GetInt("page-size")
			if pageSize < 1 {
				pageSize = 50
			}
			explore, _ := cmd.Flags().GetBool("explore")

			if explore && analysis.StatementType == "SELECT" {
				return runExplore(cmd, driver, args[1])
			}

			if analysis.StatementType == "SELECT" {
				result, hasMore, err := driver.QueryPaginated(args[1], page, pageSize)
				if err != nil {
					return err
				}
				formatted := output.FormatRows(result.Columns, result.Rows, page, pageSize, 120)
				formatted.HasMore = hasMore
				if hasMore {
					nextPage := page + 1
					formatted.NextPage = &nextPage
				}
				return emitJSON(cmd.OutOrStdout(), output.Envelope{OK: true, Data: formatted})
			}

			execResult, err := driver.Exec(args[1])
			if err != nil {
				return err
			}
			return emitJSON(cmd.OutOrStdout(), output.Envelope{OK: true, Data: map[string]any{
				"statement_type": analysis.StatementType,
				"affected_rows":  execResult.AffectedRows,
			}})
		},
	}

	cmd.Flags().Int("page", 1, "Result page number")
	cmd.Flags().Int("page-size", 50, "Rows per page")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "Confirm dangerous write or DDL statement")
	cmd.Flags().Bool("explore", false, "Return a summary instead of full rows")

	return cmd
}

var simpleTableRegex = regexp.MustCompile(`(?i)^\s*SELECT\s+.*\s+FROM\s+(\w+)`)

func runExplore(cmd *cobra.Command, driver db.Driver, sql string) error {
	// Try to extract table name from simple queries
	match := simpleTableRegex.FindStringSubmatch(sql)
	var tableName string
	if len(match) > 1 {
		tableName = match[1]
	}

	// Get sample rows (LIMIT 3)
	result, _, err := driver.QueryPaginated(sql, 1, 3)
	if err != nil {
		return err
	}

	exploreResult := map[string]any{
		"mode":         "explore",
		"column_count": len(result.Columns),
		"columns":      result.Columns,
		"sample_rows":  result.Rows,
		"note":         "This is a summary. Use query without --explore for full results.",
	}

	if tableName != "" {
		// Try to get row count using Query (Exec does not return rows)
		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
		countRes, err := driver.Query(countQuery)
		if err != nil || len(countRes.Rows) == 0 {
			exploreResult["estimated_total_rows"] = "unknown (complex query)"
		} else {
			exploreResult["estimated_total_rows"] = countRes.Rows[0][0]
		}
	} else {
		exploreResult["estimated_total_rows"] = "unknown (complex query)"
	}

	return emitJSON(cmd.OutOrStdout(), output.Envelope{OK: true, Data: exploreResult})
}
