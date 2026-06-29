package logs

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"

	"agent/helpers"
	"agent/token"

	"github.com/gorilla/websocket"
)

type Handler struct {
	OdooURL string
	APIKey  string
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tok := r.URL.Query().Get("token")
	if tok == "" {
		http.Error(w, "Missing token", http.StatusUnauthorized)
		return
	}

	vr, err := token.ValidateToken(r.Context(), h.OdooURL, h.APIKey, tok)
	if err != nil || !vr.Valid || vr.Purpose != "logs" {
		log.Printf("Log token validation failed: %v", err)
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	var params struct {
		Branch string `json:"branch"`
	}
	if err := json.Unmarshal(vr.Params, &params); err != nil || params.Branch == "" {
		http.Error(w, "Invalid token params", http.StatusBadRequest)
		return
	}

	branch := params.Branch
	if branch == "" || strings.Contains(branch, "/") || strings.Contains(branch, "..") {
		http.Error(w, "Invalid branch", http.StatusBadRequest)
		return
	}

	upgrader := websocket.Upgrader{
		CheckOrigin: helpers.CheckOrigin,
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	containerName := branch
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
