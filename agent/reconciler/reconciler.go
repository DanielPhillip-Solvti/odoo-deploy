package reconciler

import (
	"agent/helpers"
	"agent/state"
	"context"
	"encoding/json"
	"log"
	"os"
	"sort"
	"strings"
	"time"
)

type dockerContainer struct {
	Names  json.RawMessage `json:"Names"`
	Labels json.RawMessage `json:"Labels"`
	State  string          `json:"State"`
}

func parseNames(raw json.RawMessage) string {
	if raw == nil {
		return ""
	}
	// Try JSON array first: ["/name"]
	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil && len(arr) > 0 {
		return strings.TrimPrefix(arr[0], "/")
	}
	// Fallback: bare string "/name" or "name"
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimPrefix(s, "/")
	}
	return ""
}

func parseLabels(raw json.RawMessage) map[string]string {
	result := map[string]string{}
	if raw == nil {
		return result
	}

	// Try JSON object first: {"key":"val"}
	if err := json.Unmarshal(raw, &result); err == nil {
		return result
	}

	// If raw is a JSON string, unwrap it: "\"map[...]\"" or "\"key=val,...\""
	s := strings.TrimSpace(string(raw))
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		if err := json.Unmarshal(raw, &s); err != nil {
			return result
		}
		s = strings.TrimSpace(s)
	}

	// Format A: Go-style map string "map[key1:val1 key2:val2]"
	if strings.HasPrefix(s, "map[") {
		s = s[4 : len(s)-1]
		parts := splitLabelPairs(s)
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			colonIdx := strings.Index(part, ":")
			if colonIdx < 0 {
				continue
			}
			key := strings.TrimSpace(part[:colonIdx])
			val := strings.TrimSpace(part[colonIdx+1:])
			if key != "" {
				result[key] = val
			}
		}
		return result
	}

	// Format B: comma-separated "key1=val1,key2=val2"
	for _, pair := range strings.Split(s, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		eqIdx := strings.Index(pair, "=")
		if eqIdx < 0 {
			continue
		}
		key := strings.TrimSpace(pair[:eqIdx])
		val := strings.TrimSpace(pair[eqIdx+1:])
		if key != "" {
			result[key] = val
		}
	}
	return result
}

func splitLabelPairs(s string) []string {
	var pairs []string
	depth := 0
	start := 0
	for i, ch := range s {
		switch ch {
		case '[':
			depth++
		case ']':
			depth--
		case ' ':
			if depth == 0 {
				pairs = append(pairs, s[start:i])
				start = i + 1
			}
		}
	}
	if start < len(s) {
		pairs = append(pairs, s[start:])
	}
	return pairs
}

func Run(ctx context.Context, s *state.AgentState, backupDir string) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	reconcile(s, backupDir)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			reconcile(s, backupDir)
		}
	}
}

func reconcile(s *state.AgentState, backupDir string) {
	prod, staging := scanContainers()
	backups := ScanBackups(backupDir)
	s.Update(prod, staging, backups)
	log.Printf("State reconciled: production=%q, staging=%d, backups=%d", prod.Branch, len(staging), len(backups))
}

func scanContainers() (state.EnvironmentState, []state.EnvironmentState) {
	var prod state.EnvironmentState
	var staging []state.EnvironmentState

	out, err := helpers.RunCmd("docker", "ps", "--filter", "label=deploy.branch", "--format", "json")
	if err != nil {
		log.Printf("Failed to scan containers: %v", err)
		return prod, staging
	}

	out = strings.TrimSpace(out)
	if out == "" {
		return prod, staging
	}

	lines := strings.Split(out, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var c dockerContainer
		if err := json.Unmarshal([]byte(line), &c); err != nil {
			log.Printf("Failed to parse container JSON: %v", err)
			continue
		}

		labels := parseLabels(c.Labels)

		odooVersion := labels["deploy.odoo_version"]
		if odooVersion == "" {
			odooVersion = "19.0"
		}
		env := state.EnvironmentState{
			Branch:      labels["deploy.branch"],
			Status:      state.StatusActive,
			OdooVersion: odooVersion,
		}

		if c.State == "running" {
			env.Status = state.StatusActive
		} else {
			env.Status = state.StatusInactive
		}

		if labels["deploy.is_production"] == "true" {
			prod = env
		} else {
			staging = append(staging, env)
		}
	}

	return prod, staging
}

func ScanBackups(backupsDir string) []string {
	entries, err := os.ReadDir(backupsDir)
	if err != nil {
		return []string{}
	}
	backups := []string{}
	for _, e := range entries {
		if !e.IsDir() && (strings.HasSuffix(e.Name(), ".dump") || strings.HasSuffix(e.Name(), "_neutralised.dump")) {
			backups = append(backups, e.Name())
		}
	}
	sort.Strings(backups)
	return backups
}
