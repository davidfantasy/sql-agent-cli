---
name: sql-agent
description: Use when an agent needs to inspect or modify MySQL or PostgreSQL through sql-agent-cli, especially when schema-first querying, pagination discipline, dangerous write confirmation, or credential-helper handling matters.
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
1. Discover schema.
2. Narrow to the smallest useful query.
3. Read pagination metadata before asking for more data.
4. Treat dangerous write warnings as a hard stop, not as a routine retry.
5. Prefer helper-based credentials over plaintext secrets.

## Dangerous Writes
- `DELETE`, `DROP`, `TRUNCATE`, `ALTER`, and broad `UPDATE` statements are not normal retries.
- When the CLI returns a confirmation error, copy that warning back to the human.
- Ask for a yes/no decision in plain language.
- Do not add `--confirm` unless the human explicitly approves that exact operation.
- If scope is still unclear, inspect schema or run a narrower read query first.

## Example
```bash
./bin/sql-agent schema list analytics
./bin/sql-agent schema describe analytics users
./bin/sql-agent query analytics "SELECT id, email, created_at FROM users ORDER BY id" --page 1
```

## Common Mistakes
- Skipping schema inspection and guessing table shape.
- Using `SELECT *` during exploration.
- Increasing page size before tightening predicates.
- Treating `--confirm` as routine instead of an explicit human approval boundary.
- Putting secrets into command lines when a credential helper would avoid exposing them.
