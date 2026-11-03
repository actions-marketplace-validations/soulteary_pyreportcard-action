package project

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func Relative(root, path string) (string, error) {
	r, e := filepath.Rel(root, path)
	if e != nil {
		return "", e
	}
	if r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path outside project: %s", path)
	}
	return filepath.ToSlash(r), nil
}
func Excluded(rel string, exclude []string) bool {
	rel = filepath.ToSlash(rel)
	for _, x := range exclude {
		x = strings.Trim(filepath.ToSlash(x), "/")
		if x == "" {
			continue
		}
		if strings.Contains(x, "/") {
			if rel == x || strings.HasPrefix(rel, x+"/") {
				return true
			}
		} else {
			for _, p := range strings.Split(rel, "/") {
				if p == x {
					return true
				}
			}
		}
	}
	return false
}
func Discover(root string, exclude []string) ([]string, error) {
	out := []string{}
	e := filepath.WalkDir(root, func(p string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		rel, e := Relative(root, p)
		if e != nil {
			return e
		}
		if rel != "." && Excluded(rel, exclude) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 || strings.HasPrefix(d.Name(), "._") {
			return nil
		}
		if !d.IsDir() && filepath.Ext(p) == ".py" {
			if strings.ContainsAny(rel, "\n\r") {
				return fmt.Errorf("unsupported newline in source path: %q", rel)
			}
			out = append(out, rel)
		}
		return nil
	})
	sort.Strings(out)
	return out, e
}
