package requests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"
)

type Heartbeat struct {
}

type jsonRequest struct {
	JSONRPC string    `json:"jsonrpc"`
	Params  Heartbeat `json:"params"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type jsonRPCResponse struct {
	Error *jsonRPCError `json:"error"`
}

func outboundIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", fmt.Errorf("could not determine outbound IP: %w", err)
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String(), nil
}

func SendHeartbeat(odooURL, apiKey string) error {
	body, err := json.Marshal(jsonRequest{
		JSONRPC: "2.0",
		Params: Heartbeat{
		},
	})

	if err != nil {
		return fmt.Errorf("failed to marshal registration request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", odooURL+"/heartbeat", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create registration request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("registration request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registration rejected: status %d", resp.StatusCode)
	}

	var rpcResp jsonRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return fmt.Errorf("failed to decode registration response: %w", err)
	}
	if rpcResp.Error != nil {
		return fmt.Errorf("registration failed: %s", rpcResp.Error.Message)
	}
	return nil
}
