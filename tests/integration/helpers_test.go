package integration

import (
	"bytes"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type containerInfo struct {
	Name     string
	DSN      string
	HostPort string
}

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func runCommand(t *testing.T, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = repoRoot(t)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, out.String())
	}
	return out.String()
}

func dockerPort(t *testing.T, container string, containerPort int) string {
	t.Helper()
	output := runCommand(t, "docker", "port", container, fmt.Sprintf("%d/tcp", containerPort))
	// output may have multiple lines (IPv4 and IPv6); take the first
	firstLine := strings.TrimSpace(strings.SplitN(output, "\n", 2)[0])
	// format: 0.0.0.0:PORT
	parts := strings.Split(firstLine, ":")
	if len(parts) != 2 {
		t.Fatalf("unexpected docker port output: %q", output)
	}
	return parts[1]
}

func waitForDB(t *testing.T, timeout time.Duration, open func() (*sql.DB, error)) *sql.DB {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		db, err := open()
		if err == nil {
			if pingErr := db.Ping(); pingErr == nil {
				return db
			}
			_ = db.Close()
		}
		time.Sleep(1 * time.Second)
	}
	t.Fatal("database did not become ready before timeout")
	return nil
}

func startMySQLContainer(t *testing.T) containerInfo {
	t.Helper()
	name := fmt.Sprintf("sql-agent-mysql-%d", time.Now().UnixNano())
	t.Cleanup(func() {
		exec.Command("docker", "rm", "-f", name).Run()
	})

	runCommand(t, "docker", "run", "-d", "--name", name,
		"-e", "MYSQL_ROOT_PASSWORD=rootpass",
		"-e", "MYSQL_DATABASE=app",
		"-p", "3306",
		"mysql:8")

	port := dockerPort(t, name, 3306)
	dsn := fmt.Sprintf("root:rootpass@tcp(127.0.0.1:%s)/app?parseTime=true", port)
	db := waitForDB(t, 90*time.Second, func() (*sql.DB, error) {
		return sql.Open("mysql", dsn)
	})
	t.Cleanup(func() { _ = db.Close() })
	seedMySQL(t, db)

	return containerInfo{Name: name, DSN: dsn, HostPort: port}
}

func startPostgresContainer(t *testing.T) containerInfo {
	t.Helper()
	name := fmt.Sprintf("sql-agent-postgres-%d", time.Now().UnixNano())
	t.Cleanup(func() {
		exec.Command("docker", "rm", "-f", name).Run()
	})

	runCommand(t, "docker", "run", "-d", "--name", name,
		"-e", "POSTGRES_PASSWORD=postgres",
		"-e", "POSTGRES_DB=app",
		"-p", "5432",
		"postgres:18")

	port := dockerPort(t, name, 5432)
	dsn := fmt.Sprintf("postgres://postgres:postgres@127.0.0.1:%s/app?sslmode=disable", port)
	db := waitForDB(t, 90*time.Second, func() (*sql.DB, error) {
		return sql.Open("pgx", dsn)
	})
	t.Cleanup(func() { _ = db.Close() })
	seedPostgres(t, db)

	return containerInfo{Name: name, DSN: dsn, HostPort: port}
}

func seedMySQL(t *testing.T, db *sql.DB) {
	t.Helper()
	statements := []string{
		"DROP TABLE IF EXISTS users",
		"CREATE TABLE users (id BIGINT PRIMARY KEY AUTO_INCREMENT, email VARCHAR(255) NOT NULL, active BOOLEAN NOT NULL DEFAULT TRUE)",
		"INSERT INTO users (email, active) VALUES ('alice@example.com', TRUE), ('bob@example.com', FALSE)",
	}
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("mysql seed failed: %v", err)
		}
	}
}

func seedPostgres(t *testing.T, db *sql.DB) {
	t.Helper()
	statements := []string{
		"DROP TABLE IF EXISTS users",
		"CREATE TABLE users (id BIGSERIAL PRIMARY KEY, email TEXT NOT NULL, active BOOLEAN NOT NULL DEFAULT TRUE)",
		"INSERT INTO users (email, active) VALUES ('alice@example.com', TRUE), ('bob@example.com', FALSE)",
	}
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("postgres seed failed: %v", err)
		}
	}
}

func createFakeCredentialHelper(t *testing.T, username, password string) string {
	t.Helper()
	tmpDir := t.TempDir()
	helperPath := filepath.Join(tmpDir, "credential-helper")
	script := fmt.Sprintf(`#!/bin/sh
read -r req
echo '{"ok":true,"credentials":{"username":"%s","password":"%s"}}'
`, username, password)
	if err := os.WriteFile(helperPath, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	return helperPath
}

func buildCLI(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(repoRoot(t), "bin", "sql-agent")
	runCommand(t, "bash", "scripts/build.sh")
	return binary
}

func runCLI(t *testing.T, env []string, args ...string) string {
	t.Helper()
	binary := buildCLI(t)
	cmd := exec.Command(binary, args...)
	cmd.Env = append(os.Environ(), env...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		// blocked dangerous writes are expected in tests and still produce output
		if !strings.Contains(out.String(), "destructive_query_requires_confirmation") {
			t.Fatalf("sql-agent %v failed: %v\n%s", args, err, out.String())
		}
	}
	return out.String()
}
