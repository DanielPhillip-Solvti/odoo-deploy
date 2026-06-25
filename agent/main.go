package main

import (
	"agent/heartbeat"
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

	heartbeatTicker := time.NewTicker(30 * time.Second)
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
