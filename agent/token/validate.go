package token

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type ValidateResponse struct {
	Valid   bool            `json:"valid"`
	Purpose string          `json:"purpose"`
	Params  json.RawMessage `json:"params"`
}

type validateRPCResult struct {
	Result ValidateResponse `json:"result"`
}

func ValidateToken(odooURL, apiKey, tokenValue string) (ValidateResponse, error) {
	body, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"params":  map[string]string{"token": tokenValue},
	})

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", odooURL+"/agent/validate_ws_token", bytes.NewReader(body))
	if err != nil {
		return ValidateResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return ValidateResponse{}, fmt.Errorf("validation request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var rpcResp validateRPCResult
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return ValidateResponse{}, fmt.Errorf("failed to decode validation response: %w", err)
	}
	return rpcResp.Result, nil
}
