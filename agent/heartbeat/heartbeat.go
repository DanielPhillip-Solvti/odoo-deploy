package heartbeat

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type EnvironmentStatus string

const (
	EnvironmentStatusActive    EnvironmentStatus = "active"
	EnvironmentStatusInactive  EnvironmentStatus = "inactive"
	EnvironmentStatusDeploying EnvironmentStatus = "deploying"
	EnvironmentStatusError     EnvironmentStatus = "error"
)

type EnvironmentState struct {
	OdooVersion string            `json:"odoo_version"`
	Branch      string            `json:"branch"`
	Status      EnvironmentStatus `json:"status"`
}

type Heartbeat struct {
	LastEventID      int                `json:"last_event_id"`
	RepoURL          string             `json:"repo_url"`
	ProductionBranch EnvironmentState   `json:"production_branch"`
	StagingBranches  []EnvironmentState `json:"staging_branches"`
	Backups          []string           `json:"backups"`
	WSUrl            string             `json:"ws_url"`
}

func scanBackups(binaryDir string) []string {
	backupsDir := filepath.Join(binaryDir, "backups")
	entries, err := os.ReadDir(backupsDir)
	if err != nil {
		return []string{}
	}
	var backups []string
	for _, e := range entries {
		if !e.IsDir() && (strings.HasSuffix(e.Name(), ".dump") || strings.HasSuffix(e.Name(), "_neutralised.dump")) {
			backups = append(backups, e.Name())
		}
	}
	sort.Strings(backups)
	return backups
}

func buildWSUrl() string {
	publicURL := os.Getenv("PUBLIC_URL")
	if publicURL == "" {
		return ""
	}
	scheme := "ws://"
	if strings.HasPrefix(publicURL, "https://") {
		scheme = "wss://"
		publicURL = strings.TrimPrefix(publicURL, "https://")
	}
	publicURL = strings.TrimPrefix(publicURL, "http://")
	publicURL = strings.TrimRight(publicURL, "/")
	return scheme + publicURL + "/backup-ws"
}

func BuildHeartbeat() Heartbeat {
	heartbeatFile := os.Getenv("HEARTBEAT_FILE")
	if heartbeatFile == "" {
		heartbeatFile = "/data/deploy-agent/heartbeat.json"
	}

	data, err := os.ReadFile(heartbeatFile)
	if err != nil {
		repoURL := os.Getenv("REPO_URL")
		return Heartbeat{
			LastEventID:      0,
			RepoURL:          repoURL,
			ProductionBranch: EnvironmentState{},
			StagingBranches:  []EnvironmentState{},
			Backups:          []string{},
			WSUrl:            buildWSUrl(),
		}
	}

	var hb Heartbeat
	if err := json.Unmarshal(data, &hb); err != nil {
		return Heartbeat{}
	}

	if hb.RepoURL == "" {
		hb.RepoURL = os.Getenv("REPO_URL")
	}

	if hb.StagingBranches == nil {
		hb.StagingBranches = []EnvironmentState{}
	}

	binaryDir := filepath.Dir(os.Args[0])
	hb.Backups = scanBackups(binaryDir)
	hb.WSUrl = buildWSUrl()

	return hb
}
