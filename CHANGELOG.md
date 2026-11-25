# Changelog

## [1.0.5] - 2026-11-26

- Deliver the Ruff JSON formatting compatibility and analyzer updates listed below.
- Fix the release caller permission ceiling required by reusable CI.
- Add a manual validation-only run from a branch before creating an immutable tag.

The v1.0.4 tag is retained unchanged. Its release workflow failed at startup
and no v1.0.4 CLI distribution was published.

## [1.0.4] - 2026-11-26

Adapt Ruff formatting analysis to the pinned analyzer's structured JSON output.

- Parse `ruff format --check --output-format json` diagnostics instead of matching human-readable output lines.
- Reject incomplete or unexpected formatting responses instead of presenting a complete quality grade.
- Update pinned Ruff, Bandit and mypy versions; exercise both ends of the supported target syntax range in integration tests.
- Refresh setup Actions used by the composite Action.

Report findings and grades can change with the analyzer updates. The Action interface is unchanged. Use the exact tag or commit for reproducibility. The six platform CLI archives require the supported Python runtime and analyzer tools documented in README; they do not bundle Python.

The release workflow reruns full CI and builds checksummed archives. Marketplace enrollment/categories are separate from GitHub Release publication.


## 1.0.3

- Raise total Go statement coverage to 97.42% with deterministic unit tests for success, failure and boundary cases.
- Enforce a 95% coverage floor across all production Go packages in CI, retaining real-analyzer integration checks.

## 1.0.2

- Capitalize report badge labels and refresh the README badge example.
- Capitalize displayed analysis modes and incomplete status without changing machine-readable report fields.

## 1.0.1

- Preserve original Bandit severity alongside normalized gate severity and confidence.

## 1.0.0

- Add a Go CLI and Ubuntu Composite Action for Python quality reports.
- Integrate Ruff lint/format, Bandit, Radon and opt-in mypy.
- Isolate analyzer dependencies from the analyzed project.
- Generate JSON, Markdown and compact self-contained SVG badges.
- Add severity-based grades, configuration/version-aware baselines and occurrence matching.
- Distinguish incomplete analysis from quality violations.
- Add real-tool acceptance tests, README example generation and six-platform releases.
