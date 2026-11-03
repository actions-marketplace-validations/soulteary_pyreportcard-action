package project

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPathBoundaries(t *testing.T) {
	root := t.TempDir()
	if _, err := Relative(root, filepath.Dir(root)); err == nil {
		t.Fatal("accepted outside path")
	}
	if _, err := Relative(root, "relative"); err == nil {
		t.Fatal("accepted mixed absolute and relative paths")
	}
	for _, tc := range []struct {
		path    string
		exclude []string
		want    bool
	}{
		{"a/b/file", []string{"a/b"}, true}, {"a/b", []string{"a/b"}, true}, {"a/c", []string{"a/b"}, false}, {"a/b/file", []string{"b"}, true}, {"a/b", []string{"", "c"}, false}} {
		if Excluded(tc.path, tc.exclude) != tc.want {
			t.Errorf("excluded %q", tc.path)
		}
	}
	for _, name := range []string{"src/a.py", "src/z.py", "src/readme.txt", "skip.py", "vendor/a.py", "._junk.py"} {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(filepath.Join(root, "src/a.py"), filepath.Join(root, "link.py")); err != nil {
		t.Fatal(err)
	}
	got, err := Discover(root, []string{"vendor", "skip.py"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"src/a.py", "src/z.py"}) {
		t.Fatalf("discovered %v", got)
	}
	if _, err = Discover(filepath.Join(root, "missing"), nil); err == nil {
		t.Fatal("missing root accepted")
	}
}
