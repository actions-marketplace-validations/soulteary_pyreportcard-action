package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInvalidConfiguration(t *testing.T) {
	for _, data := range []string{`{"mode":"full"}`, `{"unknown":true}`, `{} {}`, `{"ruff_rules":[]}`, `{"timeout_seconds":0}`, `{"target_version":"banana"}`} {
		p := filepath.Join(t.TempDir(), "config.json")
		os.WriteFile(p, []byte(data), 0644)
		if _, e := Load(p); e == nil {
			t.Errorf("accepted %s", data)
		}
	}
}
