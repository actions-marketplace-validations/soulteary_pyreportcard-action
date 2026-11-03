# Python Report Card

[![Python Report Card example](.github/pyreportcard-example/badge.svg)](.github/pyreportcard-example/report.md) [![Go Report Card](.github/goreportcard.svg)](.github/goreportcard-report.md) [![CI](https://github.com/soulteary/pyreportcard-action/actions/workflows/ci.yml/badge.svg)](https://github.com/soulteary/pyreportcard-action/actions/workflows/ci.yml)

Generate local Python quality reports with **Ruff, Bandit, Radon and optional
mypy**. A Go CLI and GitHub Action produce JSON, Markdown and a self-contained SVG
badge, with no hosted reporting service. The badge above shows the actual result
for the bundled [typed example](testdata/clean), not a rating of this Go tool.

## GitHub Action

```yaml
name: Python quality
on: [push, pull_request, workflow_dispatch]
permissions:
  contents: read
jobs:
  report:
    runs-on: ubuntu-24.04
    steps:
      - uses: actions/checkout@v6
      - uses: soulteary/pyreportcard-action@v1
        with:
          mode: standard
      - uses: actions/upload-artifact@v4
        if: always()
        with:
          name: pyreportcard
          path: .pyreportcard/
          include-hidden-files: true
```

The Action compiles its small Go CLI from the selected Action revision, creates a
separate virtual environment for pinned analyzer versions, runs checks and writes
a GitHub Step Summary. It does not install or execute your project. The default
permissions are read-only; report commits are opt-in and run only on branch push
events. Pin an exact version or commit SHA for reproducibility.

For type checking, prepare project dependencies first and pass the interpreter:

```yaml
- uses: actions/setup-python@v5
  with:
    python-version: '3.11'
- run: |
    python -m venv .venv
    .venv/bin/python -m pip install -r requirements.txt
- uses: soulteary/pyreportcard-action@v1
  with:
    mode: typed
    project_python: .venv/bin/python
```

The **tool environment** and **project environment** serve different purposes.
The former contains analyzers, the latter supplies installed packages and stubs
to mypy. A custom mypy plugin must be available in the tool environment; use the
CLI with an explicitly prepared environment for that case.

### Inputs and outputs

- `directory`: source root relative to the workspace; default `.`.
- `mode`: `standard` (default) or `typed`.
- `python_version`: analyzer Python version; default `3.11`, requires 3.11+.
- `target_version`: source language target; default `py311`.
- `project_python`: optional project interpreter for mypy.
- `config`: optional strict JSON config relative to `directory`.
- `output_dir`: default `.pyreportcard`, relative to `directory`.
- `baseline`: previous complete compatible `report.json`.
- `fail_on`: `none`, `errors`, `new-errors` (default), or `findings`.
- `threshold`: minimum score; default `0` disables score gating.
- `commit`: default `false`; requires `contents: write` if enabled.

Outputs: `status`, `grade`, `score`, `analysis_complete`, `new_errors`, `report`,
`json`, `badge`. File outputs are absolute paths. Incomplete analysis has an empty
score output. Invalid invocation can fail before a report is available.

Mode and target-version Action inputs override the JSON configuration. See
[the complete example](examples/pyreportcard.yml).

## CLI

Build with Go 1.23+, or download a release binary. Install the validated tools in
a virtual environment, not into your global Python:

```sh
python3 -m venv .venv-tools
.venv-tools/bin/python -m pip install -r scripts/tool-requirements.txt
go install github.com/soulteary/pyreportcard-action/cmd/pyreportcard-cli@latest

pyreportcard-cli --python .venv-tools/bin/python
pyreportcard-cli --python .venv-tools/bin/python --mode typed
pyreportcard-cli --python .venv-tools/bin/python --json
```

The requirements file is in this repository; when using a downloaded CLI without
a checkout, install `ruff==0.14.3 bandit==1.8.6 radon==6.0.1 mypy==1.18.2` into your
tool environment. These are tested versions, not claims about the latest releases.
Transitive dependencies are resolved by pip; full dependency locking is the
caller's responsibility. Reports record analyzer and Python runtime versions.

`--python` paths containing a slash are relative to `--directory`, not the caller's
working directory. Bare executable names resolve through PATH. On Windows use
the virtual environment's `Scripts/python.exe`. Automatic Action installation is
currently supported on Ubuntu; CLI binaries are built for Linux/macOS/Windows,
amd64/arm64, while analyzer availability depends on platform and Python version.

Exit codes: **0** complete and policy passed; **1** complete with policy violations;
**2** incomplete analysis, invalid input or execution error. `--json` does not
change exit semantics. `--fail-on none` disables finding gates, but not execution
failures or an explicit nonzero score threshold.

## Checks and configuration

`standard` runs:

- **Ruff lint:** explicit `E4,E7,E9,F,B` rules, with no automatic fixes.
- **Ruff format:** checks only, honoring native formatting configuration.
- **Bandit:** source security checks with severity and confidence preserved.
- **Radon:** function complexity with a default limit of 10.

`typed` adds **mypy**, checking untyped bodies and requiring function annotations.
This does not prove complete type safety: explicit `Any`, ignores and native
configuration can reduce coverage. The report includes function and return-
annotation counts as descriptive metrics, not a type-coverage percentage.

Start from [.pyreportcard.example.json](.pyreportcard.example.json). Unknown JSON
keys and invalid values fail early. Native Ruff configurations are discovered;
explicit `ruff_config`, `bandit_config`, and `mypy_config` paths are also supported.
Bandit configuration is opt-in; it does not automatically read `pyproject.toml`.
Mypy discovers its native configuration unless an explicit path is supplied.

Only `.py` source files are selected in v1. Notebooks and standalone `.pyi` report
scanning are not supported; mypy can still use stubs through imports. Exclusions
are literal path components or project-relative prefixes, not globs. Virtual
environments, caches, generated build outputs and vendored code are excluded by
default. Symlinked source files are not traversed.

The report manifest controls file selection. Ruff's own file exclusions and
`.gitignore` are overridden for explicitly selected files; per-file rule ignores
remain effective. Align the report exclusions with any Bandit exclusions.

A failed/missing tool, malformed output, unsupported syntax, skipped required
files, or missing mypy dependencies/stubs produces **incomplete** with no numeric
score. Source syntax is parsed, never imported. User-selected mypy plugins and
any explicitly executed dependency-install steps remain trusted code.

## Grades and baselines

Scoring v1 uses the lowest applicable **severity ceiling**, not a percentage of
clean files. Adding clean files cannot dilute an existing problem:

- No findings: 100 / A+.
- Formatting only: at most 90 / A.
- Other lint warnings: at most 85 / B.
- Complexity or security warnings: at most 80 / B.
- Error diagnostics: at most 59 / F.
- Bandit HIGH severity with HIGH confidence: at most 39 / F.

This intentionally coarse score summarizes the worst category. The full report
provides counts, rule IDs, positions, confidence and complexity metrics. Native
tool suppressions are respected; grades should not rank repositories with
different policies. Bandit records its original severity in `tool_severity`, separately from the
normalized gate severity and confidence. Bandit warnings retain lower-confidence findings without
presenting them as confirmed vulnerabilities. Ruff security rules are off by
default to avoid duplicate checks.

```sh
pyreportcard-cli --python .venv-tools/bin/python \
  --baseline baseline.json --fail-on new-errors
```

A baseline must match the profile, configuration contents, tool versions and
analyzer Python minor version. Fingerprints include the rule, severity,
confidence, relative file, message and normalized source line. Occurrence counts
ensure a second identical finding is still new. Line shifts alone are tolerated;
source edits, renames or message changes may produce new findings. Baselines never
change the absolute grade. They must come from a trusted base revision.

## Development and releases

```sh
make test
TOOL_PYTHON=/absolute/path/to/tool/python make integration
TOOL_PYTHON=/absolute/path/to/tool/python scripts/update-badge-example.sh
```

CI validates real clean, defective, security, formatting, complexity, missing-
dependency and syntax-error fixtures, plus the local Action. It also verifies that
analysis does not execute `setup.py` and that baselines detect repeated findings.

Pushing `vX.Y.Z` runs the full CI, builds six CLI archives and SHA-256 checksums,
and prepares a draft release. Publish the draft and select Marketplace listing
in GitHub's release form. Promote the stable `v1` tag after validation. Exact
release tags are immutable. See [.github/MARKETPLACE.md](.github/MARKETPLACE.md).

Tests/coverage, dependency-vulnerability audits and PyPI package metadata grading
are intentionally outside v1. They can later be added as separate report inputs.
The CLI never installs the analyzed project or runs its test suite automatically.

## License

Apache-2.0; see [LICENSE](LICENSE). Badge styling and the report shell are adapted
from [Go Report Card](https://github.com/soulteary/goreportcard-action) and
[C++ Report Card](https://github.com/soulteary/cppreportcard-action).

## Unit-test coverage

Run `scripts/test-coverage.sh` to execute the Go unit tests with the race detector
and coverage instrumentation across all production Go packages, including the
CLI. CI requires at least **95% total statement coverage**. The gate uses exact
statement counts from `coverage.out`, not a rounded percentage, and does not
exclude production files. Analyzer protocol tests use deterministic local
processes; real-analyzer integration tests remain part of CI.

## Go implementation quality

The Go Report Card badge grades this Action's own Go implementation. CI generates
its [report](.github/goreportcard-report.md) with
[soulteary/goreportcard-action](https://github.com/soulteary/goreportcard-action).
Default-branch runs update the committed badge and report; pull requests expose
results as CI artifacts. This is separate from the language report generated
for projects using this Action.
