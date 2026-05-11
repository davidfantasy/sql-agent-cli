# Backend Architecture

`sql-agent-cli` keeps the backend deliberately small. The CLI layer in `internal/cli` parses arguments and emits JSON, while focused internal packages own storage, credential resolution, SQL safety checks, database access, and output formatting.

## Package Responsibilities

- `internal/cli`: Cobra commands for `connect`, `disconnect`, `query`, `schema`, and `count`. This layer validates arguments, resolves named connections, and emits JSON envelopes.
- `internal/config`: Encrypted local storage for named connections plus master-key handling. Connection records live under `~/.sql-agent/connections/*.enc`.
- `internal/credentials`: Merges stored connection data with password environment variables or credential-helper responses.
- `internal/safety`: Enforces the single-statement boundary, classifies statement types, and marks destructive statements or read-only queries.
- `internal/db`: Small `database/sql` driver abstraction for MySQL and PostgreSQL, including pagination injection and schema inspection helpers.
- `internal/output`: Stable JSON envelope and row formatter used by all commands.

## Command Flow

Most commands follow the same path:

1. Load the encrypted named connection.
2. Resolve credentials from env or credential helper when needed.
3. Open the MySQL or PostgreSQL driver.
4. Run the requested read, write, schema, or count operation.
5. Normalize the result into the shared JSON envelope.

`query` adds one extra step before execution: `internal/safety` analyzes the SQL so the CLI can block multi-statement input, require `--confirm` for dangerous writes, and reject writes on read-only connections.

## Driver Boundaries

The driver layer is intentionally narrow:

- `Query` for read queries.
- `QueryPaginated` for paginated reads using `LIMIT page_size + 1`.
- `Exec` for writes and DDL after safety checks pass.
- `ListSchema` and `DescribeTable` for schema-first exploration.

The CLI does not expose backend-specific scripting, stored procedures, or multi-result workflows.

## Output Contracts

All commands return JSON through `internal/output.Envelope`:

- `ok` indicates success or failure.
- `data` contains command-specific payloads.
- `error.code` is stable and machine-readable.
- `duration_ms` is reserved in the envelope even when a command currently leaves it at zero.

Query results use the token-efficient column-array shape documented in the design spec: shared `columns`, row arrays, pagination metadata, and truncation counters.

## Safety Boundaries

- One SQL statement per invocation.
- Dangerous writes require `--confirm` after a human reviews the warning.
- Read-only connections reject write and DDL statements before the database sees them.
- Credential helpers are the only path that keeps secrets out of model context.

These constraints are product behavior, not optional conveniences. New features should preserve them.
