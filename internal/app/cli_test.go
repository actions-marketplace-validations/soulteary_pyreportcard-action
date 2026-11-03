package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/soulteary/pyreportcard-action/internal/config"
	"github.com/soulteary/pyreportcard-action/internal/model"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixtureWrite(t *testing.T, root, name, content string) {
	t.Helper()
	p := filepath.Join(root, name)
	if e := os.MkdirAll(filepath.Dir(p), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(p, []byte(content), 0600); e != nil {
		t.Fatal(e)
	}
}
func appFixture(t *testing.T) (string, []string) {
	t.Helper()
	root := t.TempDir()
	fixtureWrite(t, root, "src/a.py", "value = 1\n")
	script := `#!/bin/sh
request=$(cat)
tool=$(printf '%s' "$request" | sed -n 's/.*"check":"\([^"]*\)".*/\1/p')
if [ "$ADAPTER_FAIL" = "$tool" ]; then exit 7; fi
if [ -n "$ADAPTER_SEVERITY" ]; then
 printf '{"tool":"%s","status":"ok","version":"1","total":1,"analyzed":1,"diagnostics":[{"tool":"%s","file":"src/a.py","line":1,"rule":"rule","severity":"%s","message":"finding"}]}' "$tool" "$tool" "$ADAPTER_SEVERITY"
else
 printf '{"tool":"%s","status":"ok","version":"1","total":1,"analyzed":1,"diagnostics":[]}' "$tool"
fi
`
	fixtureWrite(t, root, "adapter", script)
	if e := os.Chmod(filepath.Join(root, "adapter"), 0700); e != nil {
		t.Fatal(e)
	}
	return root, []string{"--directory", root, "--python", "./adapter", "--project-python", "./adapter", "--target-version", "py312"}
}
func runFixture(t *testing.T, args []string, want int) model.Result {
	t.Helper()
	var out, errs bytes.Buffer
	code := Run(append(args, "--json"), &out, &errs, "test")
	if code != want {
		t.Fatalf("exit=%d want=%d: %s %s", code, want, out.String(), errs.String())
	}
	var r model.Result
	if e := json.Unmarshal(out.Bytes(), &r); e != nil {
		t.Fatalf("JSON: %v: %s", e, out.String())
	}
	return r
}
func TestCLIUsageAndValidation(t *testing.T) {
	for _, tc := range []struct {
		args     []string
		code     int
		contains string
	}{
		{[]string{"--help"}, 0, "Usage"}, {[]string{"--version"}, 0, "test"}, {[]string{"--bad"}, 2, "flag provided"}, {[]string{"stray"}, 2, "positional"},
		{[]string{"--threshold", "NaN"}, 2, "threshold"}, {[]string{"--threshold", "Inf"}, 2, "threshold"}, {[]string{"--threshold", "-1"}, 2, "threshold"}, {[]string{"--threshold", "101"}, 2, "threshold"}, {[]string{"--fail-on", "bad"}, 2, "fail-on"},
		{[]string{"--directory", filepath.Join(t.TempDir(), "missing")}, 2, "no such"},
	} {
		var out, errs bytes.Buffer
		code := Run(tc.args, &out, &errs, "test")
		if code != tc.code || !strings.Contains(out.String()+errs.String(), tc.contains) {
			t.Fatalf("%v => %d %s%s", tc.args, code, &out, &errs)
		}
	}
	root, args := appFixture(t)
	for _, extra := range [][]string{{"--mode", "bad"}, {"--config", "missing"}, {"--config", "."}, {"--baseline", "missing"}} {
		var out, errs bytes.Buffer
		if Run(append(args, extra...), &out, &errs, "test") != 2 {
			t.Fatalf("accepted %v", extra)
		}
	}
	fixtureWrite(t, root, "bad.json", "not JSON")
	for _, extra := range [][]string{{"--config", "bad.json"}, {"--baseline", "bad.json"}} {
		var out, errs bytes.Buffer
		if Run(append(args, extra...), &out, &errs, "test") != 2 {
			t.Fatal(extra)
		}
	}
	empty := t.TempDir()
	var out, errs bytes.Buffer
	if Run([]string{"--directory", empty}, &out, &errs, "test") != 2 {
		t.Fatal("accepted empty project")
	}
}

type rejectWriter struct{}

func (rejectWriter) Write([]byte) (int, error) { return 0, errors.New("writer failed") }
func TestCLIPoliciesBaselinesAndOutputs(t *testing.T) {
	root, args := appFixture(t)
	r := runFixture(t, args, 0)
	if !r.Complete || r.Grade != "A+" {
		t.Fatalf("%+v", r)
	}
	r = runFixture(t, append(args, "--mode", "typed"), 0)
	if !r.Complete {
		t.Fatal(r)
	}
	t.Setenv("ADAPTER_FAIL", "mypy")
	runFixture(t, append(args, "--mode", "typed"), 2)
	t.Setenv("ADAPTER_FAIL", "")
	t.Setenv("ADAPTER_SEVERITY", "error")
	for _, policy := range []string{"errors", "findings", "new-errors"} {
		r = runFixture(t, append(args, "--fail-on", policy), 1)
		if r.NewErrors == 0 {
			t.Fatal("missing new errors")
		}
	}
	r = runFixture(t, append(args, "--fail-on", "none"), 0)
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	fixtureWrite(t, root, "baseline.json", string(data))
	r = runFixture(t, append(args, "--baseline", "baseline.json"), 0)
	if r.NewErrors != 0 {
		t.Fatal("baseline did not suppress old errors")
	}
	runFixture(t, append(args, "--fail-on", "none", "--threshold", "100"), 1)
	fixtureWrite(t, root, "baseline.json", `{}`)
	var out, errs bytes.Buffer
	if Run(append(args, "--baseline", "baseline.json"), &out, &errs, "test") != 2 {
		t.Fatal("invalid baseline accepted")
	}
	t.Setenv("ADAPTER_SEVERITY", "")
	out.Reset()
	errs.Reset()
	if Run(args, &out, &errs, "test") != 0 || !strings.Contains(out.String(), "Report Card:") {
		t.Fatalf("%s%s", &out, &errs)
	}
	if Run(append(args, "--json"), rejectWriter{}, io.Discard, "test") != 2 {
		t.Fatal("output error ignored")
	}
	fixtureWrite(t, root, "blocked", "x")
	if Run(append(args, "--output-dir", "blocked/report"), io.Discard, io.Discard, "test") != 2 {
		t.Fatal("write error ignored")
	}
}
func TestProfileFailuresAndExternalConfiguration(t *testing.T) {
	root := t.TempDir()
	c := config.Default()
	if _, err := profileID(filepath.Join(root, "missing"), c, nil); err == nil {
		t.Fatal("missing root accepted")
	}

	fixtureWrite(t, root, ".venv/ruff.toml", "ignored")
	fixtureWrite(t, root, "skip", "ignored")
	c.Exclude = append(c.Exclude, "skip")
	if err := os.Symlink(filepath.Join(root, "missing"), filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "ruff.toml")
	if err := os.WriteFile(outside, []byte("x=1"), 0600); err != nil {
		t.Fatal(err)
	}
	c.RuffConfig = outside
	if _, err := profileID(root, c, nil); err != nil {
		t.Fatal(err)
	}
	c.RuffConfig = "missing"
	if _, err := profileID(root, c, nil); err == nil {
		t.Fatal("missing config accepted")
	}

}
