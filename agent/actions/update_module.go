package actions

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type UpdateModuleParams struct {
	Branch string `json:"branch"`
	Module string `json:"module"`
}

func (p UpdateModuleParams) Validate() error {
	if strings.TrimSpace(p.Branch) == "" {
		return errors.New("branch is required")
	}
	return nil
}

func UpdateModule(p UpdateModuleParams) (string, error) {
	// TODO: add odoo -u module when the module is specified
	out, err := exec.Command("docker", "restart", p.Branch).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker restart failed: %w\n%s", err, out)
	}
	return "Odoo restarted for branch " + p.Branch, nil
}
