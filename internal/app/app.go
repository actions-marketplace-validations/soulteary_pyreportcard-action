package app

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/soulteary/pyreportcard-action/internal/checker"
	"github.com/soulteary/pyreportcard-action/internal/config"
	"github.com/soulteary/pyreportcard-action/internal/model"
	"github.com/soulteary/pyreportcard-action/internal/project"
	"github.com/soulteary/pyreportcard-action/internal/report"
)

func Run(args []string, out, errout io.Writer, version string) int {
	fs := flag.NewFlagSet("pyreportcard-cli", flag.ContinueOnError)
	fs.SetOutput(errout)
	root := fs.String("directory", ".", "project root")
	cfg := fs.String("config", "", "JSON config relative to project root")
	mode := fs.String("mode", "", "standard or typed (overrides config)")
	python := fs.String("python", "", "tool-environment Python executable")
	projectPython := fs.String("project-python", "", "Python with project dependencies for mypy")
	target := fs.String("target-version", "", "target Python version, e.g. py311")
	output := fs.String("output-dir", ".pyreportcard", "report directory relative to project root")
	baseline := fs.String("baseline", "", "previous complete compatible report.json")
	policy := fs.String("fail-on", "new-errors", "none, errors, new-errors, or findings")
	threshold := fs.Float64("threshold", 0, "minimum score, 0 disables")
	asJSON := fs.Bool("json", false, "emit JSON; exit policy still applies")
	ver := fs.Bool("version", false, "print version")
	if e := fs.Parse(args); e != nil {
		if e == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if *ver {
		fmt.Fprintln(out, version)
		return 0
	}
	fail := func(e error) int { fmt.Fprintln(errout, e); return 2 }
	if fs.NArg() != 0 {
		return fail(fmt.Errorf("unexpected positional arguments"))
	}
	if math.IsNaN(*threshold) || math.IsInf(*threshold, 0) || *threshold < 0 || *threshold > 100 {
		return fail(fmt.Errorf("threshold must be 0..100"))
	}
	switch *policy {
	case "none", "errors", "new-errors", "findings":
	default:
		return fail(fmt.Errorf("invalid fail-on policy"))
	}
	abs, e := filepath.Abs(*root)
	if e != nil {
		return fail(e)
	}
	abs, e = filepath.EvalSymlinks(abs)
	if e != nil {
		return fail(e)
	}
	resolve := func(p string) string {
		if p == "" || filepath.IsAbs(p) {
			return p
		}
		return filepath.Join(abs, p)
	}
	c, e := config.Load(resolve(*cfg))
	if e != nil {
		return fail(e)
	}
	if *mode != "" {
		c.Mode = *mode
	}
	if *python != "" {
		c.Python = *python
	}
	if *projectPython != "" {
		c.ProjectPython = *projectPython
	}
	if *target != "" {
		c.TargetVersion = *target
	}
	if e = c.Validate(); e != nil {
		return fail(e)
	}
	for _, p := range []*string{&c.Python, &c.ProjectPython} {
		if strings.ContainsAny(*p, "/\\") {
			*p = resolve(*p)
		}
	}
	for _, p := range []*string{&c.RuffConfig, &c.BanditConfig, &c.MypyConfig} {
		*p = resolve(*p)
	}
	files, e := project.Discover(abs, c.Exclude)
	if e != nil {
		return fail(e)
	}
	if len(files) == 0 {
		return fail(fmt.Errorf("no Python .py source files found"))
	}
	r := model.Result{SchemaVersion: 1, ScoringVersion: 1, Mode: c.Mode, Files: len(files), Configuration: c, Checks: []model.Check{}}
	for _, tool := range []string{"ruff-lint", "ruff-format", "bandit", "radon"} {
		r.Checks = append(r.Checks, checker.Run(abs, files, c, tool))
	}
	if c.Mode == "typed" {
		r.Checks = append(r.Checks, checker.Run(abs, files, c, "mypy"))
	} else {
		r.Checks = append(r.Checks, model.Check{Tool: "mypy", Status: "skipped", Error: "enable typed profile to check types", Diagnostics: []model.Diagnostic{}})
	}
	profile, e := profileID(abs, c, r.Checks)
	if e != nil {
		return fail(e)
	}
	r.Profile = profile
	prior := map[string]int{}
	if *baseline != "" {
		data, e := os.ReadFile(resolve(*baseline))
		if e != nil {
			return fail(e)
		}
		var old model.Result
		if e = json.Unmarshal(data, &old); e != nil {
			return fail(e)
		}
		if old.SchemaVersion != 1 || old.ScoringVersion != 1 || old.Mode != r.Mode || old.Profile != r.Profile || !old.Complete {
			return fail(fmt.Errorf("baseline must be complete with matching profile, configuration, runtimes and tool versions"))
		}
		for _, check := range old.Checks {
			for _, d := range check.Diagnostics {
				prior[d.Fingerprint]++
			}
		}
	}
	report.Finalize(abs, &r, prior)
	if e = report.Write(resolve(*output), r); e != nil {
		return fail(e)
	}
	if *asJSON {
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		if e = encoder.Encode(r); e != nil {
			return fail(e)
		}
	} else {
		fmt.Fprintf(out, "Python Report Card: %s (%s, %s)\nReports: %s\n", r.Grade, r.Mode, r.Status, resolve(*output))
	}
	if !r.Complete {
		return 2
	}
	violated := *threshold > 0 && r.Score != nil && *r.Score < *threshold
	for _, check := range r.Checks {
		for _, d := range check.Diagnostics {
			if *policy == "findings" || *policy == "errors" && d.Severity == "error" || *policy == "new-errors" && d.Severity == "error" && d.New {
				violated = true
			}
		}
	}
	if violated {
		return 1
	}
	return 0
}

func profileID(root string, c config.Config, checks []model.Check) (string, error) {
	hash := sha256.New()
	paths := []string{}
	// Include native configuration (including nested Ruff configs); ignore only
	// directories excluded from the report's source manifest.
	e := filepath.WalkDir(root, func(path string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		rel, _ := project.Relative(root, path)
		if rel != "." && project.Excluded(rel, c.Exclude) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		switch d.Name() {
		case "pyproject.toml", "ruff.toml", ".ruff.toml", "mypy.ini", ".mypy.ini", "setup.cfg", ".bandit":
			paths = append(paths, path)
		}
		return nil
	})
	if e != nil {
		return "", e
	}
	paths = append(paths, c.RuffConfig, c.BanditConfig, c.MypyConfig)
	sort.Strings(paths)
	for _, p := range paths {
		if p == "" {
			continue
		}
		data, e := os.ReadFile(p)
		if e != nil {
			return "", e
		}
		rel, e := project.Relative(root, p)
		if e != nil {
			rel = filepath.Base(p)
		}
		fmt.Fprintf(hash, "%s\x00%s\x00", rel, data)
	}
	c.Python = ""
	c.ProjectPython = ""
	c.RuffConfig = filepath.Base(c.RuffConfig)
	c.BanditConfig = filepath.Base(c.BanditConfig)
	c.MypyConfig = filepath.Base(c.MypyConfig)
	data, _ := json.Marshal(c)
	hash.Write(data)
	for _, check := range checks {
		fmt.Fprintf(hash, "%s\x00%s\x00%s\x00", check.Tool, check.Version, check.PythonVersion)
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
