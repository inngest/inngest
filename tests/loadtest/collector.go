package loadtest

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Collector is a thread-safe store for timing samples, keyed by loadtest_id.
// It is shared between the load generator (which records send times) and function
// handlers (which record first-hit and completion times).
type Collector struct {
	mu      sync.Mutex
	samples map[string]*TimingSample

	completed atomic.Int64
	target    int64

	done chan struct{}
	once sync.Once
}

// NewCollector creates a collector that expects expectedCompletions function completions.
func NewCollector(expectedCompletions int) *Collector {
	return &Collector{
		samples: make(map[string]*TimingSample),
		target:  int64(expectedCompletions),
		done:    make(chan struct{}),
	}
}

// RecordSend records when an event was sent. Called by the generator.
func (c *Collector) RecordSend(loadtestID string, sendTime time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.samples[loadtestID] = &TimingSample{
		LoadTestID: loadtestID,
		SendTime:   sendTime,
	}
}

// RecordFirstHit records when the function handler was first invoked.
// Called inside the function handler. Safe to call multiple times (only first call records).
func (c *Collector) RecordFirstHit(loadtestID string, hitTime time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	s, ok := c.samples[loadtestID]
	if !ok {
		// Event was sent before collector was aware; create a placeholder.
		s = &TimingSample{LoadTestID: loadtestID}
		c.samples[loadtestID] = s
	}
	if s.FirstHitTime.IsZero() {
		s.FirstHitTime = hitTime
	}
}

// RecordCompletion records when the function handler finished.
// Called inside the function handler after all work is done.
func (c *Collector) RecordCompletion(loadtestID string, completionTime time.Time) {
	c.mu.Lock()
	s, ok := c.samples[loadtestID]
	if !ok {
		s = &TimingSample{LoadTestID: loadtestID}
		c.samples[loadtestID] = s
	}
	s.CompleteTime = completionTime
	c.mu.Unlock()

	n := c.completed.Add(1)
	if n >= c.target {
		c.once.Do(func() { close(c.done) })
	}
}

// RecordBatchCompletion records completion for a batch of events at once.
// Used by batch workloads where multiple events map to a single function invocation.
func (c *Collector) RecordBatchCompletion(loadtestIDs []string, hitTime, completionTime time.Time) {
	c.mu.Lock()
	for _, id := range loadtestIDs {
		s, ok := c.samples[id]
		if !ok {
			s = &TimingSample{LoadTestID: id}
			c.samples[id] = s
		}
		if s.FirstHitTime.IsZero() {
			s.FirstHitTime = hitTime
		}
		s.CompleteTime = completionTime
	}
	c.mu.Unlock()

	n := c.completed.Add(int64(len(loadtestIDs)))
	if n >= c.target {
		c.once.Do(func() { close(c.done) })
	}
}

// WaitForAll blocks until all expected completions are recorded or timeout expires.
func (c *Collector) WaitForAll(timeout time.Duration) error {
	select {
	case <-c.done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timed out waiting for completions: got %d/%d", c.completed.Load(), c.target)
	}
}

// Samples returns all completed timing samples with computed durations.
func (c *Collector) Samples() []TimingSample {
	c.mu.Lock()
	defer c.mu.Unlock()

	var result []TimingSample
	for _, s := range c.samples {
		if s.CompleteTime.IsZero() || s.SendTime.IsZero() {
			continue
		}
		sample := *s
		if !sample.FirstHitTime.IsZero() {
			sample.FirstHit = sample.FirstHitTime.Sub(sample.SendTime)
			sample.FirstHitMS = float64(sample.FirstHit.Microseconds()) / 1000.0
		}
		sample.E2E = sample.CompleteTime.Sub(sample.SendTime)
		sample.E2EMS = float64(sample.E2E.Microseconds()) / 1000.0
		result = append(result, sample)
	}
	return result
}

// CompletedCount returns the number of completed events so far.
func (c *Collector) CompletedCount() int64 {
	return c.completed.Load()
}
