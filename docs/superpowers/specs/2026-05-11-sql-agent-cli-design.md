# sql-agent-cli Design Spec

## Summary

`sql-agent-cli` is a Unix-style CLI for AI agents to interact with SQL databases with predictable, token-efficient, and safety-aware behavior. v1 targets MySQL and PostgreSQL, exposes a small set of subcommands, returns JSON only, and is designed to be called from shell-based agents rather than through MCP.

This project has two deliverables:

1. A Go CLI that brokers database access for AI agents.
2. A portable skill package containing `SKILL.md` that teaches compatible agents how to use the CLI effectively.

## Context And Problem

Existing database integrations for AI agents are split between MCP servers, database-specific tools, and general SQL assistants. The common gaps for this project are:

- weak control over token usage during large result-set exploration,
- inconsistent safety behavior around destructive SQL,
- credential handling that often leaks secrets into agent-visible context,
- poor portability across different agent runtimes.

The chosen direction is a pure CLI because it works with any shell-capable agent, keeps the runtime model simple, and avoids protocol lock-in. The companion skill carries the usage policy and best practices so the agent learns how to use the CLI safely and efficiently.

## Decision Record

The following decisions were made during design discussion and are part of the v1 contract.

| Topic | Decision |
| --- | --- |
| Architecture | Pure CLI, not MCP |
| Implementation language | Go |
| Command style | Subcommands |
| Output format | JSON only |
| Session model | Stateless per command |
| SQL execution unit | Single statement only |
| Databases | MySQL, PostgreSQL |
| Result optimization | Deterministic pagination and truncation |
| Write support | Allowed in v1 with safety confirmation |
| Dangerous SQL policy | Warn and require `--confirm` |
| Read-only mode | Connection-level `--read-only` flag on `connect` |
| Total count behavior | Cheap-first, exact totals optional/conditional |
| Credential sources | Chat-provided, environment variables, credential helper |
| Credential encryption key | Environment variable only |
| Credential helper protocol | Minimal stdin/stdout JSON |
| Automatic masking | None in v1 |
| Skill packaging | Portable folder containing `SKILL.md` |
| Release scope | Local script-driven build/package for Linux/macOS, amd64/arm64 |
| Test strategy | Local script-driven MySQL/Postgres container tests |

## Non-Goals

The following are intentionally out of scope for v1:

- MCP server support.
- SQL generation from natural language.
- Multi-statement execution.
- Interactive shell mode.
- Automatic semantic summarization of query results.
- Automatic sensitive-data masking.
- Database-specific features beyond generic SQL behavior.
- Windows release artifacts.
- Automatic installation of the skill into agent tools.

## Product Goals

The CLI should optimize for the way an AI agent actually works:

- inspect schema before querying,
- run many short, targeted commands,
- avoid flooding the context window,
- surface safety boundaries explicitly,
- return machine-readable output with minimal ambiguity.

The CLI should be boring, predictable, and easy for a skill to teach.

## User Model

There are two users in practice:

1. The human operator, who decides which database the agent may access and how credentials are supplied.
2. The AI agent, which invokes the CLI and consumes its JSON output.

Important boundary: if a human pastes credentials into chat, the underlying LLM has already seen them. The CLI can still avoid plain-text persistence, but it cannot retroactively hide those credentials from the model. The only source that keeps secrets fully out of model context is the credential-helper path.

## High-Level Architecture

```text
Human
  |
  | provides connection policy / helper / env
  v
AI Agent
  |
  | shell command
  v
sql-agent-cli
  |
  +--> connection resolver
  |      +--> encrypted local store
  |      +--> environment variables
  |      +--> credential helper
  |
  +--> SQL safety analyzer
  |
  +--> database driver adapter
  |      +--> mysql
  |      +--> postgres
  |
  +--> result formatter
         +--> pagination
         +--> truncation
         +--> JSON envelope
```

## CLI Surface

The v1 command surface is intentionally small.

### `sql-agent connect <name>`

Stores a named connection configuration. Credentials may come from:

- explicit flags,
- environment variables,
- a credential helper command.

Examples:

```bash
sql-agent connect analytics --driver postgres --host db.example.com --port 5432 --database app --username analyst --password-env ANALYTICS_DB_PASSWORD
sql-agent connect prod --driver mysql --credential-helper "my-helper --profile prod"
```

### `sql-agent disconnect <name>`

Removes the named connection from local encrypted storage.

### `sql-agent query <name> <sql>`

Executes one SQL statement and returns JSON.

Capabilities:

- read queries,
- write queries,
- pagination for result sets,
- `--confirm` gate for dangerous SQL,
- consistent metadata for row counts and truncation.

### `sql-agent schema list <name>`

Returns tables and views for the connection.

### `sql-agent schema describe <name> <table>`

Returns columns, types, nullability, primary key info, and indexes when available.

### `sql-agent count <name> <target>`

Provides an explicit counting tool so the agent can ask for exact counts when needed instead of forcing exact count behavior into every paginated query.

`<target>` may be either:

- a table name, or
- a single read-only SQL subquery input.

## SQL Execution Rules

### Single-Statement Rule

v1 accepts exactly one SQL statement per invocation.

Rejected examples:

```sql
SELECT * FROM users; SELECT * FROM orders;
BEGIN; DELETE FROM users; COMMIT;
```

Reasons:

- safer analysis,
- simpler result model,
- easier error attribution,
- simpler pagination behavior,
- smaller blast radius.

### SQL Type Support

Allowed categories:

- `SELECT`
- `WITH ... SELECT`
- `INSERT`
- `UPDATE`
- `DELETE`
- `CREATE`, `ALTER`, `DROP`, `TRUNCATE`
- metadata queries supported by the backend adapter

The CLI does not promise support for stored procedures, multiple result sets, or backend-specific scripting behavior.

## Dangerous SQL Policy

The CLI must classify statements before execution.

Statements requiring `--confirm` include at minimum:

- `DELETE`
- `DROP`
- `TRUNCATE`
- `ALTER`
- `UPDATE` without a `WHERE`
- bulk-write patterns that materially change many rows

Behavior:

1. Without `--confirm`, the CLI does not execute the statement.
2. It returns structured JSON describing why the statement is blocked.
3. The skill instructs the agent to show that warning to the human before retrying with `--confirm`.

Important architectural boundary: the CLI adds a human-confirmation guard, but it is not a replacement for least-privilege database accounts. Production safety still depends on database-side permissions.

## Pagination And Token Optimization

v1 uses deterministic output reduction rules only. It does not attempt semantic summarization.

### Pagination

Default query pagination:

- default page size: 50 rows,
- configurable with `--page-size`,
- configurable with `--page`,
- `LIMIT page_size + 1` strategy used where possible to compute `has_more` cheaply.

### Exact Total Strategy

The CLI follows cheap-first semantics.

Default behavior:

- always return the current page,
- try to determine `has_more`,
- do not force an exact total count for every query,
- return exact totals only when explicitly requested or when the adapter can provide them cheaply and safely.

This avoids turning ordinary exploration queries into expensive `COUNT(*)` work on large datasets.

### Truncation Rules

For result sets, the formatter should:

- truncate long text values after a configured limit,
- replace blob/binary values with size summaries,
- keep column names in a shared `columns` array,
- return row values as arrays instead of row objects to reduce repeated tokens.

Example result shape:

```json
{
  "ok": true,
  "data": {
    "columns": ["id", "email", "bio"],
    "rows": [
      [1, "alice@example.com", "Long text truncated..."],
      [2, "bob@example.com", "Another truncated value..."]
    ],
    "page": 1,
    "page_size": 50,
    "returned_rows": 2,
    "has_more": false,
    "next_page": null,
    "total_rows_exact": null,
    "remaining_rows_exact": null,
    "truncated": true,
    "truncated_cells": 2
  },
  "warning": null,
  "duration_ms": 14
}
```

## Output Contract

All commands return JSON.

Top-level envelope:

```json
{
  "ok": true,
  "data": {},
  "warning": null,
  "error": null,
  "duration_ms": 0
}
```

Error envelope:

```json
{
  "ok": false,
  "data": null,
  "warning": null,
  "error": {
    "code": "destructive_query_requires_confirmation",
    "message": "DELETE requires --confirm",
    "details": {
      "statement_type": "DELETE"
    }
  },
  "duration_ms": 0
}
```

Conventions:

- `ok` is required.
- `error.code` is stable and machine-consumable.
- `warning` is advisory and does not replace `error`.
- output remains compact and predictable across commands.

## Credentials And Connection Storage

### Connection Sources

The CLI supports three credential acquisition paths.

#### 1. Chat-provided or flag-provided connection values

The agent calls `connect` with explicit fields or env-backed password flags.

Pros:

- simple,
- easy to bootstrap.

Limitation:

- if the secret originated in the conversation, the model already saw it.

#### 2. Environment variables

The CLI may read connection values or passwords from environment variables.

This avoids persisting secrets in shell history when used carefully.

#### 3. Credential helper

The CLI invokes an external helper and receives credentials over stdin/stdout JSON.

This is the preferred path for secrets the model should not see.

### Encryption Model

Named connection records are stored locally in encrypted form.

Key decision:

- the encryption master key comes from an environment variable, or is auto-generated if missing.
- v1 does not use the OS keyring.

Implications:

- simpler and more portable implementation,
- explicit operational model,
- users can provide `DB_AGENT_MASTER_KEY` for reproducible encryption, or let the CLI generate one automatically.

Recommended variable name:

```text
DB_AGENT_MASTER_KEY
```

Auto-generated key location:

```text
~/.sql-agent/.master_key
```

### Credential Helper Protocol

The helper contract is a minimal JSON protocol.

Request example:

```json
{
  "version": 1,
  "action": "resolve",
  "connection": "prod",
  "driver": "postgres"
}
```

Response example:

```json
{
  "ok": true,
  "credentials": {
    "host": "db.example.com",
    "port": 5432,
    "database": "app",
    "username": "agent",
    "password": "secret"
  }
}
```

The CLI must treat helper stderr as diagnostic only and never parse it as credentials.

## Security Model

Security goals for v1:

- reduce accidental destructive actions,
- keep structured secrets out of persistent plain-text storage,
- support a secret-provider path that avoids leaking credentials into the model,
- keep the execution surface narrow and inspectable.

Security principles:

- single statement only,
- JSON-only output,
- explicit confirmation for dangerous SQL,
- no hidden retries on destructive queries,
- no silent masking that changes query semantics,
- no magic query rewriting beyond pagination wrappers required for deterministic formatting.

## Driver Abstraction

The CLI should hide backend differences behind a small driver interface.

Proposed responsibilities per driver:

- connect,
- execute a single statement,
- inspect schema,
- detect dialect capabilities relevant to pagination and metadata,
- normalize native values for JSON output.

The abstraction should stay small. v1 should not invent a large ORM-like layer.

## Suggested Project Layout

```text
sql-agent-cli/
├── cmd/
│   └── sql-agent/
│       └── main.go
├── internal/
│   ├── cli/
│   │   ├── root.go
│   │   ├── connect.go
│   │   ├── disconnect.go
│   │   ├── query.go
│   │   ├── schema.go
│   │   └── count.go
│   ├── config/
│   │   ├── store.go
│   │   └── crypto.go
│   ├── credentials/
│   │   ├── resolver.go
│   │   └── helper.go
│   ├── safety/
│   │   └── analyzer.go
│   ├── db/
│   │   ├── driver.go
│   │   ├── mysql.go
│   │   ├── postgres.go
│   └── output/
│       ├── envelope.go
│       └── formatter.go
├── skill/
│   ├── SKILL.md
│   └── README.md
├── scripts/
│   └── build-skill.sh
├── .github/
│   └── workflows/
│       ├── ci.yml
│       └── release.yml
├── docs/
│   └── superpowers/
│       ├── specs/
│       └── plans/
├── go.mod
└── README.md
```

## Skill Package Design

The skill should be stored in a standalone `skill/` directory and packaged as a folder that users can copy into any compatible skill-based agent environment.

The skill must teach these behaviors:

- prefer `schema list` and `schema describe` before broad queries,
- avoid `SELECT *` unless truly necessary,
- use pagination intentionally,
- inspect `has_more` before fetching additional pages,
- ask the human before retrying a blocked destructive query with `--confirm`,
- prefer the credential-helper path when secrets should stay out of model context.

No automatic installer is required in v1.

## Distribution And Local Packaging

Because this is a new CLI artifact, build and packaging workflows are part of the design, not an afterthought.

v1 packaging expectations:

- local script-driven build entrypoints,
- Linux and macOS targets,
- `amd64` and `arm64` artifacts,
- reproducible local packaging into a `dist/` directory,
- no dependency on GitHub-hosted CI for normal development or packaging.

Windows is explicitly deferred.

Expected script surface:

- `scripts/build.sh`: build local binary for current host,
- `scripts/test.sh`: run unit and local integration test workflow,
- `scripts/package.sh`: produce release-like archives into `dist/`,
- `scripts/build-skill.sh`: package the portable skill folder.

## Testing Strategy

### Unit Tests

Required for:

- SQL danger classification,
- pagination logic,
- JSON formatting,
- credential helper protocol handling,
- encrypted store encode/decode,
- command argument validation.

### Integration Tests

Local scripts should run containerized integration tests for:

- MySQL,
- PostgreSQL.

Integration tests must verify:

- connection setup,
- schema listing and describing,
- paginated reads,
- dangerous query blocking,
- confirmed write execution.

## Failure Modes To Design For

The implementation and tests should explicitly cover these cases:

1. Missing `DB_AGENT_MASTER_KEY` with no auto-generated fallback (edge case: permission denied on home directory).
2. Credential helper returns malformed JSON.
3. Credential helper exits successfully but omits required fields.
4. A dangerous query is run without `--confirm`.
5. The query result exceeds the default page size.
6. The backend returns binary values that cannot be serialized directly.
7. The SQL text contains more than one statement.
8. The agent requests an out-of-range page.
9. A driver-specific schema inspection feature is partially unavailable.

## Rationale For Rejected Alternatives

### MCP-first design

Rejected for v1 because it narrows compatibility and adds protocol complexity before the core behavior is proven.

### Interactive shell session

Rejected for v1 because stateless commands are easier for agents to reason about, easier to test, and easier to secure.

### Semantic result summarization

Rejected for v1 because it would turn the CLI into an opinionated inference layer instead of a deterministic transport-and-formatting layer.

### Keyring-based encryption

Rejected for v1 because the chosen requirement favored simpler, explicit, environment-only key management.

### Automatic sensitive-field masking

Rejected for v1 because it can silently change data visibility and create false confidence. The v1 skill should teach safer query shape instead.

## Success Criteria

v1 is successful if:

- an agent can connect to either supported database,
- schema exploration is easy and cheap,
- large result sets do not flood the context window,
- dangerous SQL is blocked until explicit confirmation,
- local credentials are not persisted in plain text,
- the skill gives a compatible agent a reliable operating playbook,
- users can produce local packaged binaries and skill bundles from scripts without extra CI setup.

## Spec Notes For The Implementation Plan

The implementation plan should preserve these priorities:

1. deterministic behavior over clever behavior,
2. small command surface,
3. safety before convenience,
4. local build/package pipeline included in v1,
5. tests for each database promise made in the README.
