# AGENTS.md

## 1. Project Overview

`sql-agent-cli` is a Unix-style SQL CLI designed for AI agents. Its goal is to give agents a safer, more token-efficient, and more predictable way to access MySQL and PostgreSQL. The tech stack is Go + Cobra + `database/sql`. The repository is organized by responsibility into `cmd/` (entry points), `internal/` (core packages), `tests/` (integration tests), `scripts/` (local build and packaging scripts), `skill/` (portable skill packages), and `docs/` (design and architecture documents).

## 2. Quick Commands

- Build: `bash scripts/build.sh`
- Local run: `bash scripts/dev.sh --help`
- Format: `bash scripts/format.sh`
- Test: `bash scripts/test.sh`
- Package CLI: `bash scripts/package.sh`
- Package skill: `bash scripts/build-skill.sh`

Environment variables:
- `DB_AGENT_MASTER_KEY`: Optional. Auto-generated and saved to `~/.sql-agent/.master_key` if not provided.
- `VERSION`: Version number for `scripts/package.sh` output. Default: `dev`.

## 3. Behavioral Guidelines

- Prefer minimal, correct changes. Avoid premature abstraction.
- Strictly adhere to schema-first, single-statement, and JSON-only product boundaries.
- Do not treat `sql-agent-cli` as a generic SQL shell.
- Do not introduce MCP, frontend, Windows packaging, or cloud CI capabilities beyond the current scope.
- Keep documentation consistent across spec, plan, README, and AGENTS.md.
- Never silently relax dangerous write boundaries. `--confirm` is an explicit human approval point.

Karpathy-style AI programming rules, adapted for this repository:
- Understand boundaries before writing code.
- Write the minimal skeleton before adding details.
- Validate locally before declaring completion.
- Update documentation before leaving the scene.

## 4. Backend Architecture

Package structure:

```text
cmd/
  sql-agent/          CLI entry point
internal/
  cli/                Subcommand definitions and argument entry points
  config/             Connection config and encrypted storage models
  credentials/        Environment variable / credential helper parsing
  safety/             Single-statement and dangerous SQL analysis
  db/                 MySQL/PostgreSQL driver abstraction
  output/             JSON envelope and pagination/truncation models
tests/
  integration/        Multi-database integration test framework
```

Package responsibilities:
- `internal/cli`: Responsible only for command tree, arguments, and call boundaries. No database logic.
- `internal/config`: Responsible for named connections and local encrypted storage contracts.
- `internal/credentials`: Responsible for environment variable and credential helper merged resolution.
- `internal/safety`: Responsible for single-statement and dangerous SQL classification.
- `internal/db`: Responsible for unified driver interface and dialect differences.
- `internal/output`: Responsible for stable JSON output contracts and token-optimized formatting.

Core subsystem summary:
- Connection resolution: Local named connections + env + helper
- Safety gate: Dangerous write confirmation boundary
- Driver layer: Unified abstraction for both databases
- Output layer: Pagination, truncation, JSON envelope

Detailed documentation:
- `docs/architecture/backend.md`
- `docs/superpowers/specs/2026-05-11-sql-agent-cli-design.md`

## 5. Frontend Architecture

No frontend currently. Do not invent UI structures, routing, component systems, or API SDK conventions.

If a frontend is added in the future, document:
- Tech stack
- Routing approach
- API layer conventions
- Component library standards

Placeholder:
- `docs/architecture/frontend.md`

## 6. Local Development and Validation Loop

1. Edit code or documentation.
2. Run `bash scripts/format.sh`.
3. Run `bash scripts/build.sh`.
4. Run `bash scripts/test.sh`.
5. If modifying skill files, run `bash scripts/build-skill.sh`.
6. If modifying packaging logic, run `bash scripts/package.sh`.
7. Check if `README.md`, `AGENTS.md`, and relevant `docs/` need synchronization.

## 7. Quality Checks

- format: `bash scripts/format.sh`
- build: `bash scripts/build.sh`
- test: `bash scripts/test.sh`
- package-cli: `bash scripts/package.sh`
- package-skill: `bash scripts/build-skill.sh`

Command matrix:

```text
Change Go code       -> format + build + test
Change skill         -> build-skill
Change packaging     -> package
Change arch/behavior -> check spec/plan/README/AGENTS consistency
```

## 8. Documentation Rules

Document navigation:
- Design spec: `docs/superpowers/specs/2026-05-11-sql-agent-cli-design.md`
- Implementation plan: `docs/superpowers/plans/2026-05-11-sql-agent-cli.md`
- Backend architecture: `docs/architecture/backend.md`
- Frontend placeholder: `docs/architecture/frontend.md`
- Local development: `docs/development.md`
- Skill package: `skill/sql-agent/SKILL.md`

Update rules:
- Change CLI command surface: Update README, AGENTS, spec.
- Change package structure or responsibilities: Update AGENTS, `docs/architecture/backend.md`.
- Change local scripts: Update README, AGENTS, `docs/development.md`.
- Change skill triggers or behavior: Update `skill/sql-agent/SKILL.md`, README, related evals.
- Change scope boundaries: Update spec and plan to prevent implementation drift.
