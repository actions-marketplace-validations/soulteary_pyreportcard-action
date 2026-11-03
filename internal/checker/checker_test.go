package checker

import (
	"fmt"
	"github.com/soulteary/pyreportcard-action/internal/config"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A deterministic subprocess exercises the adapter protocol without installing analyzers.
func adapterStub(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "adapter")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	return path
}
func TestAdapterProtocol(t *testing.T) {
	for _, tc := range []struct{ name, body, want string }{
		{"success", `cat >/dev/null; printf '%s' '{"tool":"lint","status":"ok","total":1,"analyzed":1,"diagnostics":[]}'`, ""},
		{"malformed", `cat >/dev/null; printf 'not json'`, "JSON"},
		{"tool mismatch", `cat >/dev/null; printf '%s' '{"tool":"wrong","total":1}'`, "inconsistent"},
		{"total mismatch", `cat >/dev/null; printf '%s' '{"tool":"lint","total":2}'`, "inconsistent"},
		{"overcount", `cat >/dev/null; printf '%s' '{"tool":"lint","total":1,"analyzed":2}'`, "inconsistent"},
		{"failure", `echo diagnostic >&2; exit 7`, "diagnostic"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := config.Default()
			c.Python = adapterStub(t, tc.body)
			r := Run(t.TempDir(), []string{"a.py"}, c, "lint")
			if tc.want == "" {
				if r.Status != "ok" || r.Analyzed != 1 {
					t.Fatalf("%+v", r)
				}
			} else if r.Status != "error" || !strings.Contains(r.Error, tc.want) {
				t.Fatalf("%+v", r)
			}
		})
	}
	c := config.Default()
	c.Python = "/missing/adapter"
	if r := Run(t.TempDir(), nil, c, "lint"); r.Status != "error" || r.Error == "" {
		t.Fatal(r)
	}
	c.Python = adapterStub(t, "exit 0")
	c.TimeoutSeconds = -1
	if r := Run(t.TempDir(), nil, c, "lint"); r.Status != "error" {
		t.Fatal(r)
	}
}
func TestAdapterRequest(t *testing.T) {
	out := filepath.Join(t.TempDir(), "request")
	c := config.Default()
	c.Python = adapterStub(t, fmt.Sprintf("cat > %q\nprintf '%%s' '{\"tool\":\"lint\",\"status\":\"ok\",\"total\":1,\"analyzed\":1}'", out))
	root := t.TempDir()
	Run(root, []string{"a.py"}, c, "lint")
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"root":`, `"files":["a.py"]`, `"check":"lint"`, `"config":`} {
		if !strings.Contains(string(b), want) {
			t.Fatalf("missing %s in %s", want, b)
		}
	}
}

func TestBatchTimeout(t *testing.T) {
	for files, want := range map[int]int{0: 3, 127: 3, 128: 4, 256: 5} {
		if int(timeMultiplier(files)) != want {
			t.Fatal(files)
		}
	}
}
