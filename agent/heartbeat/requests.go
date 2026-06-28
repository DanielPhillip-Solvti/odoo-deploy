package heartbeat

import (
	"agent/state"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"time"
)

type HeartbeatRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Params  state.Heartbeat `json:"params"`
}

type HeartbeatResponse struct {
	Status  string  `json:"status"`
	Message string  `json:"message"`
	Events  []Event `json:"events"`
}

type RPCResponse struct {
	JSONRPC string            `json:"jsonrpc"`
	ID      any               `json:"id"`
	Result  HeartbeatResponse `json:"result"`
}

type EventsResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Result  struct {
		Events []Event `json:"events"`
	} `json:"result"`
}

type EventCallbackRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Params  EventCallback `json:"params"`
}

func retryDo(ctx context.Context, client *http.Client, reqFn func() (*http.Request, error), maxAttempts int) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			if backoff > 10*time.Second {
				backoff = 10 * time.Second
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		req, err := reqFn()
		if err != nil {
			lastErr = err
			continue
		}

		resp, err := client.Do(req)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		log.Printf("Request failed (attempt %d/%d): %v", attempt+1, maxAttempts, err)
	}
	return nil, lastErr
}

func ExchangeHeartbeat(ctx context.Context, odooURL, apiKey string, hb state.Heartbeat) (HeartbeatResponse, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	reqFn := func() (*http.Request, error) {
		body, err := json.Marshal(HeartbeatRequest{
			JSONRPC: "2.0",
			Params:  hb,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to marshal heartbeat data: %w", err)
		}
		req, err := http.NewRequestWithContext(ctx, "POST", odooURL+"/agent/heartbeat", bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("failed to build heartbeat request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")
		return req, nil
	}

	resp, err := retryDo(ctx, client, reqFn, 3)
	if err != nil {
		return HeartbeatResponse{}, fmt.Errorf("heartbeat request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("Heartbeat response: %s", string(respBody))

	if resp.StatusCode != http.StatusOK {
		return HeartbeatResponse{}, fmt.Errorf("heartbeat rejected: status %d", resp.StatusCode)
	}

	var rpcResp RPCResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return HeartbeatResponse{}, fmt.Errorf("failed to decode response: %w", err)
	}

	return rpcResp.Result, nil
}

func PollEvents(ctx context.Context, odooURL, apiKey string, lastEventID int) ([]Event, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	reqFn := func() (*http.Request, error) {
		body, err := json.Marshal(map[string]any{
			"jsonrpc": "2.0",
			"params":  map[string]int{"last_event_id": lastEventID},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to marshal poll request: %w", err)
		}
		req, err := http.NewRequestWithContext(ctx, "POST", odooURL+"/agent/events", bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("failed to build poll request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")
		return req, nil
	}

	resp, err := retryDo(ctx, client, reqFn, 2)
	if err != nil {
		return nil, fmt.Errorf("poll request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("poll rejected: status %d", resp.StatusCode)
	}

	var rpcResp EventsResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to decode poll response: %w", err)
	}

	return rpcResp.Result.Events, nil
}

func SendEventCallback(ctx context.Context, odooURL, apiKey string, callback EventCallback) error {
	client := &http.Client{Timeout: 10 * time.Second}

	reqFn := func() (*http.Request, error) {
		body, err := json.Marshal(EventCallbackRequest{
			JSONRPC: "2.0",
			Params:  callback,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to marshal event callback data: %w", err)
		}
		req, err := http.NewRequestWithContext(ctx, "POST", odooURL+"/agent/callback", bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("failed to build event callback request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")
		return req, nil
	}

	resp, err := retryDo(ctx, client, reqFn, 3)
	if err != nil {
		return fmt.Errorf("event callback request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("event callback rejected: status %d", resp.StatusCode)
	}

	return nil
}


