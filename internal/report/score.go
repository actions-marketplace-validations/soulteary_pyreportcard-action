package report

import (
	"crypto/sha256"
	"fmt"
	"github.com/soulteary/pyreportcard-action/internal/model"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Finalize uses severity ceilings, not clean-file density. Adding files cannot
// dilute an existing defect. Baselines match occurrence counts with source context.
func Finalize(root string, r *model.Result, prior map[string]int) {
	r.Complete = true
	r.NewErrors = 0
	r.Score = nil
	r.Grade = "incomplete"
	r.Status = "incomplete"
	for i := range r.Checks {
		c := &r.Checks[i]
		if c.Status == "error" || c.Status != "skipped" && c.Analyzed != c.Total {
			r.Complete = false
		}
		sort.Slice(c.Diagnostics, func(i, j int) bool {
			a, b := c.Diagnostics[i], c.Diagnostics[j]
			if a.File != b.File {
				return a.File < b.File
			}
			if a.Line != b.Line {
				return a.Line < b.Line
			}
			if a.Column != b.Column {
				return a.Column < b.Column
			}
			return a.Rule < b.Rule
		})
		seen := map[string]bool{}
		out := []model.Diagnostic{}
		for _, d := range c.Diagnostics {
			key := fmt.Sprintf("%s:%s:%d:%d:%s", d.Tool, d.File, d.Line, d.Column, d.Rule)
			if seen[key] {
				continue
			}
			seen[key] = true
			context := ""
			if data, e := os.ReadFile(filepath.Join(root, d.File)); e == nil {
				lines := strings.Split(string(data), "\n")
				if d.Line > 0 && d.Line <= len(lines) {
					context = strings.Join(strings.Fields(lines[d.Line-1]), " ")
				}
			}
			sum := sha256.Sum256([]byte(strings.Join([]string{d.Tool, d.Rule, d.Severity, d.Confidence, d.File, d.Message, context}, "\x00")))
			d.Fingerprint = fmt.Sprintf("%x", sum[:])
			d.New = true
			if prior[d.Fingerprint] > 0 {
				d.New = false
				prior[d.Fingerprint]--
			}
			if d.New && d.Severity == "error" {
				r.NewErrors++
			}
			out = append(out, d)
		}
		c.Diagnostics = out
	}
	if !r.Complete {
		return
	}
	score := 100.0
	r.Status = "passed"
	for _, c := range r.Checks {
		for _, d := range c.Diagnostics {
			r.Status = "findings"
			ceiling := 85.0
			if c.Tool == "ruff-format" {
				ceiling = 90
			}
			if c.Tool == "radon" || c.Tool == "bandit" {
				ceiling = 80
			}
			if d.Severity == "error" {
				ceiling = 59
			}
			if c.Tool == "bandit" && d.Severity == "error" {
				ceiling = 39
			}
			if ceiling < score {
				score = ceiling
			}
		}
	}
	r.Score = &score
	switch {
	case score >= 95:
		r.Grade = "A+"
	case score >= 90:
		r.Grade = "A"
	case score >= 80:
		r.Grade = "B"
	case score >= 70:
		r.Grade = "C"
	case score >= 60:
		r.Grade = "D"
	default:
		r.Grade = "F"
	}
}
