<div align="center">

# sql-agent-cli

[![Build](https://img.shields.io/badge/build-passing-success)](scripts/build.sh)
[![Go Version](https://img.shields.io/badge/go-1.24+-00ADD8?logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

**Agent-First SQL Gateway.**

Your AI agent should explore databases without drowning in tokens or leaking secrets.

[English](README.md) | [中文](README_CN.md)

</div>

---

## What is Agent-First?

Traditional database tools are built for humans. `sql-agent-cli` is built for **AI agents** that operate inside your context window.

When an agent queries a database, three things go wrong:

1. **Token Flooding** — A single `SELECT *` can dump 10,000 rows into the agent's context, burning thousands of tokens on noise.
2. **Secret Exposure** — When agents handle connection strings directly, passwords end up in model logs, shell history, and chat transcripts.
3. **Accidental Destruction** — A typo in a `DELETE` or `UPDATE` can erase production data before anyone notices.

`sql-agent-cli` solves all three by acting as a **proxy layer** between your agent and the database. The agent never sees credentials. Results are automatically compressed and paginated. Dangerous operations require explicit human confirmation.

## Key Design

### Token-Efficient Results

Every query result is automatically optimized for your agent's context window:

- **SQL-Level Pagination** — `LIMIT` is injected at the database layer, not in memory. The database only returns what the agent can handle.
- **Smart Truncation** — Long text and blob fields are summarized, not dumped
- **Deterministic Pagination** — Default 50 rows per page with `has_more` metadata; agent decides if more data is worth the tokens
- **Column Array Format** — `{"columns": [...], "rows": [...]}` instead of repeated object keys, saving ~30% tokens on wide tables
- **Explore Mode** — `--explore` returns a structured summary with column names, sample rows, and a cheap row-count estimate for simple table reads instead of raw paginated output. Perfect for first contact with an unfamiliar table.

### Security by Proxy

The agent **never touches** database credentials:

- **Credential Helpers** — Secrets resolved via external JSON helpers; passwords never enter the agent's context
- **Encrypted Storage** — Named connections stored with AES-256-GCM; master key auto-generated if not provided
- **Explicit Confirmation** — `DELETE`, `DROP`, `TRUNCATE`, `ALTER`, and broad `UPDATE` are blocked without `--confirm`
- **Single-Statement Limit** — One SQL per invocation prevents complex injection patterns

### Zero Dependencies

Copy the skill package to your agent's directory and run:

```bash
# OpenCode / Claude Code
cp -R dist/sql-agent ~/.config/opencode/skills/

# Or Cursor, Copilot CLI, etc.
# The embedded binary matches your platform automatically
```

No Docker, no database drivers, no runtime configuration required.

## Quick Start

### For Agent Users

```bash
# 1. Build the skill (one command)
bash scripts/build-skill.sh

# 2. Copy to your agent's skills directory
cp -R dist/sql-agent ~/.config/opencode/skills/

# 3. The agent now knows how to use sql-agent-cli
```

### For Developers

```bash
# Clone and build
git clone https://github.com/davidfantasy/sql-agent-cli.git
cd sql-agent-cli
bash scripts/build.sh

# List existing connections before creating a new one
./bin/sql-agent connection list

# Connect from your own terminal (local password prompt, verified before save)
./bin/sql-agent connection add analytics --wizard

# Or let an agent connect automatically with a password env var
export DB_PASSWORD='your-password'
./bin/sql-agent connection add analytics --driver postgres \
  --host db.example.com --port 5432 --database app \
  --username analyst --password-env DB_PASSWORD

# Or use a credential helper (recommended)
./bin/sql-agent connection add prod --driver mysql \
  --credential-helper "vault-helper --profile prod"

# Create a read-only connection (rejects all write/DDL statements)
./bin/sql-agent connection add analytics --driver postgres \
  --host db.example.com --database app \
  --username analyst --password-env DB_PASSWORD --read-only

# Remove a connection
./bin/sql-agent connection remove analytics
```

`connection add` verifies the database connection before saving it. Non-interactive mode requires `--password-env` or `--credential-helper`. If `--password-env DB_PASSWORD` is set but `DB_PASSWORD` is missing, the CLI returns a direct hint such as `export DB_PASSWORD=your-password` so the agent can ask the user to provide it outside the chat.

### Explore and Query

```bash
# Discover schema first
./bin/sql-agent schema list analytics
./bin/sql-agent schema describe analytics users

# Explore mode: summary instead of raw rows (saves tokens on first contact)
./bin/sql-agent query analytics "SELECT * FROM users" --explore

# Query with automatic pagination (LIMIT injected at database layer)
./bin/sql-agent query analytics \
  "SELECT id, email, created_at FROM users ORDER BY id" --page 1

# Ask for an exact count when you need one
./bin/sql-agent count analytics users
./bin/sql-agent count analytics "SELECT * FROM users WHERE active = true"

# Dangerous operations are blocked
./bin/sql-agent query prod "DELETE FROM sessions WHERE created_at < NOW() - INTERVAL '30 days'"
# Error: destructive_query_requires_confirmation

# Only proceed after human approval
./bin/sql-agent query prod "DELETE FROM sessions WHERE created_at < NOW() - INTERVAL '30 days'" --confirm
```

## Output Format

```json
{
  "ok": true,
  "data": {
    "columns": ["id", "email", "created_at"],
    "rows": [
      [1, "alice@example.com", "2024-01-15T09:30:00Z"],
      [2, "bob@example.com", "2024-02-20T14:45:00Z"]
    ],
    "page": 1,
    "page_size": 50,
    "returned_rows": 2,
    "has_more": false,
    "next_page": null,
    "truncated": false,
    "truncated_cells": 0
  },
  "warning": null,
  "error": null,
  "duration_ms": 14
}
```

## Architecture

```
┌─────────────┐     ┌──────────────────┐     ┌──────────────┐
│  AI Agent   │────▶│  sql-agent-cli   │────▶│   Database   │
│  (Context)  │     │  (Proxy + Gate)  │     │ (MySQL/PSQL) │
└─────────────┘     └──────────────────┘     └──────────────┘
                           │
                    ┌──────┴──────┐
                    ▼             ▼
            ┌──────────┐  ┌────────────┐
            │  Safety  │  │  Credential│
            │   Gate   │  │   Helper   │
            └──────────┘  └────────────┘
```

The agent interacts with a **controlled interface**, not the raw database. The CLI handles:
- Connection encryption and resolution
- SQL safety analysis
- Result compression and pagination
- Secret isolation via helpers

## Roadmap

| Status | Feature |
|--------|---------|
| ✅ | MySQL support |
| ✅ | PostgreSQL support |
| ✅ | Credential helper protocol |
| ✅ | Encrypted connection storage |
| ✅ | Dangerous write confirmation |
| ✅ | Automatic token optimization |
| ✅ | SQL-level pagination injection |
| ✅ | Explore mode (--explore) |
| 📋 | Microsoft SQL Server |
| 📋 | Oracle Database |

## Security Model

| Threat | Mitigation |
|--------|------------|
| Token exhaustion | Smart truncation + pagination + column array format |
| Credential leakage | Agent never sees passwords; helpers resolve externally |
| Accidental writes | Dangerous SQL blocked without `--confirm` |
| Local storage breach | AES-256-GCM encryption; auto-generated master key |
| Injection | Single-statement limit; no multi-query execution |

## Development

```bash
# Format code
bash scripts/format.sh

# Build binary
bash scripts/build.sh

# Run tests (unit + Docker integration)
bash scripts/test.sh

# Package skill with embedded binary
bash scripts/build-skill.sh

# Build release archives for all platforms
bash scripts/package.sh
```

See [AGENTS.md](AGENTS.md) for architecture details and contribution guidelines.

## License

[MIT](LICENSE)
