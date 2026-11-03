# Contributing

Use Go 1.23+ and Python 3.11+. Create a virtual environment and install
`scripts/tool-requirements.txt`, then run `make test` and
`TOOL_PYTHON=/absolute/path/to/python make integration`.

Keep tool invocation, result normalization, scoring and report rendering separate.
Add regression tests for any failure that could otherwise create a false-green
badge. Never import project modules as part of static analysis. Schema and scoring
changes require explicit version bumps and migration documentation.
