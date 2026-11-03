package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscovery(t *testing.T) {
	d := t.TempDir()
	for _, p := range []string{"src/a.py", "src/._a.py", ".venv/b.py", "vendor/c.py", "src/a.pyi"} {
		os.MkdirAll(filepath.Dir(filepath.Join(d, p)), 0755)
		os.WriteFile(filepath.Join(d, p), nil, 0644)
	}
	files, e := Discover(d, []string{".venv", "vendor"})
	if e != nil || len(files) != 1 || files[0] != "src/a.py" {
		t.Fatalf("%v %v", files, e)
	}
}
