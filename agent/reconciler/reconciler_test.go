package reconciler

import (
	"agent/state"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestParseNamesJSONArray(t *testing.T) {
	raw := json.RawMessage(`["/deploy-feature-x"]`)
	name := parseNames(raw)
	if name != "deploy-feature-x" {
		t.Fatalf("expected deploy-feature-x, got %q", name)
	}
}

func TestParseNamesBareString(t *testing.T) {
	raw := json.RawMessage(`"deploy-feature-x"`)
	name := parseNames(raw)
	if name != "deploy-feature-x" {
		t.Fatalf("expected deploy-feature-x, got %q", name)
	}
}

func TestParseNamesNil(t *testing.T) {
	name := parseNames(nil)
	if name != "" {
		t.Fatalf("expected empty string, got %q", name)
	}
}

func TestParseLabelsJSONObject(t *testing.T) {
	raw := json.RawMessage(`{"deploy.branch":"feature-x","deploy.is_production":"false","deploy.odoo_version":"19.0"}`)
	labels := parseLabels(raw)
	if labels["deploy.branch"] != "feature-x" {
		t.Fatalf("expected feature-x, got %q", labels["deploy.branch"])
	}
	if labels["deploy.is_production"] != "false" {
		t.Fatalf("expected false, got %q", labels["deploy.is_production"])
	}
	if labels["deploy.odoo_version"] != "19.0" {
		t.Fatalf("expected 19.0, got %q", labels["deploy.odoo_version"])
	}
}

func TestParseLabelsGoMapString(t *testing.T) {
	raw := json.RawMessage(`map[deploy.branch:feature-x deploy.is_production:false deploy.odoo_version:19.0]`)
	labels := parseLabels(raw)
	if labels["deploy.branch"] != "feature-x" {
		t.Fatalf("expected feature-x, got %q", labels["deploy.branch"])
	}
	if labels["deploy.is_production"] != "false" {
		t.Fatalf("expected false, got %q", labels["deploy.is_production"])
	}
	if labels["deploy.odoo_version"] != "19.0" {
		t.Fatalf("expected 19.0, got %q", labels["deploy.odoo_version"])
	}
}

func TestParseLabelsGoMapStringWithNestedBrackets(t *testing.T) {
	raw := json.RawMessage(`map[deploy.branch:feature-x ports:map[80/tcp:8080]]`)
	labels := parseLabels(raw)
	if labels["deploy.branch"] != "feature-x" {
		t.Fatalf("expected feature-x, got %q", labels["deploy.branch"])
	}
}

func TestParseLabelsCommaSeparated(t *testing.T) {
	raw := json.RawMessage(`"deploy.branch=feature-x,deploy.is_production=false,deploy.odoo_version=19.0"`)
	labels := parseLabels(raw)
	if labels["deploy.branch"] != "feature-x" {
		t.Fatalf("expected feature-x, got %q", labels["deploy.branch"])
	}
	if labels["deploy.is_production"] != "false" {
		t.Fatalf("expected false, got %q", labels["deploy.is_production"])
	}
	if labels["deploy.odoo_version"] != "19.0" {
		t.Fatalf("expected 19.0, got %q", labels["deploy.odoo_version"])
	}
}

func TestParseLabelsNil(t *testing.T) {
	labels := parseLabels(nil)
	if len(labels) != 0 {
		t.Fatalf("expected empty map, got %v", labels)
	}
}

func TestScanBackups(t *testing.T) {
	tmpDir := t.TempDir()
	backupsDir := filepath.Join(tmpDir, "backups")
	if err := os.MkdirAll(backupsDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	files := []string{
		"main_260628.dump",
		"main_260628_neutralised.dump",
		"feature-x_260628.dump",
		"some_other_file.txt",
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(backupsDir, f), []byte("test"), 0644); err != nil {
			t.Fatalf("write %s: %v", f, err)
		}
	}

	backups := scanBackups(tmpDir)
	if len(backups) != 3 {
		t.Fatalf("expected 3 backup files, got %d: %v", len(backups), backups)
	}

	expected := []string{
		"feature-x_260628.dump",
		"main_260628.dump",
		"main_260628_neutralised.dump",
	}
	for i, b := range backups {
		if b != expected[i] {
			t.Fatalf("backup[%d]: expected %q, got %q", i, expected[i], b)
		}
	}
}

func TestScanBackupsNoDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	backups := scanBackups(tmpDir)
	if backups == nil {
		t.Fatal("expected non-nil slice, got nil")
	}
	if len(backups) != 0 {
		t.Fatalf("expected 0 backups, got %d", len(backups))
	}
}

func TestFullContainerJSONRoundTripJSONLabels(t *testing.T) {
	line := `{"Names":["/deploy-feature-x"],"Labels":{"deploy.branch":"feature-x","deploy.is_production":"false","deploy.odoo_version":"19.0"},"State":"running"}`

	var c dockerContainer
	if err := json.Unmarshal([]byte(line), &c); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	labels := parseLabels(c.Labels)
	if labels["deploy.branch"] != "feature-x" {
		t.Fatalf("expected feature-x, got %q", labels["deploy.branch"])
	}
	if labels["deploy.is_production"] != "false" {
		t.Fatalf("expected false, got %q", labels["deploy.is_production"])
	}
}

func TestFullContainerJSONRoundTripGoMapLabels(t *testing.T) {
	line := `{"Names":"deploy-main","Labels":"map[deploy.branch:main deploy.is_production:true deploy.odoo_version:19.0]","State":"running"}`

	var c dockerContainer
	if err := json.Unmarshal([]byte(line), &c); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	labels := parseLabels(c.Labels)
	if labels["deploy.branch"] != "main" {
		t.Fatalf("expected main, got %q", labels["deploy.branch"])
	}
	if labels["deploy.is_production"] != "true" {
		t.Fatalf("expected true, got %q", labels["deploy.is_production"])
	}
}

func TestFullContainerJSONRoundTripCommaSeparatedLabels(t *testing.T) {
	line := `{"Names":"deploy-feature-x","Labels":"deploy.branch=feature-x,deploy.is_production=false,deploy.odoo_version=19.0","State":"running"}`

	var c dockerContainer
	if err := json.Unmarshal([]byte(line), &c); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	labels := parseLabels(c.Labels)
	if labels["deploy.branch"] != "feature-x" {
		t.Fatalf("expected feature-x, got %q", labels["deploy.branch"])
	}
	if labels["deploy.is_production"] != "false" {
		t.Fatalf("expected false, got %q", labels["deploy.is_production"])
	}
	if labels["deploy.odoo_version"] != "19.0" {
		t.Fatalf("expected 19.0, got %q", labels["deploy.odoo_version"])
	}
}

func TestStateUpdateAndBuildHeartbeat(t *testing.T) {
	s := state.New("https://github.com/example/repo")

	prod := state.EnvironmentState{Branch: "main", Status: state.StatusActive, OdooVersion: "19.0"}
	staging := []state.EnvironmentState{
		{Branch: "feature-x", Status: state.StatusActive, OdooVersion: "19.0"},
		{Branch: "feature-y", Status: state.StatusActive, OdooVersion: "18.0"},
	}
	backups := []string{"main_260628.dump", "main_260628_neutralised.dump"}

	s.Update(prod, staging, backups)

	hb := s.BuildHeartbeat()

	if hb.ProductionBranch.Branch != "main" {
		t.Fatalf("expected production branch main, got %q", hb.ProductionBranch.Branch)
	}
	if hb.ProductionBranch.Status != state.StatusActive {
		t.Fatalf("expected active status, got %q", hb.ProductionBranch.Status)
	}
	if len(hb.StagingBranches) != 2 {
		t.Fatalf("expected 2 staging branches, got %d", len(hb.StagingBranches))
	}
	if hb.StagingBranches[0].Branch != "feature-x" {
		t.Fatalf("expected feature-x, got %q", hb.StagingBranches[0].Branch)
	}
	if hb.StagingBranches[1].Branch != "feature-y" {
		t.Fatalf("expected feature-y, got %q", hb.StagingBranches[1].Branch)
	}
	if hb.StagingBranches[1].OdooVersion != "18.0" {
		t.Fatalf("expected 18.0, got %q", hb.StagingBranches[1].OdooVersion)
	}
	if len(hb.Backups) != 2 {
		t.Fatalf("expected 2 backups, got %d", len(hb.Backups))
	}
	if hb.RepoURL != "https://github.com/example/repo" {
		t.Fatalf("expected repo URL, got %q", hb.RepoURL)
	}
}

func TestStateUpdateOverwritesPreviousState(t *testing.T) {
	s := state.New("")

	s.Update(
		state.EnvironmentState{},
		[]state.EnvironmentState{{Branch: "feature-x", Status: state.StatusActive, OdooVersion: "19.0"}},
		nil,
	)

	hb1 := s.BuildHeartbeat()
	if len(hb1.StagingBranches) != 1 {
		t.Fatalf("expected 1 staging branch after first update, got %d", len(hb1.StagingBranches))
	}

	s.Update(
		state.EnvironmentState{},
		[]state.EnvironmentState{},
		nil,
	)

	hb2 := s.BuildHeartbeat()
	if len(hb2.StagingBranches) != 0 {
		t.Fatalf("expected 0 staging branches after second update, got %d", len(hb2.StagingBranches))
	}
}

func TestHeartbeatAfterStateUpdateRoundTripJSON(t *testing.T) {
	s := state.New("https://github.com/example/repo")

	s.Update(
		state.EnvironmentState{Branch: "main", Status: state.StatusActive, OdooVersion: "19.0"},
		[]state.EnvironmentState{{Branch: "feature-x", Status: state.StatusActive, OdooVersion: "19.0"}},
		[]string{"main_260628.dump"},
	)

	hb := s.BuildHeartbeat()
	data, err := json.Marshal(hb)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded state.Heartbeat
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.ProductionBranch.Branch != "main" {
		t.Fatalf("expected main, got %q", decoded.ProductionBranch.Branch)
	}
	if len(decoded.StagingBranches) != 1 || decoded.StagingBranches[0].Branch != "feature-x" {
		t.Fatalf("expected feature-x, got %+v", decoded.StagingBranches)
	}
	if len(decoded.Backups) != 1 || decoded.Backups[0] != "main_260628.dump" {
		t.Fatalf("expected main_260628.dump, got %v", decoded.Backups)
	}
}

// Docker integration test: deploys a real container and verifies
// the reconciler detects it and the heartbeat includes it.
// Run with: DOCKER_TEST=1 go test ./reconciler/ -run TestDockerDeployDetection -v
func TestDockerDeployDetection(t *testing.T) {
	if os.Getenv("DOCKER_TEST") != "1" {
		t.Skip("Set DOCKER_TEST=1 to run Docker integration test")
	}

	containerName := "deploy-test-reconciler"
	branchName := "test-branch"

	// Clean up any leftover container first
	exec.Command("docker", "rm", "-f", containerName).Run()

	// Deploy: simulate what deploy.sh does
	out, err := exec.Command("docker", "run", "-d",
		"--name", containerName,
		"--label", "deploy.branch="+branchName,
		"--label", "deploy.is_production=false",
		"--label", "deploy.odoo_version=19.0",
		"python:3-alpine",
		"sleep", "infinity",
	).CombinedOutput()
	if err != nil {
		t.Fatalf("docker run failed: %v\n%s", err, out)
	}
	defer exec.Command("docker", "rm", "-f", containerName).Run()

	// Reconciler: detect the container
	prod, staging := scanContainers()

	// Verify our test container is in the staging results
	found := false
	for _, s := range staging {
		if s.Branch == branchName {
			found = true
			if s.Status != state.StatusActive {
				t.Fatalf("expected active status, got %q", s.Status)
			}
			if s.OdooVersion != "19.0" {
				t.Fatalf("expected 19.0, got %q", s.OdooVersion)
			}
			break
		}
	}
	if !found {
		t.Fatalf("staging branch %q not found in reconciler results. staging=%+v", branchName, staging)
	}

	// State: update and build heartbeat
	s := state.New("https://github.com/example/repo")
	s.Update(prod, staging, nil)
	hb := s.BuildHeartbeat()

	foundInHB := false
	for _, branch := range hb.StagingBranches {
		if branch.Branch == branchName {
			foundInHB = true
			if branch.Status != state.StatusActive {
				t.Fatalf("heartbeat: expected active, got %q", branch.Status)
			}
			break
		}
	}
	if !foundInHB {
		t.Fatalf("branch %q not found in heartbeat staging. heartbeat=%+v", branchName, hb.StagingBranches)
	}

	t.Logf("Heartbeat payload: production=%q, staging=%d, backups=%d",
		hb.ProductionBranch.Branch, len(hb.StagingBranches), len(hb.Backups))
}
