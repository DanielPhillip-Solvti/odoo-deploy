package backup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// mockOdoo simulates /agent/validate_ws_token (JSON-RPC).
// It accepts a fixed token "valid-token" and returns the given filename in params.
type mockOdoo struct {
	server        *httptest.Server
	filename      string
	validationHit bool
}

func newMockOdoo(filename string) *mockOdoo {
	m := &mockOdoo{filename: filename}
	m.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			JSONRPC string `json:"jsonrpc"`
			Params  struct {
				Token string `json:"token"`
			} `json:"params"`
		}
		json.Unmarshal(body, &req)

		m.validationHit = true
		valid := req.Params.Token == "valid-token"

		params, _ := json.Marshal(map[string]string{"filename": m.filename})
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(struct {
			JSONRPC string `json:"jsonrpc"`
			Result  struct {
				Valid   bool            `json:"valid"`
				Purpose string          `json:"purpose"`
				Params  json.RawMessage `json:"params"`
			} `json:"result"`
		}{
			JSONRPC: "2.0",
			Result: struct {
				Valid   bool            `json:"valid"`
				Purpose string          `json:"purpose"`
				Params  json.RawMessage `json:"params"`
			}{
				Valid:   valid,
				Purpose: "backup",
				Params:  params,
			},
		})
	}))
	return m
}

func (m *mockOdoo) Close() {
	m.server.Close()
}

func (m *mockOdoo) URL() string {
	return m.server.URL
}

func TestDownloadBackup(t *testing.T) {
	// Create a temp directory structure: <tmp>/backups/<file>
	tmpDir := t.TempDir()
	backupsDir := filepath.Join(tmpDir, "backups")
	if err := os.MkdirAll(backupsDir, 0755); err != nil {
		t.Fatalf("mkdir backups: %v", err)
	}

	filename := "main_260627.dump"
	content := "Backup test file for main at Sat Jun 27 12:00:00 UTC 2026\n"

	if err := os.WriteFile(filepath.Join(backupsDir, filename), []byte(content), 0644); err != nil {
		t.Fatalf("write backup file: %v", err)
	}

	// Start mock Odoo server
	mock := newMockOdoo(filename)
	defer mock.Close()

	// Start the backup WS handler server
	handler := &Handler{
		OdooURL:   mock.URL(),
		APIKey:    "test-api-key",
		BinaryDir: tmpDir,
	}

	wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
	}))
	defer wsServer.Close()

	// Convert http URL to ws://
	wsURL := "ws" + strings.TrimPrefix(wsServer.URL, "http") + "?token=valid-token"

	// Connect as a WebSocket client
	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("ws dial failed: %v", err)
	}
	defer conn.Close()

	// Read binary chunks
	var received bytes.Buffer
	done := false
	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read message failed: %v", err)
		}
		if msgType == websocket.TextMessage {
			if string(data) == "DONE" {
				done = true
				break
			}
			if strings.HasPrefix(string(data), "ERROR") {
				t.Fatalf("server error: %s", data)
			}
		}
		if msgType == websocket.BinaryMessage {
			received.Write(data)
		}
	}

	if !done {
		t.Fatal("expected DONE message")
	}

	if received.String() != content {
		t.Fatalf("content mismatch:\ngot:  %q\nwant: %q", received.String(), content)
	}

	if !mock.validationHit {
		t.Fatal("validate_token was never called by the handler")
	}
}

func TestDownloadBackupInvalidToken(t *testing.T) {
	tmpDir := t.TempDir()
	backupsDir := filepath.Join(tmpDir, "backups")
	os.MkdirAll(backupsDir, 0755)
	os.WriteFile(filepath.Join(backupsDir, "test.dump"), []byte("data"), 0644)

	mock := newMockOdoo("test.dump")
	defer mock.Close()

	handler := &Handler{
		OdooURL:   mock.URL(),
		APIKey:    "test-api-key",
		BinaryDir: tmpDir,
	}

	wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
	}))
	defer wsServer.Close()

	// Use an invalid token — server returns 401 before WebSocket upgrade
	wsURL := "ws" + strings.TrimPrefix(wsServer.URL, "http") + "?token=bad-token"

	dialer := websocket.Dialer{HandshakeTimeout: 2 * time.Second}
	_, _, err := dialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("expected dial error for invalid token, got nil")
	}
	// gorilla/websocket returns "bad handshake" for any non-101 response
	if !strings.Contains(err.Error(), "bad handshake") {
		t.Fatalf("expected bad handshake error, got: %v", err)
	}
}

func TestDownloadBackupMissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "backups"), 0755)

	mock := newMockOdoo("nonexistent.dump")
	defer mock.Close()

	handler := &Handler{
		OdooURL:   mock.URL(),
		APIKey:    "test-api-key",
		BinaryDir: tmpDir,
	}

	wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
	}))
	defer wsServer.Close()

	wsURL := "ws" + strings.TrimPrefix(wsServer.URL, "http") + "?token=valid-token"

	dialer := websocket.Dialer{HandshakeTimeout: 2 * time.Second}
	_, _, err := dialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("expected dial error for missing file, got nil")
	}
	if !strings.Contains(err.Error(), "bad handshake") {
		t.Fatalf("expected bad handshake error, got: %v", err)
	}
}

func TestDownloadBackupPathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "backups"), 0755)

	// Create a file outside the backups dir
	secretFile := filepath.Join(tmpDir, "secret.txt")
	os.WriteFile(secretFile, []byte("sensitive"), 0644)

	mock := newMockOdoo("../../secret.txt")
	defer mock.Close()

	handler := &Handler{
		OdooURL:   mock.URL(),
		APIKey:    "test-api-key",
		BinaryDir: tmpDir,
	}

	wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
	}))
	defer wsServer.Close()

	wsURL := fmt.Sprintf("ws%s?token=valid-token", strings.TrimPrefix(wsServer.URL, "http"))
	dialer := websocket.Dialer{HandshakeTimeout: 2 * time.Second}
	_, _, err := dialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("expected dial error for path traversal filename, got nil")
	}
}

func TestDownloadBackupMissingToken(t *testing.T) {
	tmpDir := t.TempDir()
	mock := newMockOdoo("test.dump")
	defer mock.Close()
	handler := &Handler{
		OdooURL:   mock.URL(),
		APIKey:    "test-api-key",
		BinaryDir: tmpDir,
	}

	wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
	}))
	defer wsServer.Close()

	// No token query param
	wsURL := "ws" + strings.TrimPrefix(wsServer.URL, "http")

	dialer := websocket.Dialer{HandshakeTimeout: 2 * time.Second}
	_, _, err := dialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("expected dial error for missing token, got nil")
	}
}
