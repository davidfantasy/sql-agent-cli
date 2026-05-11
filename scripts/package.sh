#!/usr/bin/env bash
set -euo pipefail

VERSION="${VERSION:-dev}"
OUT_DIR="dist/${VERSION}"

rm -rf "$OUT_DIR"
mkdir -p "$OUT_DIR"

for GOOS in linux darwin; do
	for GOARCH in amd64 arm64; do
		BIN_DIR="$OUT_DIR/sql-agent-${GOOS}-${GOARCH}"
		mkdir -p "$BIN_DIR"
		GOOS="$GOOS" GOARCH="$GOARCH" go build -o "$BIN_DIR/sql-agent" ./cmd/sql-agent
		tar -C "$OUT_DIR" -czf "$OUT_DIR/sql-agent-${GOOS}-${GOARCH}.tar.gz" "sql-agent-${GOOS}-${GOARCH}"
	done
done

bash scripts/build-skill.sh
