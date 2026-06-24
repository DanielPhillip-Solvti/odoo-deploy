package actions

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type ResetBranchParams struct {
	Branch       string `json:"branch"`
	IsProduction bool   `json:"is_production"`
}

func (p ResetBranchParams) Validate() error {
	if strings.TrimSpace(p.Branch) == "" {
		return errors.New("branch is required")
	}
	return nil
}

func ResetBranch(p ResetBranchParams) (string, error) {
	baseDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getwd: %w", err)
	}
	addonsDir := filepath.Join(baseDir, "envs", p.Branch, "addons")
	cmd := exec.Command("git", "-C", addonsDir,
		"fetch")
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git fetch failed: %w\n%s", err, out)
	}
	cmd = exec.Command("git", "-C", addonsDir,
		"reset", "--hard", "origin/"+p.Branch)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git reset failed: %w\n%s", err, out)
	}
	cmd = exec.Command("git", "-C", addonsDir, "clean", "-fd")
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git clean failed: %w\n%s", err, out)
	}
	return "Repository reset to branch " + p.Branch, nil
}
