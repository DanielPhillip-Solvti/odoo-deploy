package main

import (
	"agent/requests"
	"log"
	"os"
	"time"
)

func main() {
	odooURL := os.Getenv("ODOO_URL")
	if odooURL == "" {
		log.Fatal("ODOO_URL env var not set")
	}

	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		log.Fatal("API_KEY env var not set")
	}

	// Periodically send heartbeat to Odoo server
	heartbeatTicker := time.NewTicker(30 * time.Second)
	defer heartbeatTicker.Stop()

	for {
		select {
		case <-heartbeatTicker.C:
			err := requests.SendHeartbeat(odooURL, apiKey)
			if err != nil {
				log.Printf("Error sending heartbeat: %v", err)
			} else {
				log.Println("Heartbeat sent successfully")
			}
		}
	}
}
