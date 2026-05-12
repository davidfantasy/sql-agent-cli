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

## Quick Reference
- If the named connection does not exist yet, create it with `./bin/sql-agent connect <connection> --wizard` when the human is at their own terminal.
- If the human wants the agent to connect automatically, use `./bin/sql-agent connect <connection> --driver ... --database ... --password-env ENV_NAME` or `--credential-helper ...`.
- Explain `--password-env` concretely: the human must export the variable in their shell first, for example `export DB_PASSWORD='your-password'`, then the agent can run `./bin/sql-agent connect prod --password-env DB_PASSWORD ...`.
- If `sql-agent connect` says the password env is missing, copy the env name back to the human and ask them to export it in their terminal. Do not ask them to paste the password into chat unless they explicitly insist.
- Start with `./bin/sql-agent schema list <connection>`.
- Narrow with `./bin/sql-agent schema describe <connection> <table>`.
- Query only the columns you need.
- Use the default page size first.
- Read `has_more` and `next_page` before fetching more data.
- If the CLI blocks a destructive query, stop immediately, show the warning to the human, and only retry with `--confirm` after explicit human approval.
- Prefer credential helpers when secrets should stay out of model context.
- Use `--read-only` when connecting to production databases to prevent accidental writes.
- Read-only connections reject all write/DDL statements with a clear error code.

## Core Pattern
1. If the connection is missing, decide whether the human should run `connect --wizard` locally or whether the agent can run `connect` with `--password-env` or `--credential-helper`.
2. Discover schema.
3. Narrow to the smallest useful query.
4. Read pagination metadata before asking for more data.
5. Treat dangerous write warnings as a hard stop, not as a routine retry.
6. Prefer helper-based credentials, then password env vars, before asking for any secret directly.

## Connection Setup Rules
- `connect` verifies the database connection before saving it. Failed verification means there is no usable saved connection yet.
- Use `--wizard` when the human is on the same machine and wants to enter the password locally without exposing it to the agent.
- Use non-interactive `connect` only when the password comes from `--password-env` or `--credential-helper`.
- Do not put plaintext passwords on the command line.
- If the CLI says `password env "DB_PASSWORD" is not set`, tell the human exactly what to run, for example `export DB_PASSWORD='your-password'`, then rerun `connect`.
- After a successful `connect`, continue using the saved connection name. Do not keep re-requesting the secret.

## Dangerous Writes
- `DELETE`, `DROP`, `TRUNCATE`, `ALTER`, and broad `UPDATE` statements are not normal retries.
- When the CLI returns a confirmation error, copy that warning back to the human.
- Ask for a yes/no decision in plain language.
- Do not add `--confirm` unless the human explicitly approves that exact operation.
- If scope is still unclear, inspect schema or run a narrower read query first.

## Example
```bash
./bin/sql-agent connect analytics --wizard
export DB_PASSWORD='your-password'
./bin/sql-agent connect analytics --driver postgres --host db.example.com --database app --username analyst --password-env DB_PASSWORD
./bin/sql-agent schema list analytics
./bin/sql-agent schema describe analytics users
./bin/sql-agent query analytics "SELECT id, email, created_at FROM users ORDER BY id" --page 1
```

## Common Mistakes
- Skipping schema inspection and guessing table shape.
- Asking the human to paste database passwords into chat when `--wizard`, `--password-env`, or `--credential-helper` would avoid it.
- Using `SELECT *` during exploration.
- Increasing page size before tightening predicates.
- Treating `--confirm` as routine instead of an explicit human approval boundary.
- Putting secrets into command lines when a credential helper would avoid exposing them.
