#!/usr/bin/env bash
set -euo pipefail
repo="$(cd "$(dirname "$0")/.." && pwd)"
: "${TOOL_PYTHON:?Set TOOL_PYTHON to the analyzer interpreter}"
cd "$repo"
go run ./cmd/pyreportcard-cli --directory testdata/clean --python "$TOOL_PYTHON" \
  --mode typed --output-dir "$repo/.github/pyreportcard-example" --fail-on findings
