package main

import (
	"agent/backup"
	"agent/heartbeat"
	"agent/logs"
	"agent/reconciler"
	"agent/shell"
	"agent/state"
	"context"
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

//go:embed all:assets
var assetsFS embed.FS

func extractAll(base embed.FS, dst string) error {
	return fs.WalkDir(base, "assets", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "assets" {
			return nil
		}
		relPath := strings.TrimPrefix(path, "assets/")
		dstPath := filepath.Join(dst, relPath)
		if d.IsDir() {
			return os.MkdirAll(dstPath, 0755)
		}
		data, err := base.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dstPath, data, 0755)
	})
}

func main() {
	extractAssetsDir := flag.String("extract-assets", "", "Extract embedded assets to the given directory and exit")
	flag.Parse()

	if *extractAssetsDir != "" {
		if err := extractAll(assetsFS, *extractAssetsDir); err != nil {
			log.Fatalf("Failed to extract assets: %v", err)
		}
		log.Printf("Assets extracted to %s", *extractAssetsDir)
		return
	}

	odooURL := os.Getenv("ODOO_URL")
	if odooURL == "" {
		log.Fatal("ODOO_URL env var not set")
	}

	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		log.Fatal("API_KEY env var not set")
	}

	binaryDir := filepath.Dir(os.Args[0])

	backupDir := os.Getenv("BACKUP_DIR")
	if backupDir == "" {
		backupDir = "/data/deploy-agent/backups"
	}

	wsPort := os.Getenv("WS_PORT")
	if wsPort == "" {
		wsPort = "9876"
	}

	repoURL := os.Getenv("REPO_URL")

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	agentState := state.New(repoURL)
	eventCh := make(chan heartbeat.Event, 100)

	// Goroutine 1: HTTP / WebSocket server
	mux := http.NewServeMux()
	mux.Handle("/backup-ws", &backup.Handler{
		OdooURL:   odooURL,
		APIKey:    apiKey,
		BackupDir: backupDir,
	})
	mux.Handle("/logs-ws", &logs.Handler{
		OdooURL: odooURL,
		APIKey:  apiKey,
	})
	mux.Handle("/shell-ws", &shell.Handler{
		OdooURL: odooURL,
		APIKey:  apiKey,
	})

	srv := &http.Server{
		Addr:    "0.0.0.0:" + wsPort,
		Handler: mux,
	}

	go func() {
		log.Printf("WebSocket server listening on :%s", wsPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("WS server error: %v", err)
		}
	}()

	// Goroutine 2: State reconciler — inspects Docker + backups every 15s
	go reconciler.Run(ctx, agentState, backupDir)

	// Goroutine 3: Heartbeat sender — health check every 30s
	go heartbeat.Sender(ctx, agentState, odooURL, apiKey)

	// Goroutine 4: Event poller — lightweight poll every 100ms–2s
	go heartbeat.Poller(ctx, agentState, odooURL, apiKey, eventCh)

	// Goroutine 5: Event processor — runs scripts, sends callbacks + immediate heartbeat
	go heartbeat.Processor(ctx, agentState, odooURL, apiKey, binaryDir, eventCh)

	<-ctx.Done()
	log.Println("Shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}
}
