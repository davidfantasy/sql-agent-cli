package db

import (
	"testing"
)

func TestInjectLimit_Basic(t *testing.T) {
	d := &sqlDriver{dialect: "mysql"}
	sql := d.injectLimit("SELECT * FROM users", 51, 0)
	expected := "SELECT * FROM users LIMIT 51 OFFSET 0"
	if sql != expected {
		t.Fatalf("expected %q, got %q", expected, sql)
	}
}

func TestInjectLimit_WithOffset(t *testing.T) {
	d := &sqlDriver{dialect: "postgres"}
	sql := d.injectLimit("SELECT id, name FROM orders", 51, 100)
	expected := "SELECT id, name FROM orders LIMIT 51 OFFSET 100"
	if sql != expected {
		t.Fatalf("expected %q, got %q", expected, sql)
	}
}

func TestInjectLimit_TrimsSemicolon(t *testing.T) {
	d := &sqlDriver{dialect: "mysql"}
	sql := d.injectLimit("SELECT * FROM users;", 10, 0)
	expected := "SELECT * FROM users LIMIT 10 OFFSET 0"
	if sql != expected {
		t.Fatalf("expected %q, got %q", expected, sql)
	}
}

func TestInjectLimit_SkipsIfAlreadyHasLimit(t *testing.T) {
	d := &sqlDriver{dialect: "postgres"}
	original := "SELECT * FROM users LIMIT 10"
	sql := d.injectLimit(original, 50, 0)
	if sql != original {
		t.Fatalf("expected unchanged SQL when LIMIT already present, got %q", sql)
	}
}

func TestInjectLimit_CaseInsensitive(t *testing.T) {
	d := &sqlDriver{dialect: "mysql"}
	original := "select * from users limit 5"
	sql := d.injectLimit(original, 50, 0)
	if sql != original {
		t.Fatalf("expected unchanged SQL when limit already present (case insensitive), got %q", sql)
	}
}
