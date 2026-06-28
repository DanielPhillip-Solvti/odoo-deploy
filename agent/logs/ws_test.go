package logs

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type mockOdoo struct {
	server        *httptest.Server
	branch        string
	validationHit bool
}

func newMockOdoo(branch string) *mockOdoo {
	m := &mockOdoo{branch: branch}
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

		params, _ := json.Marshal(map[string]string{"branch": m.branch})
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
				Purpose: "logs",
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

func TestLogStreamInvalidToken(t *testing.T) {
	mock := newMockOdoo("testbranch")
	defer mock.Close()

	handler := &Handler{OdooURL: mock.URL(), APIKey: "test-api-key"}
	wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
	}))
	defer wsServer.Close()

	wsURL := "ws" + strings.TrimPrefix(wsServer.URL, "http") + "?token=bad-token"
	dialer := websocket.Dialer{HandshakeTimeout: 2 * time.Second}
	_, _, err := dialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("expected dial error for invalid token, got nil")
	}
	if !strings.Contains(err.Error(), "bad handshake") {
		t.Fatalf("expected bad handshake error, got: %v", err)
	}
}

func TestLogStreamMissingToken(t *testing.T) {
	mock := newMockOdoo("testbranch")
	defer mock.Close()

	handler := &Handler{OdooURL: mock.URL(), APIKey: "test-api-key"}
	wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
	}))
	defer wsServer.Close()

	wsURL := "ws" + strings.TrimPrefix(wsServer.URL, "http")
	dialer := websocket.Dialer{HandshakeTimeout: 2 * time.Second}
	_, _, err := dialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("expected dial error for missing token, got nil")
	}
}

func TestLogStreamInvalidBranch(t *testing.T) {
	mock := newMockOdoo("../../etc/passwd")
	defer mock.Close()

	handler := &Handler{OdooURL: mock.URL(), APIKey: "test-api-key"}
	wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
	}))
	defer wsServer.Close()

	wsURL := "ws" + strings.TrimPrefix(wsServer.URL, "http") + "?token=valid-token"
	dialer := websocket.Dialer{HandshakeTimeout: 2 * time.Second}
	_, _, err := dialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("expected dial error for path traversal branch, got nil")
	}
}

func TestLogStreamDockerIntegration(t *testing.T) {
	if os.Getenv("DOCKER_TEST") != "1" {
		t.Skip("Set DOCKER_TEST=1 to run Docker integration test")
	}

	branch := "testlogsbranch"
	containerName := "deploy-" + branch

	exec.Command("docker", "rm", "-f", containerName).Run()

	containerLog := "hello from test container"
	out, err := exec.Command("docker", "run", "-d",
		"--name", containerName,
		"alpine",
		"sh", "-c", "echo '"+containerLog+"' && sleep 20",
	).CombinedOutput()
	if err != nil {
		t.Fatalf("docker run failed: %v\n%s", err, out)
	}
	defer exec.Command("docker", "rm", "-f", containerName).Run()

	time.Sleep(500 * time.Millisecond)

	mock := newMockOdoo(branch)
	defer mock.Close()

	handler := &Handler{OdooURL: mock.URL(), APIKey: "test-api-key"}
	wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
	}))
	defer wsServer.Close()

	wsURL := "ws" + strings.TrimPrefix(wsServer.URL, "http") + "?token=valid-token"
	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("ws dial failed: %v", err)
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	foundLog := false
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			break
		}
		if string(data) == containerLog {
			foundLog = true
		}
	}

	if !foundLog {
		t.Fatalf("did not find expected log line %q in streamed output", containerLog)
	}
	if !mock.validationHit {
		t.Fatal("validate_token was never called by the handler")
	}
}
