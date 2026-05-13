---
name: sql-agent
description: Use when an agent needs to connect to, inspect, query, or modify MySQL or PostgreSQL through sql-agent-cli. Invoke this whenever the user mentions database setup, connection onboarding, password-env configuration, credential helpers, schema-first querying, pagination discipline, or dangerous write confirmation, even if they do not explicitly mention sql-agent-cli.
---

# sql-agent Skill

## Overview
Use this skill to treat `sql-agent-cli` as a disciplined agent-facing database tool, not as a generic SQL shell. The goal is to keep queries narrow, output compact, writes gated, and secrets handled through the safest available source.

The CLI binary is embedded in this skill package at `bin/sql-agent`. Use the relative path when invoking commands.

## When To Use
- The user asks you to inspect, query, or update a SQL database through this repository's CLI.
- The task involves schema discovery before querying.
- The task may touch large tables where pagination and token control matter.
- The task may require a write query that should stop for human approval before `--confirm`.
- The task involves choosing between direct credentials, environment variables, and credential helpers.

Do not use raw `psql` or `mysql` when `sql-agent-cli` is the intended path.

## Agent Workflow (MANDATORY)

When the user asks you to perform a database operation **without providing a connection name**, follow this exact flow. Do not skip steps.

### Step 1: Check existing connections

Always run this first:

```bash
./bin/sql-agent connection list
```

### Step 2: If connections exist, ask the user to confirm

Present the **IP address (host)** and **database name** of every saved connection, and ask the user which one to use. Do not proceed until the user explicitly confirms.

Example confirmation prompt:

> I found saved connections:
> - `prod` → MySQL at **10.20.5.34:23306**, database **zhongya**
> - `local` → PostgreSQL at **127.0.0.1:5432**, database **app**
> Which connection should I use?

### Step 3: If no matching connection exists, guide the user to create one

If no suitable connection is found, **provide example commands** and ask the user to run them in their terminal. **Do not ask for the password in chat unless the user explicitly says they do not know how to set it up.**

**Recommended example (permanent, encrypted password):**

```bash
# Replace these values with your actual database credentials:
./bin/sql-agent connection add mydb \
  --driver mysql \
  --host YOUR_DB_HOST \
  --port 3306 \
  --database YOUR_DB_NAME \
  --username YOUR_USERNAME \
  --credential-helper 'sql-agent-inline://YOUR_PASSWORD'
```

After the user creates the connection, it is saved permanently and you can reuse it in future sessions without asking for credentials again.

### Step 4: Only ask for full credentials as a last resort

If the user **explicitly says they do not know how to create the connection**, only then ask for all required information (host, port, database, username, password), and create the connection on their behalf using `--credential-helper`.

**Never create a connection silently without user confirmation of the target database.**

## Command Reference

### Connection Management
- `./bin/sql-agent connection list` — List all saved connections (no secrets exposed).
- `./bin/sql-agent connection add <name>` — Create, verify, and save a new connection.
- `./bin/sql-agent connection remove <name>` — Delete a saved connection.
- `./bin/sql-agent connection show <name>` — Display connection details (no secrets exposed).

### Database Operations
- `./bin/sql-agent schema list <connection>` — List tables and views.
- `./bin/sql-agent schema describe <connection> <table>` — Describe one table.
- `./bin/sql-agent query <connection> <sql>` — Execute one SQL statement.
- `./bin/sql-agent count <connection> <target>` — Count rows in a table or subquery.

## Connection Setup Decision Tree

### Option A: Permanent (password encrypted, survives across sessions) — RECOMMENDED
The password is stored encrypted with AES-256-GCM in `~/.sql-agent/connections/<name>.enc`.

1. **Interactive wizard** (`--wizard` or `-i`): Use when the human is at their own terminal and can type the password directly. The password never appears in command history or model context.
   ```bash
   ./bin/sql-agent connection add prod --wizard
   ```

2. **Inline credential helper** (`--credential-helper 'sql-agent-inline://密码'`): Use for non-interactive agent setup. The password is embedded into the encrypted connection file. **Caution:** The password is exposed on the command line at creation time (visible in shell history and process lists), but once saved it is encrypted at rest.
   ```bash
   ./bin/sql-agent connection add prod --driver mysql --host db.example.com --database app --username root --credential-helper 'sql-agent-inline://secret123'
   ```

### Option B: Temporary (password NOT saved, must be set each session)
Only the environment variable **name** is saved. The actual password must be exported in the current shell before every command.

```bash
export DB_PASSWORD='your-password'
./bin/sql-agent connection add prod --driver mysql --host db.example.com --database app --username root --password-env DB_PASSWORD
```

If the CLI says `password env "DB_PASSWORD" is not set`, tell the human exactly what to run (`export DB_PASSWORD='your-password'`), then retry.

### Security Summary
| Method | Password on disk | Password in command history | Password in model context | Human interaction required |
|--------|------------------|----------------------------|---------------------------|---------------------------|
| `--wizard` | Encrypted | No | No | Yes |
| `--credential-helper` (inline) | Encrypted | **Yes** (at creation) | Yes (at creation) | No |
| `--password-env` | Not saved | No | No | No (but human must export env) |

## Core Pattern
1. Check `./bin/sql-agent connection list` before creating anything new.
2. If the connection is missing, choose the setup method based on the table above.
3. Discover schema.
4. Narrow to the smallest useful query.
5. Read pagination metadata before asking for more data.
6. Treat dangerous write warnings as a hard stop, not a routine retry.

## Dangerous Writes
- `DELETE`, `DROP`, `TRUNCATE`, `ALTER`, and broad `UPDATE` statements are not normal retries.
- When the CLI returns a confirmation error, copy that warning back to the human.
- Ask for a yes/no decision in plain language.
- Do not add `--confirm` unless the human explicitly approves that exact operation.
- If scope is still unclear, inspect schema or run a narrower read query first.

## Example
```bash
# Check existing connections first
./bin/sql-agent connection list

# Create a permanent connection (inline helper, agent-automated)
./bin/sql-agent connection add analytics --driver postgres --host db.example.com --database app --username analyst --credential-helper 'sql-agent-inline://secret123'

# Explore schema
./bin/sql-agent schema list analytics
./bin/sql-agent schema describe analytics users

# Query with pagination
./bin/sql-agent query analytics "SELECT id, email, created_at FROM users ORDER BY id" --page 1
```

## Common Mistakes
- **Skipping `connection list`** and creating duplicate connections.
- **Using `--password-env` when permanent storage is possible** — this forces the user to re-export the password every session.
- **Asking the human to paste database passwords into chat** when `--wizard`, `--credential-helper`, or `--password-env` would avoid it.
- Using `SELECT *` during exploration.
- Increasing page size before tightening predicates.
- Treating `--confirm` as routine instead of an explicit human approval boundary.
- **Silently creating connections** without showing the user the target host and database name for confirmation.
