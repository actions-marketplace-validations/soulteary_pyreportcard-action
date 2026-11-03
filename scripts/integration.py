"""Real-tool CLI acceptance tests; does not mutate the checked-out fixture."""
import json
import os
from pathlib import Path
import subprocess
import shutil
import sys
import tempfile

cli = str(Path(sys.argv[1]).resolve())
python = os.path.abspath(shutil.which(sys.argv[2]) or sys.argv[2])
clean = '"""A typed fixture."""\n\n\ndef add(x: int, y: int) -> int:\n    return x + y\n'
with tempfile.TemporaryDirectory() as directory:
    root = Path(directory)
    source = root / "example.py"
    source.write_text(clean)

    def check(code=0, extra=(), mode="standard"):
        result = subprocess.run([cli, "--directory", directory, "--python", python,
            "--mode", mode, "--json", *extra], capture_output=True, text=True)
        assert result.returncode == code, (result.returncode, result.stdout, result.stderr)
        report = json.loads(result.stdout)
        assert all((root / ".pyreportcard" / f).exists()
                   for f in ("report.json", "report.md", "badge.svg"))
        return report

    report = check(mode="typed")
    assert report["grade"] == "A+" and report["analysis_complete"], report
    assert len(report["checks"]) == 5
    source.write_text('"""A defect."""\n\nvalue = missing\n')
    report = check(1)
    assert report["new_errors"] == 1 and report["score"] == 59, report
    (root / "baseline.json").write_text(json.dumps(report))
    check(extra=("--baseline", "baseline.json"))
    source.write_text('"""A defect."""\n\nvalue = missing\nvalue = missing\n')
    report = check(1, extra=("--baseline", "baseline.json"))
    assert report["new_errors"] == 1, report
    source.write_text('"""Syntax error."""\n\ndef broken(:\n')
    report = check(2, extra=("--fail-on", "none"))
    assert not report["analysis_complete"] and report["score"] is None
    source.write_text('"""Missing dependencies."""\n\nimport missing_dependency_for_reportcard\n')
    report = check(2, mode="typed", extra=("--fail-on", "none"))
    assert not report["analysis_complete"]
    source.write_text('"""Known type defect."""\n\nvalue: int = "wrong"\n')
    report = check(1, mode="typed")
    assert any(d["tool"] == "mypy" for c in report["checks"] for d in c["diagnostics"])
    source.write_text('"""Formatting fixture."""\n\nvalue=1\n')
    report = check()
    assert report["score"] == 90, report
    assert source.read_text().endswith("value=1\n")
    source.write_text('"""Security fixture."""\n\nimport subprocess\n\ndef run(command: str) -> None:\n    subprocess.run(command, shell=True)\n')
    report = check(1)
    assert any(d["tool"] == "bandit" and d["severity"] == "error"
               for c in report["checks"] for d in c["diagnostics"]), report
    source.write_text('"""Complexity fixture."""\n\ndef work(x: int) -> int:\n' +
                      ''.join(f'    if x == {i}:\n        return {i}\n' for i in range(12)) + '    return -1\n')
    report = check()
    assert next(c for c in report["checks"] if c["tool"] == "radon")["metrics"]["over_limit"] == 1
    source.write_text(clean)
    (root / "setup.py").write_text('from pathlib import Path\n\nPath("executed").write_text("bad")\n')
    check()
    assert not (root / "executed").exists(), "Analyzer executed project code"
    (root / "setup.py").unlink()
    (root / "pyproject.toml").write_text('[tool.ruff]\nline-length = 88\n')
    assert check()["grade"] == "A+"
print("All real-analyzer acceptance tests passed.")
