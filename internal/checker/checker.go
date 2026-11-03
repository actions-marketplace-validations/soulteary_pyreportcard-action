package checker

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/soulteary/pyreportcard-action/internal/config"
	"github.com/soulteary/pyreportcard-action/internal/model"
	"os/exec"
	"strings"
	"time"
)

//go:embed runner.py
var driver string

func Run(root string, files []string, c config.Config, name string) model.Check {
	check := model.Check{Tool: name, Status: "error", Total: len(files), Diagnostics: []model.Diagnostic{}}
	request := struct {
		Root   string        `json:"root"`
		Files  []string      `json:"files"`
		Config config.Config `json:"config"`
		Check  string        `json:"check"`
	}{root, files, c, name}
	data, e := json.Marshal(request)
	if e != nil {
		check.Error = e.Error()
		return check
	}
	// The Python adapter also bounds every tool invocation. Leave room for batched calls.
	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout()*timeMultiplier(len(files)))
	defer cancel()
	cmd := exec.CommandContext(ctx, c.Python, "-I", "-c", driver)
	cmd.Dir = root
	cmd.Stdin = bytes.NewReader(data)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, e := cmd.Output()
	if e != nil {
		check.Error = fmt.Sprintf("%s: %v: %s", name, e, strings.TrimSpace(stderr.String()))
		return check
	}
	if e = json.Unmarshal(out, &check); e != nil {
		check.Status = "error"
		check.Error = "invalid adapter JSON: " + e.Error()
		return check
	}
	if check.Tool != name || check.Total != len(files) || check.Analyzed > check.Total {
		check.Status = "error"
		check.Error = "inconsistent adapter result"
	}
	return check
}

func timeMultiplier(files int) time.Duration { return time.Duration(files/128 + 3) }
