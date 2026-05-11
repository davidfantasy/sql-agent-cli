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
