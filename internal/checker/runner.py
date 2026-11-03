"""Embedded tool adapter. Does not import or execute the analyzed project."""
import ast
import importlib.metadata
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tokenize

request = json.load(sys.stdin)
config = request["config"]
root = Path(request["root"])
files = request["files"]
name = request["check"]
result = {"tool": name, "version": "", "status": "passed", "analyzed": 0,
          "total": len(files), "diagnostics": [], "metrics": {},
          "python_version": ".".join(map(str, sys.version_info[:2]))}


def finding(rule, severity, filename, line, column, message, confidence="", tool_severity=""):
    path = Path(filename)
    if not path.is_absolute():
        path = root / path
    try:
        relative = path.resolve().relative_to(root).as_posix()
    except ValueError:
        return
    if relative not in files:
        return
    result["diagnostics"].append({"tool": name, "rule": rule, "severity": severity,
        "confidence": confidence, "tool_severity": tool_severity, "file": relative, "line": line, "column": column,
        "message": message})


def run(module, args, allowed=(0, 1)):
    env = {k: v for k, v in os.environ.items() if not k.startswith("RUFF_")}
    process = subprocess.run([sys.executable, "-I", "-m", module, *args],
        cwd=root, text=True, encoding="utf-8", errors="replace", capture_output=True,
        timeout=config["timeout_seconds"], env=env)
    if process.returncode not in allowed:
        raise RuntimeError(f"{module} exited {process.returncode}: {process.stderr or process.stdout}")
    return process


def batches(items, size=128):
    for i in range(0, len(items), size):
        yield items[i:i + size]


def ruff_common():
    # The Report Card manifest controls selection; native per-file ignores remain.
    args = ["--target-version", config["target_version"], "--no-respect-gitignore",
            "--config", "force-exclude=false", "--config", "exclude=[]",
            "--config", "extend-exclude=[]"]
    if config["ruff_config"]:
        args += ["--config", config["ruff_config"]]
    return args


def lint():
    error_rules = {"F821", "F822", "F823", "F831", "F701", "F702", "F704", "F706", "F707", "F722", "F901", "E999", "invalid-syntax"}
    for batch in batches(files):
        proc = run("ruff", ["check", "--no-fix", "--no-cache", "--output-format", "json",
            "--select", ",".join(config["ruff_rules"]), *ruff_common(), *[str(root / p) for p in batch]])
        data = json.loads(proc.stdout)
        if not isinstance(data, list) or (proc.returncode == 1 and not data):
            raise ValueError("invalid Ruff diagnostic response")
        for item in data:
            location = item["location"]
            rule = item.get("code") or "invalid-syntax"
            finding(rule, "error" if rule in error_rules else "warning", item["filename"],
                    location["row"], location["column"], item["message"])
            if rule in {"invalid-syntax", "E999"}:
                result["status"] = "error"
                result["error"] = "Python syntax prevents complete analysis"
        result["analyzed"] += len(batch)


def formatting():
    for batch in batches(files):
        proc = run("ruff", ["format", "--check", "--no-cache", *ruff_common(), *[str(root / p) for p in batch]])
        changed = [line.removeprefix("Would reformat: ") for line in proc.stdout.splitlines()
                   if line.startswith("Would reformat: ")]
        if proc.returncode == 1 and not changed:
            raise ValueError("Ruff format failed without a file list")
        for path in changed:
            finding("format", "warning", path, 1, 1, "File differs from configured Ruff format")
        result["analyzed"] += len(batch)


def security():
    for batch in batches(files):
        args = ["-q", "-f", "json"]
        if config["bandit_config"]:
            args += ["-c", config["bandit_config"]]
        proc = run("bandit", [*args, *[str(root / p) for p in batch]])
        data = json.loads(proc.stdout)
        if not isinstance(data.get("results"), list) or "metrics" not in data:
            raise ValueError("invalid Bandit response")
        covered = set()
        for path in data["metrics"]:
            if path == "_totals":
                continue
            try:
                covered.add((root / path).resolve().relative_to(root).as_posix())
            except ValueError:
                pass
        if not set(batch).issubset(covered):
            raise ValueError("Bandit skipped selected files; align report exclusions with Bandit configuration")
        if data.get("errors"):
            raise ValueError(f"Bandit could not analyze files: {data['errors']}")
        for item in data["results"]:
            confidence = item["issue_confidence"].lower()
            severity = "error" if item["issue_severity"] == "HIGH" and confidence == "high" else "warning"
            finding(item["test_id"], severity, item["filename"], item["line_number"],
                    item.get("col_offset", 0) + 1, item["issue_text"], confidence, item["issue_severity"].lower())
        result["analyzed"] += len(batch)


def complexity():
    from radon.complexity import cc_visit
    count, maximum, over = 0, 0, 0
    for filename in files:
        with tokenize.open(root / filename) as stream:
            source = stream.read()
        target_minor = int(config["target_version"][3:])
        ast.parse(source, filename=filename, feature_version=(3, target_minor))
        seen = set()
        def visit(block):
            nonlocal count, maximum, over
            if hasattr(block, "methods"):
                for method in block.methods:
                    visit(method)
                return
            key = (block.lineno, block.name)
            if key in seen:
                return
            seen.add(key)
            count += 1
            maximum = max(maximum, block.complexity)
            if block.complexity > config["complexity_limit"]:
                over += 1
                finding("complexity", "warning", filename, block.lineno, block.col_offset + 1,
                        f"{block.name} has cyclomatic complexity {block.complexity}; limit is {config['complexity_limit']}")
            for closure in getattr(block, "closures", []):
                visit(closure)
        for block in cc_visit(source):
            visit(block)
        result["analyzed"] += 1
    result["metrics"] = {"functions": count, "max_complexity": maximum, "over_limit": over}


def typing():
    args = ["--show-column-numbers", "--show-error-codes", "--no-error-summary", "--no-pretty",
            "--no-color-output", "--check-untyped-defs", "--disallow-untyped-defs",
            "--follow-imports=silent", "--cache-dir", os.devnull]
    if config["mypy_config"]:
        args += ["--config-file", config["mypy_config"]]
    if config["project_python"]:
        args += ["--python-executable", config["project_python"]]
    version = config["target_version"].removeprefix("py")
    args += ["--python-version", f"{version[0]}.{version[1:]}"]
    proc = run("mypy", [*args, *[str(root / p) for p in files]])
    pattern = re.compile(r"^(.+?):(\d+):(\d+): (error|warning|note): (.*?)(?:  \[([^\]]+)\])?$")
    parsed_errors = 0
    for line in proc.stdout.splitlines():
        match = pattern.match(line)
        if not match:
            if line.strip():
                raise ValueError(f"unrecognized mypy output: {line}")
            continue
        filename, row, column, severity, message, rule = match.groups()
        if severity == "note":
            continue
        parsed_errors += 1
        finding(rule or "mypy", severity, filename, int(row), int(column), message)
        if rule in {"import-not-found", "import-untyped"}:
            result["status"] = "error"
            result["error"] = "Missing dependencies or type stubs prevent complete type analysis"
    if proc.returncode == 1 and not parsed_errors:
        raise ValueError("mypy failed without diagnostics")
    result["analyzed"] = len(files)
    total, annotated = 0, 0
    for filename in files:
        with tokenize.open(root / filename) as stream:
            source = stream.read()
        for node in ast.walk(ast.parse(source)):
            if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef)):
                total += 1
                if node.returns is not None:
                    annotated += 1
    result["metrics"] = {"functions": total, "functions_with_return_annotations": annotated}


try:
    package = {"ruff-lint": "ruff", "ruff-format": "ruff", "bandit": "bandit",
               "radon": "radon", "mypy": "mypy"}[name]
    result["version"] = importlib.metadata.version(package)
    {"ruff-lint": lint, "ruff-format": formatting, "bandit": security,
     "radon": complexity, "mypy": typing}[name]()
    if result["status"] != "error" and result["diagnostics"]:
        result["status"] = "findings"
except Exception as exc:
    result["status"] = "error"
    result["error"] = str(exc)
# stdout is a protocol; no tool logs or source execution belongs on it.
json.dump(result, sys.stdout)
