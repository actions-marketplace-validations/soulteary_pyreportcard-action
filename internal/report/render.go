package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/soulteary/pyreportcard-action/internal/model"
)

func escape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "|", "&#124;", "\n", " ", "\r", " ", "`", "&#96;")
	return r.Replace(s)
}
func Markdown(r model.Result) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Python Report Card\n\n**Grade: %s** · Mode: %s · Status: %s\n\n", r.Grade, r.Mode, r.Status)
	if r.Score != nil {
		fmt.Fprintf(&b, "Score: %.1f / 100\n\n", *r.Score)
	}
	fmt.Fprintf(&b, "Files: %d · New errors: %d · Scoring version: %d\n\n", r.Files, r.NewErrors, r.ScoringVersion)
	if r.Mode == "standard" {
		b.WriteString("Standard analysis does not include type checking.\n\n")
	}
	if !r.Complete {
		b.WriteString("**Analysis incomplete. No quality grade is available.**\n\n")
	}
	b.WriteString("| Tool | Version | Status | Analyzed units |\n| --- | --- | --- | --- |\n")
	for _, c := range r.Checks {
		fmt.Fprintf(&b, "| %s | %s | %s | %d / %d |\n", escape(c.Tool), escape(c.Version), c.Status, c.Analyzed, c.Total)
	}
	for _, c := range r.Checks {
		fmt.Fprintf(&b, "\n## %s\n\n", c.Tool)
		if c.Error != "" {
			fmt.Fprintf(&b, "%s\n\n", escape(c.Error))
		}
		if len(c.Metrics) > 0 {
			keys := make([]string, 0, len(c.Metrics))
			for key := range c.Metrics {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				fmt.Fprintf(&b, "- %s: %.0f\n", escape(key), c.Metrics[key])
			}
			b.WriteString("\n")
		}
		if len(c.Diagnostics) == 0 {
			continue
		}
		b.WriteString("| File | Line | Severity | Confidence | Rule | New | Message |\n| --- | --- | --- | --- | --- | --- | --- |\n")
		for _, d := range c.Diagnostics {
			fmt.Fprintf(&b, "| %s | %d | %s | %s | %s | %t | %s |\n", escape(d.File), d.Line, diagnosticSeverity(d), escape(d.Confidence), escape(d.Rule), d.New, escape(d.Message))
		}
	}
	return b.String()
}
func Write(dir string, r model.Result) error {
	if e := os.MkdirAll(dir, 0755); e != nil {
		return e
	}
	data, e := json.MarshalIndent(r, "", "  ")
	if e != nil {
		return e
	}
	for _, f := range []struct {
		name string
		data []byte
	}{{"report.json", append(data, '\n')}, {"report.md", []byte(Markdown(r))}, {"badge.svg", []byte(SVG(r))}} {
		if e = os.WriteFile(filepath.Join(dir, f.name), f.data, 0644); e != nil {
			return e
		}
	}
	return nil
}

func diagnosticSeverity(d model.Diagnostic) string {
	if d.ToolSeverity != "" {
		return escape(d.Severity + " (" + d.ToolSeverity + ")")
	}
	return escape(d.Severity)
}
