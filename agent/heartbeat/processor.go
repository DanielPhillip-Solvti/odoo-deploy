package heartbeat

import (
	"agent/reconciler"
	"agent/state"
	"context"
	"encoding/json"
	"fmt"
	"log"
)

func Processor(ctx context.Context, s *state.AgentState, odooURL, apiKey, binaryDir string, eventCh <-chan Event) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-eventCh:
			if !ok {
				return
			}

			msg, err := HandleEvent(event)
			callback := EventCallback{
				EventID: event.ID,
				Status:  "success",
				Message: msg,
			}
			if err != nil {
				log.Printf("Event %d failed: %v", event.ID, err)
				callback.Status = "fail"
				callback.Message = err.Error()
			} else {
				log.Printf("Event %d completed successfully", event.ID)
				updateStateFromEvent(s, event, binaryDir)
			}

			if sendErr := SendEventCallback(ctx, odooURL, apiKey, callback); sendErr != nil {
				log.Printf("Failed to send callback for event %d: %v", event.ID, sendErr)
			}

			hb := s.BuildHeartbeat()
			if _, err := ExchangeHeartbeat(ctx, odooURL, apiKey, hb); err != nil {
				log.Printf("Post-callback heartbeat failed: %v", err)
			} else {
				log.Printf("Post-callback heartbeat sent for event %d", event.ID)
			}
		}
	}
}

func updateStateFromEvent(s *state.AgentState, event Event, binaryDir string) {
	var params map[string]any
	if err := json.Unmarshal(event.Parameters, &params); err != nil {
		log.Printf("Failed to parse event %d parameters for state update: %v", event.ID, err)
		return
	}

	switch event.Action {
	case "deploy":
		branch, _ := params["branch"].(string)
		if branch == "" {
			return
		}
		env := state.EnvironmentState{
			Branch: branch,
			Status: state.StatusActive,
		}
		if v, ok := params["odoo_version"]; ok {
			env.OdooVersion, _ = v.(string)
		}
		isProd := fmt.Sprintf("%v", params["is_production"])
		if isProd == "true" || isProd == "1" {
			s.SetProductionBranch(env)
		} else {
			s.AddStagingBranch(env)
		}

	case "undeploy":
		branch, _ := params["branch"].(string)
		if branch == "" {
			return
		}
		pb := s.GetProductionBranch()
		if pb.Branch == branch {
			s.SetProductionBranch(state.EnvironmentState{})
		}
		s.RemoveStagingBranch(branch)

	case "backup":
		backups := reconciler.ScanBackups(binaryDir)
		s.Update(state.EnvironmentState{}, nil, backups)
	}
}
