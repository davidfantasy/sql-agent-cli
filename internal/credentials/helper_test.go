package credentials

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFakeHelper(t *testing.T, script string) string {
	t.Helper()
	tmpDir := t.TempDir()
	helperPath := filepath.Join(tmpDir, "helper")
	if err := os.WriteFile(helperPath, []byte("#!/bin/sh\n"+script), 0755); err != nil {
		t.Fatal(err)
	}
	return helperPath
}

func TestRunCredentialHelper_ExecutesCommandAndParsesResponse(t *testing.T) {
	helper := writeFakeHelper(t, `
read -r req
echo '{"ok":true,"credentials":{"username":"agent","password":"secret","host":"prod.internal","port":5432}}'
`)

	creds, err := RunCredentialHelper("test", "postgres", helper)
	if err != nil {
		t.Fatal(err)
	}

	if creds.Username != "agent" {
		t.Fatalf("expected agent, got %q", creds.Username)
	}
	if creds.Password != "secret" {
		t.Fatalf("expected secret, got %q", creds.Password)
	}
	if creds.Host != "prod.internal" {
		t.Fatalf("expected prod.internal, got %q", creds.Host)
	}
	if creds.Port != 5432 {
		t.Fatalf("expected port 5432, got %d", creds.Port)
	}
}

func TestRunCredentialHelper_PropagatesCommandFailure(t *testing.T) {
	helper := writeFakeHelper(t, `
echo '{"ok":false}' >&2
exit 1
`)

	_, err := RunCredentialHelper("test", "postgres", helper)
	if err == nil {
		t.Fatal("expected helper failure to propagate")
	}
}

func TestRunCredentialHelper_RejectsInvalidJSON(t *testing.T) {
	helper := writeFakeHelper(t, `
echo 'not json'
`)

	_, err := RunCredentialHelper("test", "postgres", helper)
	if err == nil {
		t.Fatal("expected invalid JSON to fail")
	}
}

func TestRunCredentialHelper_StderrIsDiagnosticOnly(t *testing.T) {
	helper := writeFakeHelper(t, `
echo 'some diagnostic info' >&2
echo '{"ok":true,"credentials":{"username":"agent"}}'
`)

	creds, err := RunCredentialHelper("test", "postgres", helper)
	if err != nil {
		t.Fatal(err)
	}
	if creds.Username != "agent" {
		t.Fatalf("expected agent, got %q", creds.Username)
	}
}

func TestRunCredentialHelper_SendsRequestJSON(t *testing.T) {
	helper := writeFakeHelper(t, `
read -r req
echo "$req" | grep '"version":1' > /dev/null || { echo '{"ok":false}' >&2; exit 1; }
echo '{"ok":true,"credentials":{"username":"agent"}}'
`)

	creds, err := RunCredentialHelper("prod", "mysql", helper)
	if err != nil {
		t.Fatal(err)
	}
	if creds.Username != "agent" {
		t.Fatalf("expected agent, got %q", creds.Username)
	}
}
