package heartbeat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"
)

type HeartbeatRequest struct {
	JSONRPC string           `json:"jsonrpc"`
	Params  Heartbeat `json:"params"`
}

type HeartbeatResponse struct {
	Status string            `json:"status"`
	Message string            `json:"message"`
	Events  []Event `json:"events"`
}

func outboundIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", fmt.Errorf("could not determine outbound IP: %w", err)
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String(), nil
}

func ExchangeHeartbeat(odooURL, apiKey string) error {
	body, err := json.Marshal(HeartbeatRequest{
		JSONRPC: "2.0",	
		Params: BuildHeartbeat(),
	})

	if err != nil {
		return fmt.Errorf("failed to marshal heartbeat data: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", odooURL+"/agent/heartbeat", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to build heartbeat request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("heartbeat request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("heartbeat rejected: status %d", resp.StatusCode)
	}

	var rpcResp HeartbeatResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return fmt.Errorf("failed to decode heartbeat response: %w", err)
	}

	return HandleEvents(odooURL, apiKey, rpcResp.Events)
}

func SendEventCallback(odooURL, apiKey string, eventID int) error {
	// TODO: Implement
	return nil
}
