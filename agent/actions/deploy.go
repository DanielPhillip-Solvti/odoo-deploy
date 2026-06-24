package actions

import (
	"agent/config"
	"agent/helpers"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type DeployParams struct {
	AddonsRepository string `json:"addons_repository"`
	Branch           string `json:"branch"`
	OdooVersion      string `json:"odoo_version"`
	IsProduction     bool   `json:"is_production"`
}

func (p DeployParams) Validate() error {
	if strings.TrimSpace(p.AddonsRepository) == "" {
		return errors.New("addons_repository is required")
	}
	if strings.TrimSpace(p.Branch) == "" {
		return errors.New("branch is required")
	}
	if strings.TrimSpace(p.OdooVersion) == "" {
		return errors.New("odoo_version is required")
	}
	return nil
}

// Deploy handles cloning, database setup, image building, and runtime mapping.
func Deploy(p DeployParams) (string, error) {
	addonsRepository := p.AddonsRepository
	branch := p.Branch
	odooVersion := p.OdooVersion
	isProduction := p.IsProduction
	envDir := config.EnvPathAgent(branch)
	token := p.GithubToken

	// Create environment directory if it doesn't exist
	if err := os.MkdirAll(envDir, 0755); err != nil {
		return "", fmt.Errorf("mkdir envDir: %w", err)
	}
	image := "odoo-" + branch

	// --- 1. Validate and Handle Caddyfile Sync ---
	caddyPath := filepath.Join(config.AgentPathAgent, "Caddyfile")
	if info, err := os.Stat(caddyPath); err == nil && info.IsDir() {
		if err := os.RemoveAll(caddyPath); err != nil {
			return "", fmt.Errorf("remove Caddyfile directory mistake: %w", err)
		}
	}
	if _, err := os.Stat(caddyPath); os.IsNotExist(err) {
		if err := os.WriteFile(caddyPath, []byte(""), 0644); err != nil {
			return "", fmt.Errorf("create Caddyfile asset: %w", err)
		}
	}

	// --- 2. Update Caddy Routing Rules ---
	domain := os.Getenv("DOMAIN")
	if domain == "" {
		domain = "localhost"
	}

	// Dynamic mapping: routes external requests from caddy container to odoo branch container over 'web' network
	caddyEntry := fmt.Sprintf("\nhttp://%s.%s {\n    reverse_proxy %s:8069\n}\n", branch, domain, branch)
	existing, err := os.ReadFile(caddyPath)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("read Caddyfile: %w", err)
	}

	if !strings.Contains(string(existing), fmt.Sprintf("http://%s.%s", branch, domain)) {
		f, err := os.OpenFile(caddyPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return "", fmt.Errorf("open Caddyfile: %w", err)
		}
		_, werr := f.WriteString(caddyEntry)
		f.Close()
		if werr != nil {
			return "", fmt.Errorf("write Caddyfile: %w", werr)
		}

		// Zero-Downtime Hot Reload: Signal the Caddy container to instantly parse the new rules
		if _, err := helpers.RunCmd("docker", "exec", "deploy-caddy", "caddy", "reload", "--config", "/etc/caddy/Caddyfile"); err != nil {
			return "", fmt.Errorf("failed to reload caddy configuration proxy: %w", err)
		}
	}

	repoURL := strings.Replace(
		addonsRepository,
		"https://github.com/",
		fmt.Sprintf("https://x-access-token:%s@github.com/", token),
		1,
	)

	// --- 3. Clone or Update Custom Addons ---
	addonsDir := config.AddonsPathAgent(branch)
	if _, err := helpers.RunCmd("git", "clone", repoURL, addonsDir); err != nil {
		return "", err
	}

	// --- 4. Write Configuration and Dockerfile Manifests ---
	if err := os.MkdirAll(envDir, 0755); err != nil {
		return "", fmt.Errorf("mkdir envDir: %w", err)
	}
	_, hasReqs := os.Stat(filepath.Join(addonsDir, "requirements.txt"))
	if err := os.WriteFile(filepath.Join(envDir, "Dockerfile"), []byte(helpers.EnvDockerfile(hasReqs == nil)), 0644); err != nil {
		return "", fmt.Errorf("write Dockerfile: %w", err)
	}
	confPath := filepath.Join(envDir, "odoo.conf")
	if err := os.WriteFile(confPath, []byte(helpers.OdooConf(branch)), 0644); err != nil {
		return "", fmt.Errorf("write odoo.conf: %w", err)
	}

	// --- 5. Prepare Isolated Database Matrix ---
	backupTemplate := "_neutralised"
	if isProduction {
		backupTemplate = "_latest"
	}
	// Note: We reference the 'db' container name natively since the agent coordinates with host privileges
	if _, err := helpers.RunCmd("docker", "exec", "deploy-db", "dropdb", "-U", "odoo", "--if-exists", branch); err != nil {
		return "", err
	}

	_, templateErr := helpers.RunCmd("docker", "exec", "deploy-db", "psql", "-U", "odoo", "-lqt", "--csv")
	templateExists := false
	if templateErr == nil {
		out, _ := helpers.RunCmd("docker", "exec", "deploy-db", "psql", "-U", "odoo", "-tAc",
			"SELECT 1 FROM pg_database WHERE datname='"+backupTemplate+"'")
		templateExists = strings.TrimSpace(out) == "1"
	}

	if templateExists {
		if _, err := helpers.RunCmd("docker", "exec", "deploy-db", "createdb", "-U", "odoo", "-T", backupTemplate, branch); err != nil {
			return "", err
		}
	} else {
		if _, err := helpers.RunCmd("docker", "exec", "deploy-db", "createdb", "-U", "odoo", branch); err != nil {
			return "", err
		}
	}

	// --- 6. Build Target Odoo Core Image ---
	if _, err := helpers.RunCmd("docker", "build",
		"--build-arg", "ODOO_VERSION="+odooVersion,
		"-t", image,
		envDir,
	); err != nil {
		return "", err
	}
	helpers.RunCmd("docker", "rm", "-f", branch)

	// --- 7. Execution and Initialization ---
	if !templateExists {
		if _, err := helpers.RunCmd("docker", "run", "--rm",
			"--name", branch+"-init",
			"--network", "db-network",
			"-v", addonsDir+":"+config.AddonsPathContainer,
			"-v", confPath+":/etc/odoo/odoo.conf",
			image,
			"./odoo/odoo-bin", "-c", "/etc/odoo/odoo.conf", "-i", "base", "--stop-after-init",
		); err != nil {
			return "", fmt.Errorf("odoo schema initialization failed: %w", err)
		}
	}

	// Deploy container directly into the 'web' network plane where Caddy can discover it
	if _, err := helpers.RunCmd("docker", "run", "-d",
		"--name", branch,
		"--network", "web",
		"--restart", "unless-stopped",
		"-v", addonsDir+":"+config.AddonsPathContainer,
		"-v", confPath+":/etc/odoo/odoo.conf",
		image,
		"./odoo/odoo-bin", "-c", "/etc/odoo/odoo.conf",
	); err != nil {
		return "", err
	}

	// Tie the running branch container to the database bridge network as its second interface
	if _, err := helpers.RunCmd("docker", "network", "connect", "db-network", branch); err != nil {
		return "", fmt.Errorf("failed to bridge odoo instance to backend database network: %w", err)
	}

	return "Deployment complete for " + branch, nil
}
