#!/usr/bin/env bash
set -euo pipefail
repo="$(cd "$(dirname "$0")/.." && pwd)"
: "${TOOL_PYTHON:?Set TOOL_PYTHON to an interpreter with tool-requirements.txt installed}"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
cd "$repo"
go build -o "$tmp/pyreportcard-cli" ./cmd/pyreportcard-cli
"$TOOL_PYTHON" scripts/integration.py "$tmp/pyreportcard-cli" "$TOOL_PYTHON"
