# Changelog

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
