package heartbeat

import (
	"agent/state"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type heartbeatCapture struct {
	Payload state.Heartbeat `json:"params"`
}

func TestProcessorDeploySendsHeartbeatWithNewEnv(t *testing.T) {
	origHandle := HandleEvent
	HandleEvent = func(event Event) (string, error) {
		return "deploy ok", nil
	}
	defer func() { HandleEvent = origHandle }()

	var capturedHB heartbeatCapture
	callbackCalled := false

	mockOdoo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if strings.HasSuffix(r.URL.Path, "/callback") {
			callbackCalled = true
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "result": map[string]string{"status": "success"}})
			return
		}
		if strings.HasSuffix(r.URL.Path, "/heartbeat") {
			var req struct {
				Params state.Heartbeat `json:"params"`
			}
			json.Unmarshal(body, &req)
			capturedHB.Payload = req.Params
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "result": map[string]any{"status": "success", "message": "", "events": []any{}}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockOdoo.Close()

	s := state.New("https://github.com/example/repo")

	s.Update(state.EnvironmentState{}, []state.EnvironmentState{}, nil)

	eventCh := make(chan Event, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go Processor(ctx, s, mockOdoo.URL, "test-key", eventCh)

	params, _ := json.Marshal(map[string]any{
		"branch":        "feature-x",
		"is_production": true,
	})
	eventCh <- Event{
		ID:         99,
		Action:     "deploy",
		Parameters: params,
	}

	for wait := 10; wait > 0; wait-- {
		time.Sleep(100 * time.Millisecond)
		if capturedHB.Payload.ProductionBranch.Branch == "feature-x" {
			break
		}
	}

	pb := capturedHB.Payload.ProductionBranch
	if pb.Branch != "feature-x" {
		t.Fatalf("expected heartbeat production branch 'feature-x', got %q", pb.Branch)
	}
	if pb.Status != state.StatusActive {
		t.Fatalf("expected heartbeat status 'active', got %q", pb.Status)
	}
	if !callbackCalled {
		t.Fatal("event callback was not sent")
	}
}

func TestProcessorDeployStaging(t *testing.T) {
	origHandle := HandleEvent
	HandleEvent = func(event Event) (string, error) {
		return "deploy ok", nil
	}
	defer func() { HandleEvent = origHandle }()

	var capturedHB heartbeatCapture

	mockOdoo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if strings.HasSuffix(r.URL.Path, "/callback") {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "result": map[string]string{"status": "success"}})
			return
		}
		if strings.HasSuffix(r.URL.Path, "/heartbeat") {
			var req struct {
				Params state.Heartbeat `json:"params"`
			}
			json.Unmarshal(body, &req)
			capturedHB.Payload = req.Params
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "result": map[string]any{"status": "success", "message": "", "events": []any{}}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockOdoo.Close()

	s := state.New("https://github.com/example/repo")
	eventCh := make(chan Event, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go Processor(ctx, s, mockOdoo.URL, "test-key", eventCh)

	params, _ := json.Marshal(map[string]any{
		"branch":        "feature-y",
		"is_production": false,
	})
	eventCh <- Event{
		ID:         100,
		Action:     "deploy",
		Parameters: params,
	}

	for wait := 10; wait > 0; wait-- {
		time.Sleep(100 * time.Millisecond)
		if len(capturedHB.Payload.StagingBranches) > 0 {
			break
		}
	}

	if len(capturedHB.Payload.StagingBranches) != 1 {
		t.Fatalf("expected 1 staging branch, got %d", len(capturedHB.Payload.StagingBranches))
	}
	sb := capturedHB.Payload.StagingBranches[0]
	if sb.Branch != "feature-y" {
		t.Fatalf("expected staging branch 'feature-y', got %q", sb.Branch)
	}
	if sb.Status != state.StatusActive {
		t.Fatalf("expected status 'active', got %q", sb.Status)
	}
}

func TestProcessorUndeployRemovesBranch(t *testing.T) {
	origHandle := HandleEvent
	HandleEvent = func(event Event) (string, error) {
		return "undeploy ok", nil
	}
	defer func() { HandleEvent = origHandle }()

	var capturedHB heartbeatCapture

	mockOdoo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if strings.HasSuffix(r.URL.Path, "/callback") {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "result": map[string]string{"status": "success"}})
			return
		}
		if strings.HasSuffix(r.URL.Path, "/heartbeat") {
			var req struct {
				Params state.Heartbeat `json:"params"`
			}
			json.Unmarshal(body, &req)
			capturedHB.Payload = req.Params
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "result": map[string]any{"status": "success", "message": "", "events": []any{}}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockOdoo.Close()

	s := state.New("https://github.com/example/repo")
	s.AddStagingBranch(state.EnvironmentState{Branch: "feature-z", Status: state.StatusActive})

	eventCh := make(chan Event, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go Processor(ctx, s, mockOdoo.URL, "test-key", eventCh)

	params, _ := json.Marshal(map[string]any{"branch": "feature-z"})
	eventCh <- Event{
		ID:         101,
		Action:     "undeploy",
		Parameters: params,
	}

	for wait := 10; wait > 0; wait-- {
		time.Sleep(100 * time.Millisecond)
		if len(capturedHB.Payload.StagingBranches) == 0 {
			break
		}
	}

	if len(capturedHB.Payload.StagingBranches) != 0 {
		t.Fatalf("expected 0 staging branches after undeploy, got %d: %+v", len(capturedHB.Payload.StagingBranches), capturedHB.Payload.StagingBranches)
	}
}

func TestProcessorDeployWithOdooVersion(t *testing.T) {
	origHandle := HandleEvent
	HandleEvent = func(event Event) (string, error) {
		return "deploy ok", nil
	}
	defer func() { HandleEvent = origHandle }()

	var capturedHB heartbeatCapture

	mockOdoo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if strings.HasSuffix(r.URL.Path, "/callback") {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "result": map[string]string{"status": "success"}})
			return
		}
		if strings.HasSuffix(r.URL.Path, "/heartbeat") {
			var req struct {
				Params state.Heartbeat `json:"params"`
			}
			json.Unmarshal(body, &req)
			capturedHB.Payload = req.Params
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "result": map[string]any{"status": "success", "message": "", "events": []any{}}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockOdoo.Close()

	s := state.New("https://github.com/example/repo")
	eventCh := make(chan Event, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go Processor(ctx, s, mockOdoo.URL, "test-key", eventCh)

	params, _ := json.Marshal(map[string]any{
		"branch":        "versioned",
		"is_production": false,
		"odoo_version":  "18.0",
	})
	eventCh <- Event{
		ID:         102,
		Action:     "deploy",
		Parameters: params,
	}

	for wait := 10; wait > 0; wait-- {
		time.Sleep(100 * time.Millisecond)
		if len(capturedHB.Payload.StagingBranches) > 0 {
			break
		}
	}

	if len(capturedHB.Payload.StagingBranches) != 1 {
		t.Fatalf("expected 1 staging branch, got %d", len(capturedHB.Payload.StagingBranches))
	}
	sb := capturedHB.Payload.StagingBranches[0]
	if sb.Branch != "versioned" {
		t.Fatalf("expected 'versioned', got %q", sb.Branch)
	}
	if sb.OdooVersion != "18.0" {
		t.Fatalf("expected odoo_version '18.0', got %q", sb.OdooVersion)
	}
}
