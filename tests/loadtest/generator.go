package loadtest

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngestgo"
	"golang.org/x/time/rate"
)

// Generator sends events at a controlled rate according to a LoadProfile.
type Generator struct {
	Client    inngestgo.Client
	EventName string
	Profile   LoadProfile
	Collector *Collector
}

// Run sends events according to the LoadProfile, blocking until generation is complete.
// It returns the total number of events actually sent.
func (g *Generator) Run(ctx context.Context) (int, error) {
	limiter := rate.NewLimiter(rate.Limit(g.Profile.Rate), max(1, int(g.Profile.Rate)))

	var cancel context.CancelFunc
	if g.Profile.Duration > 0 {
		ctx, cancel = context.WithTimeout(ctx, g.Profile.Duration)
		defer cancel()
	}

	var (
		sent   int
		mu     sync.Mutex
		errors []error
	)

	for g.Profile.MaxEvents == 0 || sent < g.Profile.MaxEvents {
		if err := limiter.Wait(ctx); err != nil {
			// Context cancelled or timed out — that's normal for duration-based profiles.
			break
		}

		loadtestID := uuid.New().String()
		sendTime := time.Now()
		g.Collector.RecordSend(loadtestID, sendTime)

		_, err := g.Client.Send(ctx, inngestgo.Event{
			Name: g.EventName,
			Data: map[string]any{
				"loadtest_id": loadtestID,
				"sent_at":     sendTime.UnixMilli(),
			},
		})
		if err != nil {
			mu.Lock()
			errors = append(errors, err)
			mu.Unlock()
			continue
		}

		sent++
	}

	if len(errors) > 0 {
		return sent, fmt.Errorf("generator: %d send errors (first: %w)", len(errors), errors[0])
	}
	return sent, nil
}
