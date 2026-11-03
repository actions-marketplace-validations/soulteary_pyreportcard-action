package app

import (
	"github.com/soulteary/pyreportcard-action/internal/config"
	"github.com/soulteary/pyreportcard-action/internal/model"
	"os"
	"path/filepath"
	"testing"
)

func TestProfileTracksNativeConfigAndVersions(t *testing.T) {
	root := t.TempDir()
	c := config.Default()
	checks := []model.Check{{Tool: "ruff-lint", Version: "1", PythonVersion: "3.11"}}
	first, e := profileID(root, c, checks)
	if e != nil {
		t.Fatal(e)
	}
	os.WriteFile(filepath.Join(root, "pyproject.toml"), []byte("[tool.ruff]\nline-length=100\n"), 0644)
	second, e := profileID(root, c, checks)
	if e != nil || first == second {
		t.Fatal("configuration change not tracked")
	}
	checks[0].Version = "2"
	third, _ := profileID(root, c, checks)
	if second == third {
		t.Fatal("tool upgrade not tracked")
	}
}
