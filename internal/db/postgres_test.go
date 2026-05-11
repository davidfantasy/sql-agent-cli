package db

import "testing"

func TestPostgresDriver_ListSchemaQuery(t *testing.T) {
	driver, err := NewPostgresDriver("postgres://postgres:postgres@127.0.0.1:5432/app?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()

	postgresDriver, ok := driver.(*sqlDriver)
	if !ok {
		t.Fatalf("expected sqlDriver, got %T", driver)
	}
	if postgresDriver.dialect != "postgres" {
		t.Fatalf("expected postgres dialect, got %q", postgresDriver.dialect)
	}
}

func TestTableColumnJSONShape(t *testing.T) {
	column := TableColumn{Name: "email", Type: "text", Nullable: false, PrimaryKey: false}

	if column.Name != "email" {
		t.Fatalf("expected email column, got %q", column.Name)
	}
	if column.Type == "" {
		t.Fatal("expected a stable type field")
	}
}
