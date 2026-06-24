package actions

import (
	"agent/helpers"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type UndeployParams struct {
	Branch string `json:"branch"`
}

func (p UndeployParams) Validate() error {
	if strings.TrimSpace(p.Branch) == "" {
		return errors.New("branch is required")
	}

	return nil
}

func Undeploy(p UndeployParams) (string, error) {
	branch := p.Branch
	baseDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getwd: %w", err)
	}
	envDir := filepath.Join(baseDir, "envs", branch)

	// Stop and remove container
	helpers.RunCmd("docker", "rm", "-f", branch)

	// Remove image
	helpers.RunCmd("docker", "rmi", "odoo-"+branch)

	// Remove from Caddyfile
	domain := os.Getenv("DOMAIN")
	if domain == "" {
		domain = "localhost"
	}
	caddyPath := filepath.Join(baseDir, "Caddyfile")
	if content, err := os.ReadFile(caddyPath); err == nil {
		entry := fmt.Sprintf("\nhttp://%s.%s {\n    reverse_proxy %s:8069\n}\n", branch, domain, branch)
		updated := strings.ReplaceAll(string(content), entry, "")
		os.WriteFile(caddyPath, []byte(updated), 0644)
		helpers.RunCmd("docker", "compose", "up", "caddy", "-d", "--force-recreate")
	}

	// Remove env directory
	os.RemoveAll(envDir)

	return "Undeployment complete for " + branch, nil
}
