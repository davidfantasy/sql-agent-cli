# Read-Only Mode Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add connection-level read-only mode that rejects write/DDL statements before database execution.

**Architecture:** Extend `Connection` and `Resolved` structs with `ReadOnly bool`, extend `Analysis` with `IsReadOnly bool`, enforce at query-time after connection resolution.

**Tech Stack:** Go, Cobra, standard library JSON/encoding

---

## File Structure

| File | Action | Responsibility |
|------|--------|---------------|
| `internal/config/store.go` | Modify | Add `ReadOnly bool` to `Connection` struct |
| `internal/credentials/resolver.go` | Modify | Add `ReadOnly bool` to `Resolved` struct, pass through in `Resolve()` |
| `internal/safety/analyzer.go` | Modify | Add `IsReadOnly bool` to `Analysis`, classify statements |
| `internal/safety/analyzer_test.go` | Modify | Add tests for `IsReadOnly` classification |
| `internal/cli/connect.go` | Modify | Add `--read-only` flag |
| `internal/cli/connect_test.go` | Modify | Add test for `--read-only` persistence |
| `internal/cli/query.go` | Modify | Add read-only enforcement check |
| `internal/cli/query_test.go` | Modify | Add tests for read-only rejection and read allowance |
| `tests/integration/postgres_test.go` | Modify | Add read-only integration test |
| `tests/integration/mysql_test.go` | Modify | Add read-only integration test |

---

### Task 1: Extend `Connection` struct with `ReadOnly`

**Files:**
- Modify: `internal/config/store.go`

- [ ] **Step 1: Add `ReadOnly` field to `Connection` struct**

```go
type Connection struct {
    Name             string `json:"name"`
    Driver           string `json:"driver"`
    Host             string `json:"host,omitempty"`
    Port             int    `json:"port,omitempty"`
    Database         string `json:"database"`
    Username         string `json:"username,omitempty"`
    PasswordEnv      string `json:"password_env,omitempty"`
    CredentialHelper string `json:"credential_helper,omitempty"`
    ReadOnly         bool   `json:"read_only,omitempty"`
}
```

- [ ] **Step 2: Run existing config tests to ensure backward compatibility**

Run: `go test ./internal/config/... -v`
Expected: PASS (old connections without `read_only` deserialize as `false`)

- [ ] **Step 3: Commit**

```bash
git add internal/config/store.go
git commit -m "feat(config): add ReadOnly field to Connection struct"
```

---

### Task 2: Extend `Resolved` struct and `Resolve()` function

**Files:**
- Modify: `internal/credentials/resolver.go`

- [ ] **Step 1: Add `ReadOnly` field to `Resolved` struct**

```go
type Resolved struct {
    Driver   string
    Host     string
    Port     int
    Database string
    Username string
    Password string
    ReadOnly bool
}
```

- [ ] **Step 2: Pass `ReadOnly` through in `Resolve()`**

In `Resolve()`, after initializing `resolved`, add:

```go
resolved.ReadOnly = conn.ReadOnly
```

Full `Resolve()` should look like:

```go
func Resolve(conn config.Connection, helper func(config.Connection) (HelperCredentials, error)) (Resolved, error) {
    resolved := Resolved{
        Driver:   conn.Driver,
        Host:     conn.Host,
        Port:     conn.Port,
        Database: conn.Database,
        Username: conn.Username,
        ReadOnly: conn.ReadOnly,
    }
    // ... rest of function unchanged
}
```

- [ ] **Step 3: Run credential tests**

Run: `go test ./internal/credentials/... -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/credentials/resolver.go
git commit -m "feat(credentials): pass ReadOnly through Resolve"
```

---

### Task 3: Extend safety analyzer with `IsReadOnly` classification

**Files:**
- Modify: `internal/safety/analyzer.go`
- Modify: `internal/safety/analyzer_test.go`

- [ ] **Step 1: Add `IsReadOnly` to `Analysis` struct**

```go
type Analysis struct {
    StatementType   string
    RequiresConfirm bool
    IsReadOnly      bool
}
```

- [ ] **Step 2: Set `IsReadOnly` in `Analyze()`**

After determining `statementType`, add classification logic:

```go
analysis := Analysis{StatementType: statementType}
if statementType == "SELECT" {
    analysis.IsReadOnly = true
}
```

The full function should classify as follows:
- `SELECT`, `WITH` → `IsReadOnly: true`
- `INSERT`, `UPDATE`, `DELETE`, `CREATE`, `ALTER`, `DROP`, `TRUNCATE` → `IsReadOnly: false`

- [ ] **Step 3: Add tests for `IsReadOnly` classification**

Add to `internal/safety/analyzer_test.go`:

```go
func TestAnalyze_SelectIsReadOnly(t *testing.T) {
    result, err := Analyze("SELECT id FROM users")
    if err != nil {
        t.Fatal(err)
    }
    if !result.IsReadOnly {
        t.Fatal("expected SELECT to be read-only")
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
```

- [ ] **Step 4: Run safety tests**

Run: `go test ./internal/safety/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/safety/analyzer.go internal/safety/analyzer_test.go
git commit -m "feat(safety): classify statements as read-only or write"
```

---

### Task 4: Add `--read-only` flag to `connect` command

**Files:**
- Modify: `internal/cli/connect.go`
- Modify: `internal/cli/connect_test.go`

- [ ] **Step 1: Add `readOnly` variable and flag**

In `newConnectCommand()`, add:

```go
var readOnly bool
```

Pass `ReadOnly: readOnly` to `config.SaveConnection`:

```go
return config.SaveConnection(config.Connection{
    Name:             args[0],
    Driver:           driver,
    Host:             host,
    Port:             port,
    Database:         database,
    Username:         username,
    PasswordEnv:      passwordEnv,
    CredentialHelper: credentialHelper,
    ReadOnly:         readOnly,
})
```

Add flag definition:

```go
cmd.Flags().BoolVar(&readOnly, "read-only", false, "Mark connection as read-only")
```

- [ ] **Step 2: Add test for `--read-only` persistence**

Add to `internal/cli/connect_test.go`:

```go
func TestConnectCommand_SavesReadOnlyConnection(t *testing.T) {
    t.Setenv(config.MasterKeyEnv, "0123456789abcdef0123456789abcdef")
    t.Setenv("HOME", t.TempDir())

    cmd := newConnectCommand()
    cmd.SetArgs([]string{"readonly-test", "--driver", "postgres", "--database", "app", "--read-only"})

    if err := cmd.Execute(); err != nil {
        t.Fatal(err)
    }

    conn, err := config.LoadConnection("readonly-test")
    if err != nil {
        t.Fatal(err)
    }
    if !conn.ReadOnly {
        t.Fatalf("expected connection to be read-only, got ReadOnly=%v", conn.ReadOnly)
    }
}
```

- [ ] **Step 3: Run connect tests**

Run: `go test ./internal/cli/... -run TestConnectCommand -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/cli/connect.go internal/cli/connect_test.go
git commit -m "feat(connect): add --read-only flag"
```

---

### Task 5: Add read-only enforcement to `query` command

**Files:**
- Modify: `internal/cli/query.go`

- [ ] **Step 1: Add read-only check after connection resolution**

After the existing safety analysis and confirm check, after `resolveConnection(args[0])`, add:

```go
if resolved.ReadOnly && !analysis.IsReadOnly {
    marshalErr := emitJSON(cmd.OutOrStdout(), output.Envelope{
        OK:      false,
        Data:    nil,
        Warning: nil,
        Error: output.ErrorBody{
            Code:    "read_only_connection_rejected",
            Message: fmt.Sprintf("connection %q is read-only; write statements are not allowed", args[0]),
            Details: map[string]any{
                "connection":     args[0],
                "statement_type": analysis.StatementType,
            },
        },
        DurationMS: 0,
    })
    if marshalErr != nil {
        return marshalErr
    }
    return errors.New("read-only connection rejected write statement")
}
```

This check should be placed **after** the existing `RequiresConfirm` check and **after** `resolveConnection()`, but **before** `openDriver()`. This ensures:
1. Dangerous write confirmation still works on read-write connections.
2. We have access to `resolved.ReadOnly`.
3. We reject before opening a database connection.

- [ ] **Step 2: Add import for `fmt` if not already present**

Ensure `fmt` is in the import block:

```go
import (
    "errors"
    "fmt"
    "regexp"
    // ... other imports
)
```

- [ ] **Step 3: Commit**

```bash
git add internal/cli/query.go
git commit -m "feat(query): reject write statements on read-only connections"
```

---

### Task 6: Add unit tests for read-only query enforcement

**Files:**
- Modify: `internal/cli/query_test.go`

- [ ] **Step 1: Add test for read-only connection rejecting write**

Add to `internal/cli/query_test.go`:

```go
func TestQueryCommand_RejectsWriteOnReadOnlyConnection(t *testing.T) {
    t.Setenv(config.MasterKeyEnv, "0123456789abcdef0123456789abcdef")
    t.Setenv("HOME", t.TempDir())

    // Create a read-only connection
    connectCmd := newConnectCommand()
    connectCmd.SetArgs([]string{"readonly-conn", "--driver", "postgres", "--database", "app", "--read-only"})
    if err := connectCmd.Execute(); err != nil {
        t.Fatal(err)
    }

    cmd := newQueryCommand()
    buf := &bytes.Buffer{}
    cmd.SetOut(buf)
    cmd.SetErr(buf)
    cmd.SetArgs([]string{"readonly-conn", "DELETE FROM users"})

    err := cmd.Execute()
    if err == nil {
        t.Fatal("expected write on read-only connection to fail")
    }
    if !strings.Contains(buf.String(), "read_only_connection_rejected") {
        t.Fatalf("expected read_only_connection_rejected error, got %q", buf.String())
    }
    if !strings.Contains(buf.String(), "readonly-conn") {
        t.Fatalf("expected error to mention connection name, got %q", buf.String())
    }
}
```

- [ ] **Step 2: Add test for read-only connection allowing read**

Since we need a real database for SELECT to succeed, this test will verify that the read-only check does not block SELECT statements (the statement will fail later due to no real database, but not at the read-only check):

```go
func TestQueryCommand_ReadOnlyConnectionAllowsRead(t *testing.T) {
    t.Setenv(config.MasterKeyEnv, "0123456789abcdef0123456789abcdef")
    t.Setenv("HOME", t.TempDir())

    // Create a read-only connection
    connectCmd := newConnectCommand()
    connectCmd.SetArgs([]string{"readonly-conn", "--driver", "postgres", "--database", "app", "--read-only"})
    if err := connectCmd.Execute(); err != nil {
        t.Fatal(err)
    }

    cmd := newQueryCommand()
    buf := &bytes.Buffer{}
    cmd.SetOut(buf)
    cmd.SetErr(buf)
    cmd.SetArgs([]string{"readonly-conn", "SELECT 1"})

    err := cmd.Execute()
    // Should NOT fail with read_only_connection_rejected
    // It will fail because there's no real database, but that's expected for unit tests
    if strings.Contains(buf.String(), "read_only_connection_rejected") {
        t.Fatalf("did not expect read_only_connection_rejected for SELECT, got %q", buf.String())
    }
}
```

- [ ] **Step 3: Run query tests**

Run: `go test ./internal/cli/... -run TestQueryCommand -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/cli/query_test.go
git commit -m "test(query): add read-only connection tests"
```

---

### Task 7: Add integration tests for read-only mode

**Files:**
- Modify: `tests/integration/postgres_test.go`
- Modify: `tests/integration/mysql_test.go`

- [ ] **Step 1: Add read-only integration test for PostgreSQL**

Add to `tests/integration/postgres_test.go` at the end of `TestPostgresE2E`:

```go
    // Test read-only connection rejects write
    runCLI(t, env, "connect", "pg-readonly", "--driver", "postgres", "--host", "127.0.0.1", "--port", container.HostPort, "--database", "app", "--username", "postgres", "--password-env", "PG_PASSWORD", "--read-only")
    
    readOnlyBlockedOut := runCLI(t, append(env, "PG_PASSWORD=postgres"), "query", "pg-readonly", "DELETE FROM users")
    if !strings.Contains(readOnlyBlockedOut, "read_only_connection_rejected") {
        t.Fatalf("expected read-only connection to block write, got %q", readOnlyBlockedOut)
    }

    // Test read-only connection allows read
    readOnlyQueryOut := runCLI(t, append(env, "PG_PASSWORD=postgres"), "query", "pg-readonly", "SELECT id, email FROM users ORDER BY id")
    if !strings.Contains(readOnlyQueryOut, "alice@example.com") {
        t.Fatalf("expected read-only connection to allow read, got %q", readOnlyQueryOut)
    }
```

- [ ] **Step 2: Add read-only integration test for MySQL**

Add to `tests/integration/mysql_test.go` at the end of `TestMySQLE2E`:

```go
    // Test read-only connection rejects write
    runCLI(t, env, "connect", "mysql-readonly", "--driver", "mysql", "--host", "127.0.0.1", "--port", container.HostPort, "--database", "app", "--username", "root", "--password-env", "MYSQL_PASSWORD", "--read-only")
    
    readOnlyBlockedOut := runCLI(t, append(env, "MYSQL_PASSWORD=rootpass"), "query", "mysql-readonly", "DELETE FROM users")
    if !strings.Contains(readOnlyBlockedOut, "read_only_connection_rejected") {
        t.Fatalf("expected read-only connection to block write, got %q", readOnlyBlockedOut)
    }

    // Test read-only connection allows read
    readOnlyQueryOut := runCLI(t, append(env, "MYSQL_PASSWORD=rootpass"), "query", "mysql-readonly", "SELECT id, email FROM users ORDER BY id")
    if !strings.Contains(readOnlyQueryOut, "alice@example.com") {
        t.Fatalf("expected read-only connection to allow read, got %q", readOnlyQueryOut)
    }
```

- [ ] **Step 3: Run integration tests (requires Docker)**

Run: `bash scripts/test.sh`
Expected: All tests PASS including new read-only integration tests

- [ ] **Step 4: Commit**

```bash
git add tests/integration/postgres_test.go tests/integration/mysql_test.go
git commit -m "test(integration): add read-only mode e2e tests"
```

---

### Task 8: Update documentation

**Files:**
- Modify: `docs/superpowers/specs/2026-05-11-sql-agent-cli-design.md` (optional: add read-only to decision table)
- Modify: `README.md` (add `--read-only` to connect examples)
- Modify: `skill/sql-agent/SKILL.md` (teach agents about read-only connections)

- [ ] **Step 1: Update spec decision table**

In `docs/superpowers/specs/2026-05-11-sql-agent-cli-design.md`, add to the Decision Record table:

```markdown
| Read-only mode | Connection-level `--read-only` flag on `connect` |
```

- [ ] **Step 2: Update README with `--read-only` example**

Add to README in the `connect` section:

```markdown
Create a read-only connection:

```bash
sql-agent connect analytics --driver postgres --host db.example.com --database app --read-only
```
```

- [ ] **Step 3: Update SKILL.md**

Add to `skill/sql-agent/SKILL.md` (in the usage section):

```markdown
- Use `--read-only` when connecting to production databases to prevent accidental writes.
- Read-only connections reject all write/DDL statements with a clear error code.
```

- [ ] **Step 4: Commit**

```bash
git add docs/superpowers/specs/2026-05-11-sql-agent-cli-design.md README.md skill/sql-agent/SKILL.md
git commit -m "docs: document read-only mode"
```

---

### Task 9: Final validation

- [ ] **Step 1: Run all quality checks**

```bash
bash scripts/format.sh
bash scripts/build.sh
bash scripts/test.sh
```

Expected: All pass without errors.

- [ ] **Step 2: Commit any formatting changes**

```bash
git diff --quiet || git commit -am "style: format read-only mode changes"
```

---

## Spec Coverage Check

| Spec Requirement | Implementing Task |
|-----------------|-------------------|
| `ReadOnly bool` in `Connection` struct | Task 1 |
| `ReadOnly bool` in `Resolved` struct | Task 2 |
| `IsReadOnly bool` in `Analysis` struct | Task 3 |
| `--read-only` flag on `connect` | Task 4 |
| Reject write statements on read-only connections | Task 5 |
| Error code `read_only_connection_rejected` | Task 5 |
| Unit tests for safety classification | Task 3 |
| Unit tests for connect persistence | Task 4 |
| Unit tests for query enforcement | Task 6 |
| Integration tests for PostgreSQL | Task 7 |
| Integration tests for MySQL | Task 7 |
| Backward compatibility | Task 1 (omitempty) |
| Documentation updates | Task 8 |

## Placeholder Scan

- No "TBD", "TODO", or "implement later" found.
- All test code is complete with actual assertions.
- All file paths are exact.
- All commands have expected outputs.
- Type names consistent across tasks (`ReadOnly`, `IsReadOnly`, `read_only_connection_rejected`).

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-05-11-read-only-mode.md`. Two execution options:

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

Which approach?
