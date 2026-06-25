package heartbeat

import (
	"agent/actions"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

type Event struct {
	ID         int             `json:"id"`
	Action     string          `json:"action"`
	Timestamp  time.Time       `json:"timestamp"`
	Parameters json.RawMessage `json:"parameters"`
}

func HandleEvents(odooURL, apiKey string, events []Event) error {
	for _, event := range events {
		_, err := HandleEvent(event)

		if err != nil {
			SendEventCallback(odooURL, apiKey, event.ID)
		} else {
			SendEventCallback(odooURL, apiKey, event.ID)
		}
	}
	return nil
}

// ACTIONS = [
//     ('deploy', 'Deploy'),
//     ('undeploy', 'Undeploy'),
//     ('backup', 'Backup'),
//     ('restore_backup', 'Restore Backup'),
//     ('reset_branch', 'Reset Branch'),
//     ('update_module', 'Update Module'),
//     ('download_dump', 'Download Dump'),
//     ('stream_logs', 'Stream Logs'),
// ]

func HandleEvent(event Event) (string, error) {
	switch event.Action {
	case "deploy":
		var params actions.DeployParams
		json.Unmarshal(event.Parameters, &params)
		params.Validate()
		log.Printf("Handling deploy event with parameters: %+v\n", params)
		return actions.Deploy(params)
	case "undeploy":
		var params actions.UndeployParams
		json.Unmarshal(event.Parameters, &params)
		params.Validate()
		log.Printf("Handling undeploy event with parameters: %+v\n", params)
		return actions.Undeploy(params)
	case "backup":
		var params actions.BackupParams
		json.Unmarshal(event.Parameters, &params)
		params.Validate()
		log.Printf("Handling backup event with parameters: %+v\n", params)
		return actions.Backup(params)
	case "restore_backup":
		var params actions.RestoreBackupParams
		json.Unmarshal(event.Parameters, &params)
		params.Validate()
		return actions.RestoreBackup(params)
	case "reset_branch":
		var params actions.ResetBranchParams
		json.Unmarshal(event.Parameters, &params)
		params.Validate()
		return actions.ResetBranch(params)
	case "update_module":
		var params actions.UpdateModuleParams
		json.Unmarshal(event.Parameters, &params)
		params.Validate()
		return actions.UpdateModule(params)
	case "download_dump":
		var params actions.DownloadDumpParams
		json.Unmarshal(event.Parameters, &params)
		params.Validate()
		return actions.DownloadDump(params)
	case "stream_logs":
		var params actions.StreamLogsParams
		json.Unmarshal(event.Parameters, &params)
		params.Validate()
		return actions.StreamLogs(params)
	default:
		return "", fmt.Errorf("unknown action: %s", event.Action)
	}
}
