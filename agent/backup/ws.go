package backup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type Handler struct {
	OdooURL   string
	APIKey    string
	BinaryDir string
}

type validateResponse struct {
	Valid    bool   `json:"valid"`
	Filename string `json:"filename"`
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
		log.Printf("Token validation failed: %v", err)
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	filename := vr.Filename
	if filename == "" || strings.Contains(filename, "/") || strings.Contains(filename, "..") {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(h.BinaryDir, "backups", filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "Backup file not found", http.StatusNotFound)
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

	f, err := os.Open(filePath)
	if err != nil {
		log.Printf("Failed to open backup file: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte("ERROR: Failed to open backup file"))
		return
	}
	defer f.Close()

	buf := make([]byte, 32768)
	for {
		n, err := f.Read(buf)
		if n > 0 {
			if writeErr := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); writeErr != nil {
				log.Printf("WebSocket write error: %v", writeErr)
				return
			}
		}
		if err == io.EOF {
			conn.WriteMessage(websocket.TextMessage, []byte("DONE"))
			return
		}
		if err != nil {
			log.Printf("File read error: %v", err)
			conn.WriteMessage(websocket.TextMessage, []byte("ERROR: File read failed"))
			return
		}
	}
}

func (h *Handler) validateToken(token string) (validateResponse, error) {
	body, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"params":  map[string]string{"token": token},
	})

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", h.OdooURL+"/agent/validate_token", bytes.NewReader(body))
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
