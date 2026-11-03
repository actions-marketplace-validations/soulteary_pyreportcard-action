# Security

Analyze untrusted pull requests with read-only credentials. Never use
pull_request_target to execute untrusted source with privileged tokens.

The default analyzers parse source; the tool does not install the project, import
its modules, execute setup.py or run tests. Analyzer packages and configured mypy
plugins execute code and must be trusted. Prepare project environments in isolated
jobs. Baselines used to waive findings must come from a trusted base revision.

Report vulnerabilities privately through the repository's GitHub security
reporting feature when enabled. Do not publish secrets or working exploit details
in a public issue.
