package db

import (
	"database/sql"
	"fmt"
	"strings"
)

type QueryOutput struct {
	Columns []string
	Rows    [][]any
}

type ExecOutput struct {
	AffectedRows int64
}

type TableColumn struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Nullable   bool   `json:"nullable"`
	PrimaryKey bool   `json:"primary_key"`
}

type TableDescription struct {
	Table   string        `json:"table"`
	Columns []TableColumn `json:"columns"`
}

type Driver interface {
	Query(sql string) (QueryOutput, error)
	QueryPaginated(sql string, page, pageSize int) (QueryOutput, bool, error)
	Exec(sql string) (ExecOutput, error)
	ListSchema() ([]string, error)
	DescribeTable(name string) (any, error)
	Close() error
}

type sqlDriver struct {
	db      *sql.DB
	dialect string
}

func (d *sqlDriver) Query(query string) (QueryOutput, error) {
	rows, err := d.db.Query(query)
	if err != nil {
		return QueryOutput{}, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return QueryOutput{}, err
	}

	out := QueryOutput{Columns: columns}
	for rows.Next() {
		values := make([]any, len(columns))
		targets := make([]any, len(columns))
		for i := range values {
			targets[i] = &values[i]
		}
		if err := rows.Scan(targets...); err != nil {
			return QueryOutput{}, err
		}
		for i := range values {
			values[i] = normalizeValue(values[i])
		}
		out.Rows = append(out.Rows, values)
	}

	if err := rows.Err(); err != nil {
		return QueryOutput{}, err
	}
	return out, nil
}

func (d *sqlDriver) Exec(statement string) (ExecOutput, error) {
	result, err := d.db.Exec(statement)
	if err != nil {
		return ExecOutput{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return ExecOutput{}, err
	}
	return ExecOutput{AffectedRows: affected}, nil
}

func (d *sqlDriver) ListSchema() ([]string, error) {
	query := ""
	switch d.dialect {
	case "mysql":
		query = "SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE() ORDER BY table_name"
	case "postgres":
		query = "SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' ORDER BY table_name"
	default:
		return nil, fmt.Errorf("unsupported dialect %q", d.dialect)
	}

	result, err := d.Query(query)
	if err != nil {
		return nil, err
	}

	tables := make([]string, 0, len(result.Rows))
	for _, row := range result.Rows {
		if len(row) == 0 {
			continue
		}
		if name, ok := row[0].(string); ok {
			tables = append(tables, name)
			continue
		}
		if data, ok := row[0].([]byte); ok {
			tables = append(tables, string(data))
		}
	}
	return tables, nil
}

func (d *sqlDriver) DescribeTable(name string) (any, error) {
	query := ""
	args := []any{name}
	switch d.dialect {
	case "mysql":
		query = `SELECT c.column_name, c.column_type, c.is_nullable = 'YES' AS nullable,
			COALESCE(k.column_name IS NOT NULL, FALSE) AS primary_key
		FROM information_schema.columns c
		LEFT JOIN information_schema.key_column_usage k
			ON c.table_schema = k.table_schema
			AND c.table_name = k.table_name
			AND c.column_name = k.column_name
			AND k.constraint_name = 'PRIMARY'
		WHERE c.table_schema = DATABASE() AND c.table_name = ?
		ORDER BY c.ordinal_position`
	case "postgres":
		query = `SELECT c.column_name, c.data_type, c.is_nullable = 'YES' AS nullable,
			COALESCE(tc.constraint_type = 'PRIMARY KEY', FALSE) AS primary_key
		FROM information_schema.columns c
		LEFT JOIN information_schema.key_column_usage kcu
			ON c.table_schema = kcu.table_schema
			AND c.table_name = kcu.table_name
			AND c.column_name = kcu.column_name
		LEFT JOIN information_schema.table_constraints tc
			ON kcu.constraint_name = tc.constraint_name
			AND kcu.table_schema = tc.table_schema
		WHERE c.table_schema = 'public' AND c.table_name = $1
		ORDER BY c.ordinal_position`
	default:
		return nil, fmt.Errorf("unsupported dialect %q", d.dialect)
	}

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	description := TableDescription{Table: name}
	for rows.Next() {
		var column TableColumn
		if err := rows.Scan(&column.Name, &column.Type, &column.Nullable, &column.PrimaryKey); err != nil {
			return nil, err
		}
		description.Columns = append(description.Columns, column)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return description, nil
}

func (d *sqlDriver) QueryPaginated(query string, page, pageSize int) (QueryOutput, bool, error) {
	// Fetch one extra row to determine if there are more results
	limit := pageSize + 1
	offset := (page - 1) * pageSize

	paginatedSQL := d.injectLimit(query, limit, offset)
	out, err := d.Query(paginatedSQL)
	if err != nil {
		return QueryOutput{}, false, err
	}

	hasMore := len(out.Rows) > pageSize
	if hasMore {
		out.Rows = out.Rows[:pageSize]
	}

	return out, hasMore, nil
}

func (d *sqlDriver) injectLimit(sql string, limit, offset int) string {
	sql = strings.TrimSpace(sql)
	sql = strings.TrimSuffix(sql, ";")

	// Defensive: if user already wrote a LIMIT clause, do not double-inject.
	upper := strings.ToUpper(sql)
	if strings.Contains(upper, "LIMIT") {
		return sql
	}

	switch d.dialect {
	case "mysql", "postgres":
		// Both MySQL and PostgreSQL support LIMIT ... OFFSET ...
		return fmt.Sprintf("%s LIMIT %d OFFSET %d", sql, limit, offset)
	default:
		return sql
	}
}

func (d *sqlDriver) Close() error {
	return d.db.Close()
}

func normalizeValue(value any) any {
	if data, ok := value.([]byte); ok {
		return string(data)
	}
	return value
}
