package backup

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"agent/token"

	"github.com/gorilla/websocket"
)

type Handler struct {
	OdooURL   string
	APIKey    string
	BinaryDir string
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tok := r.URL.Query().Get("token")
	if tok == "" {
		http.Error(w, "Missing token", http.StatusUnauthorized)
		return
	}

	vr, err := token.ValidateToken(h.OdooURL, h.APIKey, tok)
	if err != nil || !vr.Valid || vr.Purpose != "backup" {
		log.Printf("Backup token validation failed: %v", err)
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	var params struct {
		Filename string `json:"filename"`
	}
	if err := json.Unmarshal(vr.Params, &params); err != nil || params.Filename == "" {
		http.Error(w, "Invalid token params", http.StatusBadRequest)
		return
	}

	filename := params.Filename
	if strings.Contains(filename, "/") || strings.Contains(filename, "..") {
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
		conn.WriteMessage(websocket.TextMessage, []byte("ERROR: Failed to open backup file"))
		return
	}
	defer f.Close()

	buf := make([]byte, 32768)
	for {
		n, err := f.Read(buf)
		if n > 0 {
			if writeErr := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); writeErr != nil {
				return
			}
		}
		if err == io.EOF {
			conn.WriteMessage(websocket.TextMessage, []byte("DONE"))
			return
		}
		if err != nil {
			conn.WriteMessage(websocket.TextMessage, []byte("ERROR: File read failed"))
			return
		}
	}
}
