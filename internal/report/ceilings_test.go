package report

import (
	"github.com/soulteary/pyreportcard-action/internal/model"
	"os"
	"path/filepath"
	"testing"
)

func TestSeverityCeilingsAndOrdering(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a"), []byte("one\ntwo\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		tool, rule, severity, grade string
		score                       float64
	}{{"ruff-format", "", "warning", "A", 90}, {"ruff-lint", "", "warning", "B", 85}, {"radon", "complexity", "warning", "B", 80}, {"ruff-lint", "", "error", "F", 59}, {"bandit", "", "error", "F", 39}, {"bandit", "", "warning", "B", 80}} {
		r := model.Result{Checks: []model.Check{{Tool: tc.tool, Status: "ok", Total: 1, Analyzed: 1, Diagnostics: []model.Diagnostic{{Tool: tc.tool, Rule: tc.rule, Severity: tc.severity, File: "a", Line: 1}}}}}
		Finalize(root, &r, nil)
		if r.Grade != tc.grade || r.Score == nil || *r.Score != tc.score || r.Status != "findings" {
			t.Fatalf("%+v => %+v", tc, r)
		}
	}
	ds := []model.Diagnostic{{File: "z", Rule: "x", Line: 3}, {File: "a", Rule: "z", Line: 1, Column: 1}, {File: "a", Rule: "a", Line: 1, Column: 1}, {File: "a", Rule: "a", Line: 1, Column: 2}, {File: "a", Rule: "a", Line: 2}, {File: "a", Rule: "a", Line: 1, Column: 1}}
	r := model.Result{Checks: []model.Check{{Status: "ok", Diagnostics: ds}}}
	Finalize(root, &r, nil)
	got := r.Checks[0].Diagnostics
	if len(got) != 5 || got[0].Rule != "a" || got[1].Rule != "z" || got[2].Column != 2 || got[4].File != "z" {
		t.Fatalf("wrong dedup/order: %+v", got)
	}
}
