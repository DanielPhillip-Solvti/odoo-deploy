package actions

import (
	"errors"
	"strings"
)

type BackupParams struct {
	Branch       string `json:"branch"`
	IsProduction bool   `json:"is_production"`
	WithDump     bool   `json:"with_dump"`
}

func (p BackupParams) Validate() error {
	if strings.TrimSpace(p.Branch) == "" {
		return errors.New("branch is required")
	}
	if !p.IsProduction {
		return errors.New("backups are only allowed for production branches")
	}
	return nil
}

func Backup(p BackupParams) (string, error) {
	// update the _latest template, if !with_dump return
	// create dump, the create neutralised dump
	// restore the _neutralised template from the neutralised dump
	return "Not implemented", nil
}
