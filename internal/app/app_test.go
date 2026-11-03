package app

import (
	"bytes"
	"encoding/json"
	"github.com/soulteary/pyreportcard-action/internal/model"
	"os"
	"path/filepath"
	"testing"
)

func TestUnavailablePython(t *testing.T) {
	d := t.TempDir()
	os.WriteFile(filepath.Join(d, "a.py"), []byte("x = 1\n"), 0644)
	var out, err bytes.Buffer
	code := Run([]string{"--directory", d, "--python", "/not-installed/python", "--json", "--fail-on", "none"}, &out, &err, "test")
	if code != 2 {
		t.Fatalf("%d %s", code, err.String())
	}
	var r model.Result
	if e := json.Unmarshal(out.Bytes(), &r); e != nil {
		t.Fatal(e)
	}
	if r.Complete || r.Score != nil {
		t.Fatal(r)
	}
	if _, e := os.Stat(filepath.Join(d, ".pyreportcard/badge.svg")); e != nil {
		t.Fatal(e)
	}
}
