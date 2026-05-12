package safety

import "testing"

func TestAnalyze_RejectsMultipleStatements(t *testing.T) {
	_, err := Analyze("SELECT 1; SELECT 2")
	if err == nil {
		t.Fatal("expected multi-statement query to fail")
	}
}

func TestAnalyze_RequiresConfirmForDelete(t *testing.T) {
	result, err := Analyze("DELETE FROM users")
	if err != nil {
		t.Fatal(err)
	}
	if !result.RequiresConfirm {
		t.Fatal("expected DELETE to require confirmation")
	}
}

func TestAnalyze_AllowsSimpleSelect(t *testing.T) {
	result, err := Analyze("SELECT id FROM users")
	if err != nil {
		t.Fatal(err)
	}
	if result.RequiresConfirm {
		t.Fatal("did not expect select to require confirmation")
	}
}

func TestAnalyze_RequiresConfirmForUpdateWithoutWhere(t *testing.T) {
	result, err := Analyze("UPDATE users SET active = false")
	if err != nil {
		t.Fatal(err)
	}
	if !result.RequiresConfirm {
		t.Fatal("expected UPDATE without WHERE to require confirmation")
	}
}

func TestAnalyze_SelectIsReadOnly(t *testing.T) {
	result, err := Analyze("SELECT id FROM users")
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsReadOnly {
		t.Fatal("expected SELECT to be read-only")
	}
}

func TestAnalyze_ShowTablesIsReadOnly(t *testing.T) {
	result, err := Analyze("SHOW TABLES")
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsReadOnly {
		t.Fatal("expected SHOW TABLES to be read-only")
	}
}

func TestAnalyze_DescribeIsReadOnly(t *testing.T) {
	result, err := Analyze("DESCRIBE users")
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsReadOnly {
		t.Fatal("expected DESCRIBE to be read-only")
	}
}

func TestAnalyze_ExplainIsReadOnly(t *testing.T) {
	result, err := Analyze("EXPLAIN SELECT * FROM users")
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsReadOnly {
		t.Fatal("expected EXPLAIN to be read-only")
	}
}

func TestAnalyze_InsertIsNotReadOnly(t *testing.T) {
	result, err := Analyze("INSERT INTO users (email) VALUES ('test@example.com')")
	if err != nil {
		t.Fatal(err)
	}
	if result.IsReadOnly {
		t.Fatal("expected INSERT to not be read-only")
	}
}

func TestAnalyze_DeleteIsNotReadOnly(t *testing.T) {
	result, err := Analyze("DELETE FROM users WHERE id = 1")
	if err != nil {
		t.Fatal(err)
	}
	if result.IsReadOnly {
		t.Fatal("expected DELETE to not be read-only")
	}
}

func TestAnalyze_UpdateIsNotReadOnly(t *testing.T) {
	result, err := Analyze("UPDATE users SET active = true WHERE id = 1")
	if err != nil {
		t.Fatal(err)
	}
	if result.IsReadOnly {
		t.Fatal("expected UPDATE to not be read-only")
	}
}
