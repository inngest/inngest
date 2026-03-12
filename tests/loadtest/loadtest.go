// Package loadtest provides a framework for load testing the Inngest dev server.
//
// It supports running a matrix of system configurations x workloads x load profiles,
// measuring end-to-end latencies, and comparing results against baselines.
//
// Usage:
//
//	LOADTEST=1 go test ./tests/loadtest/ -run TestLoadMatrix -timeout 30m
package loadtest

import (
	"time"
)

// SystemConfig describes how to configure the dev server instance.
type SystemConfig struct {
	// Name is a human-readable identifier for this configuration.
	Name string

	// QueueWorkers is the number of concurrent queue workers. Default: 100.
	QueueWorkers int

	// Tick is the queue poll interval. Default: 150ms.
	Tick time.Duration

	// EnvVars are additional environment variables to set before starting the server.
	// For example: {"EXPERIMENTAL_KEY_QUEUES_ENABLE": "true"}.
	EnvVars map[string]string
}

// Workload describes what functions to register and what events trigger them.
type Workload struct {
	// Name is a human-readable identifier for this workload.
	Name string

	// SetupFn registers functions on the given client and returns the event name
	// that the generator should send to trigger the workload.
	// The collector is provided so that function handlers can record timing data.
	SetupFn func(client WorkloadClient, collector *Collector) (eventName string, err error)

	// ExpectedCompletions returns the expected number of completions given a total
	// number of events sent. For most workloads this equals the event count, but
	// for batched workloads it may be fewer (e.g. ceil(events / batchSize)).
	// If nil, defaults to identity (events == completions).
	ExpectedCompletions func(totalEventsSent int) int
}

// WorkloadClient wraps an inngestgo.Client with the functionality needed by workloads.
type WorkloadClient interface {
	CreateSimpleFunction(id string, eventName string, handler func(eventData map[string]any) error) error
	CreateBatchFunction(id string, eventName string, batchSize int, batchTimeout time.Duration, handler func(eventsData []map[string]any) error) error
	CreateDebounceFunction(id string, eventName string, period time.Duration, key string, handler func(eventData map[string]any) error) error
	CreateConcurrencyFunction(id string, eventName string, limit int, key string, handler func(eventData map[string]any) error) error
	CreateThrottleFunction(id string, eventName string, limit uint, period time.Duration, handler func(eventData map[string]any) error) error
	CreateMultiStepFunction(id string, eventName string, steps int, handler func(stepIndex int, eventData map[string]any) error) error
}

// LoadProfile controls how events are generated.
type LoadProfile struct {
	// Name is a human-readable identifier for this profile.
	Name string

	// Rate is the target number of events per second.
	Rate float64

	// MaxEvents stops generation after this many events. 0 means unlimited (must set Duration).
	MaxEvents int

	// Duration stops generation after this long. 0 means unlimited (must set MaxEvents).
	Duration time.Duration
}

// Scenario is one cell in the matrix: a specific (SystemConfig, Workload, LoadProfile) tuple.
type Scenario struct {
	System   SystemConfig
	Workload Workload
	Load     LoadProfile
}

// TimingSample records one event's journey through the system.
type TimingSample struct {
	LoadTestID   string        `json:"loadtest_id"`
	SendTime     time.Time     `json:"send_time"`
	FirstHitTime time.Time     `json:"first_hit_time"`
	CompleteTime time.Time     `json:"complete_time"`
	FirstHitMS   float64       `json:"first_hit_ms"`
	E2EMS        float64       `json:"e2e_ms"`
	FirstHit     time.Duration `json:"-"`
	E2E          time.Duration `json:"-"`
}

// ScenarioResult holds aggregate results for one scenario.
type ScenarioResult struct {
	Scenario       string  `json:"scenario"`
	TotalEvents    int     `json:"total_events"`
	TotalCompleted int     `json:"total_completed"`
	TotalErrors    int     `json:"total_errors"`
	DurationMS     float64 `json:"duration_ms"`

	P50FirstHitMS  float64 `json:"p50_first_hit_ms"`
	P95FirstHitMS  float64 `json:"p95_first_hit_ms"`
	P99FirstHitMS  float64 `json:"p99_first_hit_ms"`
	MeanFirstHitMS float64 `json:"mean_first_hit_ms"`

	P50E2EMS  float64 `json:"p50_e2e_ms"`
	P95E2EMS  float64 `json:"p95_e2e_ms"`
	P99E2EMS  float64 `json:"p99_e2e_ms"`
	MeanE2EMS float64 `json:"mean_e2e_ms"`

	ThroughputPerSec float64 `json:"throughput_per_sec"`
}

// MatrixResult holds all scenario results for one run.
type MatrixResult struct {
	RunAt     time.Time        `json:"run_at"`
	GitCommit string           `json:"git_commit"`
	Results   []ScenarioResult `json:"results"`
}
