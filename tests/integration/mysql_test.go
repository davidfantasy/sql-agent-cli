package integration

import (
	"strings"
	"testing"
)

func TestMySQLE2E(t *testing.T) {
	container := startMySQLContainer(t)
	if container.Name == "" {
		t.Fatal("expected mysql container details")
	}

	home := t.TempDir()
	env := []string{"DB_AGENT_MASTER_KEY=test-master-key", "HOME=" + home}

	runCLI(t, env, "connect", "mysql-test", "--driver", "mysql", "--host", "127.0.0.1", "--port", container.HostPort, "--database", "app", "--username", "root", "--password-env", "MYSQL_PASSWORD")
	queryOut := runCLI(t, append(env, "MYSQL_PASSWORD=rootpass"), "query", "mysql-test", "SELECT id, email FROM users ORDER BY id")
	if !strings.Contains(queryOut, "alice@example.com") {
		t.Fatalf("expected query output to include seeded mysql data, got %q", queryOut)
	}

	schemaOut := runCLI(t, append(env, "MYSQL_PASSWORD=rootpass"), "schema", "list", "mysql-test")
	if !strings.Contains(schemaOut, "users") {
		t.Fatalf("expected schema list to include users table, got %q", schemaOut)
	}

	blockedOut := runCLI(t, append(env, "MYSQL_PASSWORD=rootpass"), "query", "mysql-test", "DELETE FROM users")
	if !strings.Contains(blockedOut, "destructive_query_requires_confirmation") {
		t.Fatalf("expected blocked dangerous write output, got %q", blockedOut)
	}
}

func TestMySQLE2E_WithCredentialHelper(t *testing.T) {
	container := startMySQLContainer(t)
	if container.Name == "" {
		t.Fatal("expected mysql container details")
	}

	home := t.TempDir()
	env := []string{"DB_AGENT_MASTER_KEY=test-master-key", "HOME=" + home}

	helperPath := createFakeCredentialHelper(t, "root", "rootpass")

	runCLI(t, env, "connect", "mysql-helper-test", "--driver", "mysql", "--host", "127.0.0.1", "--port", container.HostPort, "--database", "app", "--credential-helper", helperPath)
	queryOut := runCLI(t, env, "query", "mysql-helper-test", "SELECT id, email FROM users ORDER BY id")
	if !strings.Contains(queryOut, "alice@example.com") {
		t.Fatalf("expected credential-helper query output to include seeded mysql data, got %q", queryOut)
	}

	schemaOut := runCLI(t, env, "schema", "list", "mysql-helper-test")
	if !strings.Contains(schemaOut, "users") {
		t.Fatalf("expected credential-helper schema list to include users table, got %q", schemaOut)
	}
}
