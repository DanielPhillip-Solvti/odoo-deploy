package heartbeat

import (
	"agent/helpers"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ActionDef struct {
	ScriptName     string
	RequiredParams []string
}

var actionMap = map[string]ActionDef{
	"deploy":         {ScriptName: "deploy.sh", RequiredParams: []string{"branch", "is_production", "addons_repository"}},
	"undeploy":       {ScriptName: "undeploy.sh", RequiredParams: []string{"branch"}},
	"backup":         {ScriptName: "backup.sh", RequiredParams: []string{"branch"}},
	"restore_backup": {ScriptName: "restore_backup.sh", RequiredParams: []string{"branch"}},
	"reset_branch":   {ScriptName: "reset_branch.sh", RequiredParams: []string{"branch"}},
	"update_module":  {ScriptName: "update_module.sh", RequiredParams: []string{"branch", "module_name"}},
	"install_module": {ScriptName: "install_module.sh", RequiredParams: []string{"branch", "module_name"}},
}

type Event struct {
	ID         int             `json:"id"`
	Action     string          `json:"action"`
	Timestamp  time.Time       `json:"timestamp"`
	Parameters json.RawMessage `json:"parameters"`
}

type EventCallback struct {
	EventID int    `json:"event_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

var HandleEvent = handleEvent

func handleEvent(event Event) (string, error) {
	def, ok := actionMap[event.Action]
	if !ok {
		return "", fmt.Errorf("unknown action: %s", event.Action)
	}

	var params map[string]any
	if err := json.Unmarshal(event.Parameters, &params); err != nil {
		return "", fmt.Errorf("failed to parse parameters for action %s: %w", event.Action, err)
	}

	for _, key := range def.RequiredParams {
		if _, found := params[key]; !found {
			return "", fmt.Errorf("missing required parameter '%s' for action %s", key, event.Action)
		}
	}

	scriptsDir := filepath.Join(filepath.Dir(os.Args[0]), "scripts")

	env := os.Environ()
	for k, v := range params {
		if v == nil {
			continue
		}
		envKey := strings.ToUpper(k)
		envVal := fmt.Sprintf("%v", v)
		env = append(env, envKey+"="+envVal)
	}

	scriptPath := filepath.Join(scriptsDir, def.ScriptName)
	log.Printf("Running %s for action %s (event %d)", scriptPath, event.Action, event.ID)

	out, err := helpers.RunCmdWithEnv(env, "bash", scriptPath)
	if err != nil {
		return out, fmt.Errorf("action %s failed: %w", event.Action, err)
	}
	return strings.TrimSpace(out), nil
}
