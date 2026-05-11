# sql-agent-cli Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go-based MySQL/PostgreSQL CLI for AI agents with encrypted named connections, deterministic JSON output, dangerous-write confirmation gates, portable skill packaging, and Docker-backed local integration tests.

**Architecture:** The CLI remains a thin command layer over small internal packages for config, credential resolution, safety analysis, database access, and JSON formatting. v1 removes SQLite to keep local compilation simple and focuses on MySQL/PostgreSQL through `database/sql` with Go-native drivers and Docker-based end-to-end verification.

**Tech Stack:** Go, Cobra, `database/sql`, `github.com/go-sql-driver/mysql`, `github.com/jackc/pgx/v5/stdlib`, Go testing, Docker CLI, local shell scripts.

---

## File Map

### Modify

- `go.mod`: remove SQLite dependency, add MySQL/PostgreSQL drivers.
- `internal/cli/connect.go`: restrict drivers to mysql/postgres and keep encrypted persistence wiring.
- `internal/cli/disconnect.go`: keep delete behavior.
- `internal/cli/query.go`: load connection, resolve credentials, analyze SQL, open driver, emit JSON.
- `internal/cli/schema.go`: execute schema list/describe via selected driver.
- `internal/cli/count.go`: execute count through selected driver.
- `internal/db/driver.go`: define backend abstraction and shared table-description model.
- `internal/db/mysql.go`: implement MySQL driver.
- `internal/db/postgres.go`: implement PostgreSQL driver.
- `scripts/test.sh`: run unit tests plus Docker-backed integration tests.
- `README.md`: document supported databases and Docker test loop.
- `AGENTS.md`: keep architecture/docs aligned with MySQL/PostgreSQL-only scope.
- `docs/development.md`: document Docker dependency.
- `docs/superpowers/specs/2026-05-11-sql-agent-cli-design.md`: keep spec aligned with implementation scope.
- `skill/sql-agent/SKILL.md`: keep skill aligned with MySQL/PostgreSQL scope and human confirmation rule.
- `skill/sql-agent/evals/evals.json`: reflect current support and safety behavior.

### Create

- `internal/db/mysql_test.go`: TDD coverage for MySQL driver behavior that does not require live server mocking.
- `internal/db/postgres_test.go`: TDD coverage for PostgreSQL driver behavior that does not require live server mocking.
- `tests/integration/helpers_test.go`: shared helpers for Docker lifecycle and CLI execution.
- `tests/integration/mysql_test.go`: MySQL end-to-end integration tests.
- `tests/integration/postgres_test.go`: PostgreSQL end-to-end integration tests.

### Delete

- `internal/db/sqlite.go`
- `internal/db/sqlite_test.go`
- `tests/integration/sqlite_test.go`

## Task 1: Remove SQLite from the v1 contract

**Files:**
- Modify: `AGENTS.md`
- Modify: `README.md`
- Modify: `docs/development.md`
- Modify: `docs/superpowers/specs/2026-05-11-sql-agent-cli-design.md`
- Modify: `skill/sql-agent/SKILL.md`
- Modify: `skill/sql-agent/evals/evals.json`
- Delete: `internal/db/sqlite.go`
- Delete: `internal/db/sqlite_test.go`
- Delete: `tests/integration/sqlite_test.go`

- [ ] **Step 1: Remove SQLite-specific files**

Delete the SQLite files listed above.

- [ ] **Step 2: Rewrite docs to MySQL/PostgreSQL-only language**

Keep these exact statements aligned across docs:

```text
v1 supports MySQL and PostgreSQL only.
SQLite is deferred to keep local build requirements small and predictable.
Dangerous writes must be shown to the human before retrying with --confirm.
```

- [ ] **Step 3: Verify SQLite references are gone from shipped files**

Run: `rg -n "SQLite|sqlite3|modernc.org/sqlite|NewSQLiteDriver|sqlite" AGENTS.md README.md docs skill internal tests`
Expected: no matches in shipped v1 files except historical notes you intentionally keep out of implementation paths.

## Task 2: Rework dependencies for MySQL and PostgreSQL

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: Write the failing dependency-level build check**

Run: `go test ./internal/db -run TestMySQLDriver_ListSchemaQuery -v`
Expected: FAIL because MySQL driver tests and constructors do not exist yet.

- [ ] **Step 2: Add database driver dependencies**

`go.mod` should include:

```go
require (
	github.com/go-sql-driver/mysql v1.8.1
	github.com/jackc/pgx/v5 v5.7.4
	github.com/spf13/cobra v1.9.1
)
```

- [ ] **Step 3: Tidy modules**

Run: `go mod tidy`
Expected: success without SQLite dependency.

## Task 3: Define the shared DB abstraction for MySQL/PostgreSQL

**Files:**
- Modify: `internal/db/driver.go`

- [ ] **Step 1: Write the failing driver contract tests**

Add tests that expect a table-description struct and query/exec behavior helpers to exist.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/db -run 'TestMySQLDriver|TestPostgresDriver' -v`
Expected: FAIL because the richer contract is not implemented.

- [ ] **Step 3: Implement the shared types**

Add a focused contract like:

```go
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
```

- [ ] **Step 4: Re-run the tests**

Run: `go test ./internal/db -run 'TestMySQLDriver|TestPostgresDriver' -v`
Expected: still FAIL, but now for missing backend implementation instead of missing types.

## Task 4: Implement the MySQL driver with TDD

**Files:**
- Modify: `internal/db/mysql.go`
- Create: `internal/db/mysql_test.go`

- [ ] **Step 1: Write the failing MySQL unit tests**

Add tests for SQL text returned by schema helpers and constructor validation behavior. Keep them independent of a live database.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/db -run TestMySQLDriver -v`
Expected: FAIL because `NewMySQLDriver` and MySQL methods are undefined.

- [ ] **Step 3: Implement minimal MySQL driver**

Requirements:
- `NewMySQLDriver(dsn string) (Driver, error)`
- `Query`, `Exec`, `Close`
- `ListSchema` using `information_schema.tables WHERE table_schema = DATABASE()`
- `DescribeTable` using `information_schema.columns` plus primary-key metadata

- [ ] **Step 4: Re-run MySQL unit tests**

Run: `go test ./internal/db -run TestMySQLDriver -v`
Expected: PASS.

## Task 5: Implement the PostgreSQL driver with TDD

**Files:**
- Modify: `internal/db/postgres.go`
- Create: `internal/db/postgres_test.go`

- [ ] **Step 1: Write the failing PostgreSQL unit tests**

Add tests for constructor existence and schema SQL behavior.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/db -run TestPostgresDriver -v`
Expected: FAIL because `NewPostgresDriver` and PostgreSQL methods are undefined.

- [ ] **Step 3: Implement minimal PostgreSQL driver**

Requirements:
- `NewPostgresDriver(dsn string) (Driver, error)` using `pgx` stdlib
- `Query`, `Exec`, `Close`
- `ListSchema` using `information_schema.tables WHERE table_schema = 'public'`
- `DescribeTable` using `information_schema.columns` and key metadata

- [ ] **Step 4: Re-run PostgreSQL unit tests**

Run: `go test ./internal/db -run TestPostgresDriver -v`
Expected: PASS.

## Task 6: Wire real query execution in the CLI with TDD

**Files:**
- Modify: `internal/cli/query.go`
- Modify: `internal/cli/connect.go`
- Modify: `internal/credentials/resolver.go`
- Create or modify: `internal/cli/query_test.go`

- [ ] **Step 1: Write failing tests for unsupported driver and blocked dangerous queries**

Add tests for:
- unsupported driver rejected at `connect`
- dangerous query blocked without `--confirm`
- confirmation error JSON includes `statement_type` and human-review instructions

- [ ] **Step 2: Run tests to verify they fail for the new cases**

Run: `go test ./internal/cli -run 'TestConnectCommand|TestQueryCommand' -v`
Expected: FAIL on the new assertions.

- [ ] **Step 3: Implement the minimal execution flow**

Flow:

```text
load connection -> validate driver -> resolve credentials -> analyze SQL ->
if confirm required and flag absent, emit structured error JSON ->
open mysql/postgres driver -> query or exec -> format result -> emit JSON
```

- [ ] **Step 4: Re-run the CLI tests**

Run: `go test ./internal/cli -run 'TestConnectCommand|TestQueryCommand' -v`
Expected: PASS.

## Task 7: Wire schema and count commands with TDD

**Files:**
- Modify: `internal/cli/schema.go`
- Modify: `internal/cli/count.go`
- Modify: `internal/cli/schema_test.go`

- [ ] **Step 1: Write failing tests for schema/count command behavior**

Cover:
- `schema list` and `schema describe` arg validation
- `count` exact arg validation
- JSON emission path delegates to the selected backend

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/cli -run 'TestSchemaCommand|TestCountCommand' -v`
Expected: FAIL because runtime behavior is not implemented.

- [ ] **Step 3: Implement minimal command behavior**

Requirements:
- `schema list <name>` emits `{"tables": [...]}` in envelope data
- `schema describe <name> <table>` emits `TableDescription`
- `count <name> <target>` emits count rows through JSON envelope
- If target is a SQL query, wrap as `SELECT COUNT(*) AS count FROM (<query>) AS count_target`

- [ ] **Step 4: Re-run the CLI tests**

Run: `go test ./internal/cli -run 'TestSchemaCommand|TestCountCommand' -v`
Expected: PASS.

## Task 8: Add Docker-backed end-to-end tests for MySQL and PostgreSQL

**Files:**
- Create: `tests/integration/helpers_test.go`
- Modify: `tests/integration/mysql_test.go`
- Modify: `tests/integration/postgres_test.go`
- Modify: `scripts/test.sh`

- [ ] **Step 1: Write the failing integration tests**

MySQL test should:
- start or reuse a Docker container
- create a test table and seed rows
- use the built CLI or package functions to run `schema list`, `schema describe`, `query`, and blocked `DELETE`

PostgreSQL test should do the same.

- [ ] **Step 2: Run integration tests to verify they fail**

Run: `go test ./tests/integration -run 'TestMySQLE2E|TestPostgresE2E' -v`
Expected: FAIL because the helpers and real backend wiring do not exist yet.

- [ ] **Step 3: Implement shared Docker helpers**

Support:
- `docker run` with deterministic container names
- readiness polling with bounded timeout
- cleanup via `docker rm -f`
- DSN construction for MySQL/PostgreSQL

- [ ] **Step 4: Update `scripts/test.sh`**

Behavior:
- export default `DB_AGENT_MASTER_KEY`
- run unit tests
- if Docker is available, run integration tests
- if Docker is unavailable, fail clearly because Docker is a required local dependency for v1 verification

- [ ] **Step 5: Re-run integration tests**

Run: `go test ./tests/integration -run 'TestMySQLE2E|TestPostgresE2E' -v`
Expected: PASS with local containers.

## Task 9: Final verification

**Files:**
- Verify repository state only

- [ ] **Step 1: Format the repository**

Run: `bash scripts/format.sh`
Expected: success.

- [ ] **Step 2: Run the full test script**

Run: `bash scripts/test.sh`
Expected: unit + Docker-backed integration tests pass.

- [ ] **Step 3: Build the binary**

Run: `bash scripts/build.sh`
Expected: success.

- [ ] **Step 4: Build the skill package**

Run: `bash scripts/build-skill.sh`
Expected: `dist/sql-agent-skill/SKILL.md` exists.

- [ ] **Step 5: Placeholder scan**

Run: `rg -n "T[O]DO|TB[D]" AGENTS.md README.md docs skill internal tests scripts`
Expected: no placeholder hits in shipped files.

## Self-Review Notes

- Spec coverage: driver scope, dangerous-write gating, Docker-backed testing, and portable skill guidance are all represented.
- Placeholder scan: all tasks contain concrete files, commands, and verification steps.
- Type consistency: the plan centers on the existing `config`, `credentials`, `safety`, `output`, and `cli` packages, with DB scope narrowed to MySQL/PostgreSQL.
