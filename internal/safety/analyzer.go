package safety

import (
	"errors"
	"strings"
)

// Analysis describes the CLI safety decision for one SQL statement.
type Analysis struct {
	StatementType   string
	RequiresConfirm bool
	IsReadOnly      bool
}

func Analyze(sql string) (Analysis, error) {
	trimmed := strings.TrimSpace(sql)
	if trimmed == "" {
		return Analysis{}, errors.New("sql must not be empty")
	}

	if strings.Contains(trimmed, ";") && !strings.HasSuffix(trimmed, ";") {
		return Analysis{}, errors.New("only one SQL statement is allowed")
	}

	trimmed = strings.TrimSpace(strings.TrimSuffix(trimmed, ";"))
	fields := strings.Fields(strings.ToUpper(trimmed))
	if len(fields) == 0 {
		return Analysis{}, errors.New("sql must not be empty")
	}

	statementType := fields[0]
	if statementType == "WITH" {
		statementType = "SELECT"
	}

	analysis := Analysis{StatementType: statementType}
	if statementType == "SELECT" {
		analysis.IsReadOnly = true
	}
	if statementType == "DELETE" || statementType == "DROP" || statementType == "TRUNCATE" || statementType == "ALTER" {
		analysis.RequiresConfirm = true
	}
	if statementType == "UPDATE" && !strings.Contains(strings.ToUpper(trimmed), " WHERE ") {
		analysis.RequiresConfirm = true
	}

	return analysis, nil
}
