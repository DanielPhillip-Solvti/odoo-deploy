package shell

import (
	"encoding/json"
	"log"
	"net/http"
	"os/exec"
	"strings"

	"agent/helpers"
	"agent/token"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

type Handler struct {
	OdooURL string
	APIKey  string
}

type resizeMsg struct {
	Type string `json:"type"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tok := r.URL.Query().Get("token")
	if tok == "" {
		http.Error(w, "Missing token", http.StatusUnauthorized)
		return
	}

	vr, err := token.ValidateToken(r.Context(), h.OdooURL, h.APIKey, tok)
	if err != nil || !vr.Valid || vr.Purpose != "shell" {
		log.Printf("Shell token validation failed: %v", err)
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	var params struct {
		Branch  string `json:"branch"`
		Command string `json:"command"`
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

	shellCmd := params.Command
	if shellCmd == "" {
		shellCmd = "odoo -c /etc/odoo/odoo.conf -d " + branch + " shell"
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

	cmd := exec.Command("docker", "exec", "-it", branch, "bash", "-c", shellCmd)

	ptmx, err := pty.Start(cmd)
	if err != nil {
		log.Printf("Failed to start PTY: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte("ERROR: Failed to start shell"))
		return
	}
	defer ptmx.Close()

	pty.Setsize(ptmx, &pty.Winsize{Rows: 24, Cols: 80})

	done := make(chan struct{})

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				cp := make([]byte, n)
				copy(cp, buf[:n])
				if writeErr := conn.WriteMessage(websocket.BinaryMessage, cp); writeErr != nil {
					ptmx.Close()
					break
				}
			}
			if err != nil {
				break
			}
		}
		close(done)
	}()

	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				ptmx.Close()
				break
			}

			if len(msg) > 0 && msg[0] == '{' {
				var rm resizeMsg
				if json.Unmarshal(msg, &rm) == nil && rm.Type == "resize" && rm.Cols > 0 && rm.Rows > 0 {
					pty.Setsize(ptmx, &pty.Winsize{Rows: uint16(rm.Rows), Cols: uint16(rm.Cols)})
					continue
				}
			}

			ptmx.Write(msg)
		}
	}()

	<-done
	cmd.Wait()
}
