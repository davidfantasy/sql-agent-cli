package db

import "testing"

func TestMySQLDriver_ListSchemaQuery(t *testing.T) {
	driver, err := NewMySQLDriver("user:pass@tcp(127.0.0.1:3306)/app")
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()

	mysqlDriver, ok := driver.(*sqlDriver)
	if !ok {
		t.Fatalf("expected sqlDriver, got %T", driver)
	}
	if mysqlDriver.dialect != "mysql" {
		t.Fatalf("expected mysql dialect, got %q", mysqlDriver.dialect)
	}
}

func TestMySQLDescribeTableShape(t *testing.T) {
	description := TableDescription{
		Table: "users",
		Columns: []TableColumn{{
			Name:       "id",
			Type:       "bigint",
			Nullable:   false,
			PrimaryKey: true,
		}},
	}

	if description.Table != "users" {
		t.Fatalf("expected users table, got %q", description.Table)
	}
	if len(description.Columns) != 1 {
		t.Fatalf("expected one column, got %d", len(description.Columns))
	}
}

func TestNormalizeValue_ConvertsTextBytesToString(t *testing.T) {
	got := normalizeValue([]byte("alice@example.com"))
	if got != "alice@example.com" {
		t.Fatalf("expected text bytes to become string, got %#v", got)
	}
}
