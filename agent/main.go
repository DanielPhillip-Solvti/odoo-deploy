package main

import (
	"agent/backup"
	"agent/heartbeat"
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

//go:embed all:assets
var assetsFS embed.FS

func extractSubDir(base embed.FS, src, dst string) error {
	if !strings.HasSuffix(src, "/") {
		src += "/"
	}
	return fs.WalkDir(base, strings.TrimSuffix(src, "/"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath := strings.TrimPrefix(path, src)
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

	wsPort := os.Getenv("WS_PORT")
	if wsPort == "" {
		wsPort = "9876"
	}

	wsHandler := &backup.Handler{
		OdooURL:   odooURL,
		APIKey:    apiKey,
		BinaryDir: binaryDir,
	}

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/backup-ws", wsHandler)
		log.Printf("Backup WebSocket server listening on :%s", wsPort)
		if err := http.ListenAndServe("0.0.0.0:"+wsPort, mux); err != nil {
			log.Fatalf("WS server error: %v", err)
		}
	}()

	heartbeatTicker := time.NewTicker(5 * time.Second)
	defer heartbeatTicker.Stop()

	for {
		select {
		case <-heartbeatTicker.C:
			err := heartbeat.ExchangeHeartbeat(odooURL, apiKey)
			if err != nil {
				log.Printf("Error sending heartbeat: %v", err)
			} else {
				log.Println("Heartbeat sent successfully")
			}
		}
	}
}
