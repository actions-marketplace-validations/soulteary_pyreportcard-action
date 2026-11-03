#!/usr/bin/env bash
set -euo pipefail
: "${RUNNER_TEMP:?}"
: "${GITHUB_OUTPUT:?}"
tools_dir="$(mktemp -d "$RUNNER_TEMP/pyreportcard-tools.XXXXXX")"
python -m venv "$tools_dir"
"$tools_dir/bin/python" -m pip install --disable-pip-version-check -r "$GITHUB_ACTION_PATH/scripts/tool-requirements.txt"
printf 'python=%s\n' "$tools_dir/bin/python" >> "$GITHUB_OUTPUT"
