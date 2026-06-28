package heartbeat

import (
	"agent/state"
	"context"
	"log"
	"time"
)

func Sender(ctx context.Context, s *state.AgentState, odooURL, apiKey string) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hb := s.BuildHeartbeat()
			_, err := ExchangeHeartbeat(ctx, odooURL, apiKey, hb)
			if err != nil {
				log.Printf("Heartbeat send failed: %v", err)
			} else {
				log.Println("Heartbeat sent successfully")
			}
		}
	}
}
