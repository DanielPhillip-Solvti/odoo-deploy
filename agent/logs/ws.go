package logs

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type Handler struct {
	OdooURL string
	APIKey  string
}

type validateResponse struct {
	Valid  bool   `json:"valid"`
	Branch string `json:"branch"`
}

type validateRPCResult struct {
	Result validateResponse `json:"result"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Missing token", http.StatusUnauthorized)
		return
	}

	vr, err := h.validateToken(token)
	if err != nil || !vr.Valid {
		log.Printf("Log token validation failed: %v", err)
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	branch := vr.Branch
	if branch == "" || strings.Contains(branch, "/") || strings.Contains(branch, "..") {
		http.Error(w, "Invalid branch", http.StatusBadRequest)
		return
	}

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	containerName := "deploy-" + branch
	cmd := exec.Command("docker", "logs", "-f", "--tail", "100", containerName)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("Failed to create stdout pipe: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte("ERROR: Failed to attach to container logs"))
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Printf("Failed to create stderr pipe: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte("ERROR: Failed to attach to container logs"))
		return
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start docker logs: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte("ERROR: Container not found or not running"))
		return
	}

	done := make(chan struct{})

	go func() {
		reader := bufio.NewReader(io.MultiReader(stdout, stderr))
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			line = strings.TrimRight(line, "\n\r")
			if writeErr := conn.WriteMessage(websocket.TextMessage, []byte(line)); writeErr != nil {
				break
			}
		}
		close(done)
	}()

	select {
	case <-done:
	case <-r.Context().Done():
		cmd.Process.Kill()
	}

	cmd.Wait()
	conn.WriteMessage(websocket.TextMessage, []byte("DONE"))
}

func (h *Handler) validateToken(token string) (validateResponse, error) {
	body, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"params":  map[string]string{"token": token},
	})

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", h.OdooURL+"/agent/validate_log_token", bytes.NewReader(body))
	if err != nil {
		return validateResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+h.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return validateResponse{}, fmt.Errorf("validation request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var rpcResp validateRPCResult
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return validateResponse{}, fmt.Errorf("failed to decode validation response: %w", err)
	}
	return rpcResp.Result, nil
}
