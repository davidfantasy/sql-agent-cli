#!/usr/bin/env bash
set -euo pipefail

export DB_AGENT_MASTER_KEY="${DB_AGENT_MASTER_KEY:-test-master-key}"

go test ./internal/... ./cmd/...

if ! command -v docker >/dev/null 2>&1; then
  printf 'docker is required for v1 integration tests\n' >&2
  exit 1
fi

docker info >/dev/null 2>&1 || {
  printf 'docker daemon is not available\n' >&2
  exit 1
}

go test ./tests/integration -v
