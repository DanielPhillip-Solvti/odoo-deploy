package actions

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type RestoreBackupParams struct {
	Branch       string `json:"branch"`
	IsProduction bool   `json:"is_production"`
}

func (p RestoreBackupParams) Validate() error {
	if strings.TrimSpace(p.Branch) == "" {
		return errors.New("branch is required")
	}
	return nil
}

func RestoreBackup(p RestoreBackupParams) (string, error) {
	backupTemplate := "_neutralised"
	if p.IsProduction {
		backupTemplate = "_latest"
	}

	// Check template exists
	out, err := exec.Command("docker", "exec", "db", "psql", "-U", "odoo", "-tAc",
		"SELECT 1 FROM pg_database WHERE datname='"+backupTemplate+"'").CombinedOutput()
	if err != nil || strings.TrimSpace(string(out)) != "1" {
		return "", fmt.Errorf("backup template %s does not exist", backupTemplate)
	}

	if out, err := exec.Command("docker", "exec", "db", "dropdb", "-U", "odoo", "--if-exists", p.Branch).CombinedOutput(); err != nil {
		return "", fmt.Errorf("dropdb failed: %w\n%s", err, out)
	}
	if out, err := exec.Command("docker", "exec", "db", "createdb", "-U", "odoo", "-T", backupTemplate, p.Branch).CombinedOutput(); err != nil {
		return "", fmt.Errorf("createdb failed: %w\n%s", err, out)
	}

	// Restart Odoo so it picks up the restored database
	if out, err := exec.Command("docker", "restart", p.Branch).CombinedOutput(); err != nil {
		return "", fmt.Errorf("docker restart failed: %w\n%s", err, out)
	}

	return "Backup restored for branch " + p.Branch, nil
}
