# Read-Only Mode Design Spec

## Summary

Add connection-level read-only mode to `sql-agent-cli`. When a connection is marked read-only, all write/DDL statements (`INSERT`, `UPDATE`, `DELETE`, `CREATE`, `ALTER`, `DROP`, `TRUNCATE`) are rejected at the CLI level before reaching the database. Read queries (`SELECT`, `WITH ... SELECT`) and schema inspection continue to work normally.

## Context And Problem

Currently, `sql-agent-cli` supports write operations with a `--confirm` gate for dangerous SQL. However, there is no way to create a connection that is permanently restricted to read-only operations. This is a gap for:

- Production databases where the human operator wants to guarantee the agent cannot mutate data.
- Shared connections where different agents have different privilege levels.
- Compliance scenarios where write access must be explicitly enabled per-connection.

## Decision Record

| Topic | Decision |
| --- | --- |
| Scope | Connection-level only (set at `connect` time) |
| Storage | `ReadOnly bool` persisted in encrypted connection record |
| Enforcement | CLI-level rejection before database execution |
| Rejected statements | `INSERT`, `UPDATE`, `DELETE`, `CREATE`, `ALTER`, `DROP`, `TRUNCATE` |
| Allowed statements | `SELECT`, `WITH ... SELECT`, schema inspection (`schema list`, `schema describe`) |
| Error contract | Structured JSON with code `read_only_connection_rejected` |
| Override | None; read-only is permanent for the connection |
| Backward compatibility | Old connections without `ReadOnly` field default to `false` (write allowed) |

## Non-Goals

- Query-level read-only override (e.g. `--write` on a read-only connection).
- Environment-variable global read-only mode.
- Database-side read-only enforcement (e.g. `SET SESSION CHARACTERISTICS AS TRANSACTION READ ONLY`).
- Automatic detection of read-only database users.

## Architecture Changes

### 1. `internal/config/store.go`

Add `ReadOnly bool` to `Connection` struct:

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

The `omitempty` ensures backward compatibility: old connections without the field deserialize as `false`.

### 2. `internal/cli/connect.go`

Add `--read-only` flag:

```go
var readOnly bool
// ...
cmd.Flags().BoolVar(&readOnly, "read-only", false, "Mark connection as read-only")
```

Pass `ReadOnly: readOnly` to `config.SaveConnection`.

### 3. `internal/safety/analyzer.go`

Extend `Analysis` with `IsReadOnly` classification:

```go
type Analysis struct {
    StatementType   string
    RequiresConfirm bool
    IsReadOnly      bool
}
```

`Analyze()` sets `IsReadOnly = true` for `SELECT` and `WITH` statements, `false` for all others.

### 4. `internal/cli/query.go`

After loading connection and analyzing SQL, enforce read-only boundary:

```go
if resolved.ReadOnly && !analysis.IsReadOnly {
    return emitJSON(..., output.Envelope{
        OK: false,
        Error: output.ErrorBody{
            Code:    "read_only_connection_rejected",
            Message: fmt.Sprintf("connection %q is read-only; write statements are not allowed", args[0]),
            Details: map[string]any{
                "connection":      args[0],
                "statement_type":  analysis.StatementType,
            },
        },
    })
}
```

This check happens **before** `resolveConnection` opens the database driver.

## Data Flow

```text
User: sql-agent connect prod --read-only --driver postgres ...
  -> config.SaveConnection({Name: "prod", ReadOnly: true, ...})

Agent: sql-agent query prod "DELETE FROM users"
  -> config.LoadConnection("prod")        // ReadOnly = true
  -> safety.Analyze("DELETE FROM users")  // IsReadOnly = false
  -> ReadOnly && !IsReadOnly              // true
  -> Return error JSON (no DB connection opened)

Agent: sql-agent query prod "SELECT * FROM users"
  -> config.LoadConnection("prod")        // ReadOnly = true
  -> safety.Analyze("SELECT ...")         // IsReadOnly = true
  -> ReadOnly && !IsReadOnly              // false
  -> Proceed to driver.QueryPaginated(...)
```

## Error Contract

When a write statement is rejected on a read-only connection:

```json
{
  "ok": false,
  "data": null,
  "warning": null,
  "error": {
    "code": "read_only_connection_rejected",
    "message": "connection \"prod\" is read-only; write statements are not allowed",
    "details": {
      "connection": "prod",
      "statement_type": "DELETE"
    }
  },
  "duration_ms": 0
}
```

Error code `read_only_connection_rejected` is stable and machine-consumable.

## CLI Surface Changes

### `sql-agent connect <name>`

New flag:

```text
--read-only    Mark connection as read-only (default: false)
```

Example:

```bash
sql-agent connect analytics --driver postgres --host db.example.com --database app --read-only
```

### `sql-agent query <name> <sql>`

No new flags. Behavior change: if `<name>` is a read-only connection, write statements are rejected.

### `sql-agent schema list <name>`

No change. Schema inspection is always allowed on read-only connections.

### `sql-agent schema describe <name> <table>`

No change. Schema inspection is always allowed on read-only connections.

### `sql-agent count <name> <target>`

No change. `COUNT` is read-only and always allowed.

## Backward Compatibility

- Existing connections stored without `read_only` field deserialize as `ReadOnly: false`.
- `sql-agent connect` without `--read-only` creates a normal read-write connection.
- No behavior change for existing connections or new connections without the flag.

## Testing Strategy

### Unit Tests

1. `safety.Analyze` correctly classifies read-only vs. write statements.
2. `query` command rejects write statements on read-only connections.
3. `query` command allows read statements on read-only connections.
4. `connect` command persists `ReadOnly` flag correctly.
5. `LoadConnection` defaults `ReadOnly` to `false` for legacy records.

### Integration Tests

1. Connect with `--read-only`, then query a write statement → rejected.
2. Connect with `--read-only`, then query a read statement → succeeds.
3. Connect without `--read-only`, then query a write statement with `--confirm` → succeeds.

## Success Criteria

- A read-only connection rejects all write/DDL statements without opening a database connection.
- Read queries and schema inspection work normally on read-only connections.
- The feature is backward compatible with existing connections.
- Error output follows the existing JSON envelope contract.
- Tests cover unit and integration scenarios.
