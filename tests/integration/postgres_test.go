package integration

import (
	"strings"
	"testing"
)

func TestPostgresE2E(t *testing.T) {
	container := startPostgresContainer(t)
	if container.Name == "" {
		t.Fatal("expected postgres container details")
	}

	home := t.TempDir()
	env := []string{"DB_AGENT_MASTER_KEY=test-master-key", "HOME=" + home}

	runCLI(t, env, "connect", "pg-test", "--driver", "postgres", "--host", "127.0.0.1", "--port", container.HostPort, "--database", "app", "--username", "postgres", "--password-env", "PG_PASSWORD")
	queryOut := runCLI(t, append(env, "PG_PASSWORD=postgres"), "query", "pg-test", "SELECT id, email FROM users ORDER BY id")
	if !strings.Contains(queryOut, "alice@example.com") {
		t.Fatalf("expected query output to include seeded postgres data, got %q", queryOut)
	}

	schemaOut := runCLI(t, append(env, "PG_PASSWORD=postgres"), "schema", "list", "pg-test")
	if !strings.Contains(schemaOut, "users") {
		t.Fatalf("expected schema list to include users table, got %q", schemaOut)
	}

	blockedOut := runCLI(t, append(env, "PG_PASSWORD=postgres"), "query", "pg-test", "DELETE FROM users")
	if !strings.Contains(blockedOut, "destructive_query_requires_confirmation") {
		t.Fatalf("expected blocked dangerous write output, got %q", blockedOut)
	}
}
