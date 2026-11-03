package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

type Config struct {
	Mode            string   `json:"mode"`
	Python          string   `json:"python"`
	ProjectPython   string   `json:"project_python"`
	TargetVersion   string   `json:"target_version"`
	Exclude         []string `json:"exclude"`
	RuffRules       []string `json:"ruff_rules"`
	RuffConfig      string   `json:"ruff_config"`
	BanditConfig    string   `json:"bandit_config"`
	MypyConfig      string   `json:"mypy_config"`
	ComplexityLimit int      `json:"complexity_limit"`
	TimeoutSeconds  int      `json:"timeout_seconds"`
}

func Default() Config {
	return Config{Mode: "standard", Python: "python3", TargetVersion: "py311", Exclude: []string{".git", ".venv", "venv", "__pycache__", ".mypy_cache", ".ruff_cache", "build", "dist", "vendor", "third_party"}, RuffRules: []string{"E4", "E7", "E9", "F", "B"}, ComplexityLimit: 10, TimeoutSeconds: 120}
}
func (c Config) Validate() error {
	if c.Mode != "standard" && c.Mode != "typed" {
		return fmt.Errorf("mode must be standard or typed")
	}
	if c.Python == "" || c.ComplexityLimit < 1 || c.TimeoutSeconds < 1 {
		return fmt.Errorf("python, positive complexity_limit and timeout_seconds are required")
	}
	switch c.TargetVersion {
	case "py39", "py310", "py311", "py312", "py313", "py314":
	default:
		return fmt.Errorf("unsupported target_version")
	}
	if len(c.RuffRules) == 0 {
		return fmt.Errorf("ruff_rules cannot be empty")
	}
	return nil
}
func Load(path string) (Config, error) {
	c := Default()
	if path != "" {
		f, e := os.Open(path)
		if e != nil {
			return c, e
		}
		defer f.Close()
		d := json.NewDecoder(f)
		d.DisallowUnknownFields()
		if e = d.Decode(&c); e != nil {
			return c, e
		}
		if e = d.Decode(new(any)); e != io.EOF {
			return c, fmt.Errorf("expected exactly one configuration object")
		}
	}
	return c, c.Validate()
}
func (c Config) Timeout() time.Duration { return time.Duration(c.TimeoutSeconds) * time.Second }
