package heartbeat

import (
	"agent/state"
	"context"
	"log"
	"math"
	"time"
)

func Poller(ctx context.Context, s *state.AgentState, odooURL, apiKey string, eventCh chan<- Event) {
	const (
		minDelay = 100 * time.Millisecond
		maxDelay = 2 * time.Second
	)

	ticker := time.NewTicker(minDelay)
	defer ticker.Stop()
	delay := minDelay

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			events, err := PollEvents(ctx, odooURL, apiKey, s.LastEventID())
			if err != nil {
				log.Printf("Event poll failed: %v", err)
				continue
			}

			for _, e := range events {
				select {
				case eventCh <- e:
					s.SetLastEventID(e.ID)
				case <-ctx.Done():
					return
				}
			}

			if len(events) > 0 {
				delay = minDelay
			} else {
				delay = time.Duration(math.Min(float64(delay)*1.5, float64(maxDelay)))
			}
			ticker.Reset(delay)
		}
	}
}
