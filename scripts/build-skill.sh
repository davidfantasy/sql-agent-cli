#!/usr/bin/env bash
set -euo pipefail

SKILL_NAME="sql-agent"
GOOS=$(go env GOOS)
GOARCH=$(go env GOARCH)

rm -rf "dist/${SKILL_NAME}"
mkdir -p "dist/${SKILL_NAME}/bin"

# Build platform-specific binary
GOOS="$GOOS" GOARCH="$GOARCH" go build -o "dist/${SKILL_NAME}/bin/sql-agent" ./cmd/sql-agent

# Copy skill docs and evals
cp -R "skill/${SKILL_NAME}/"* "dist/${SKILL_NAME}/"

echo "Built ${SKILL_NAME} for ${GOOS}-${GOARCH}"
