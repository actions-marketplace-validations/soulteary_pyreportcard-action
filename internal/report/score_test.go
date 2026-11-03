package report

import (
	"github.com/soulteary/pyreportcard-action/internal/model"
	"os"
	"path/filepath"
	"testing"
)

func TestIncomplete(t *testing.T) {
	r := model.Result{Checks: []model.Check{{Status: "error"}}}
	Finalize(t.TempDir(), &r, nil)
	if r.Complete || r.Score != nil || r.Grade != "incomplete" {
		t.Fatal(r)
	}
}
func TestDuplicateOccurrenceBaseline(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "a.py"), []byte("x = missing\nx = missing\n"), 0644)
	d := model.Diagnostic{Tool: "ruff-lint", Rule: "F821", Severity: "error", File: "a.py", Line: 1, Message: "Undefined name missing"}
	second := d
	second.Line = 2
	r := model.Result{Files: 1000, Checks: []model.Check{{Tool: "ruff-lint", Status: "findings", Analyzed: 1, Total: 1, Diagnostics: []model.Diagnostic{d}}}}
	Finalize(root, &r, nil)
	fp := r.Checks[0].Diagnostics[0].Fingerprint
	r.Checks[0].Diagnostics = []model.Diagnostic{d, second}
	Finalize(root, &r, map[string]int{fp: 1})
	if r.NewErrors != 1 || *r.Score != 59 {
		t.Fatal(r)
	}
}
