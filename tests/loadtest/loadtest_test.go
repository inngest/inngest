package loadtest

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestLoadMatrix runs the full matrix of system configurations, workloads, and load profiles.
// Skipped by default; set LOADTEST=1 to run.
//
// Usage:
//
//	LOADTEST=1 go test ./tests/loadtest/ -run TestLoadMatrix -timeout 30m -v
//
// To save results:
//
//	LOADTEST=1 LOADTEST_OUTPUT=results.json go test ./tests/loadtest/ -run TestLoadMatrix -timeout 30m
//
// To compare against a baseline:
//
//	LOADTEST=1 LOADTEST_OUTPUT=current.json LOADTEST_BASELINE=baseline.json go test ./tests/loadtest/ -timeout 30m
func TestLoadMatrix(t *testing.T) {
	if os.Getenv("LOADTEST") == "" {
		t.Skip("Set LOADTEST=1 to run load tests")
	}

	configs := []SystemConfig{
		{
			Name:         "default",
			QueueWorkers: 100,
			Tick:         150 * time.Millisecond,
		},
		{
			Name:         "fast-tick",
			QueueWorkers: 200,
			Tick:         50 * time.Millisecond,
		},
		{
			Name:         "key-queues",
			QueueWorkers: 100,
			Tick:         150 * time.Millisecond,
			EnvVars:      map[string]string{"EXPERIMENTAL_KEY_QUEUES_ENABLE": "true"},
		},
	}

	workloads := []Workload{
		SimpleWorkload(),
		BatchWorkload(5, 2*time.Second),
		ConcurrencyWorkload(2),
	}

	profiles := []LoadProfile{
		{Name: "low", Rate: 10, MaxEvents: 50},
		{Name: "medium", Rate: 50, MaxEvents: 200},
	}

	result := RunMatrix(t, configs, workloads, profiles)

	// Write results to file.
	outPath := os.Getenv("LOADTEST_OUTPUT")
	if outPath == "" {
		outPath = "loadtest_results.json"
	}
	f, err := os.Create(outPath)
	require.NoError(t, err)
	defer f.Close()
	require.NoError(t, WriteJSON(f, result))
	t.Logf("Results written to %s", outPath)

	// Compare against baseline if provided.
	baselinePath := os.Getenv("LOADTEST_BASELINE")
	if baselinePath != "" {
		baseline, err := LoadBaseline(baselinePath)
		require.NoError(t, err)
		report, regressed := CompareResults(result, *baseline, 20.0)
		t.Log(report)
		if regressed {
			t.Error("Performance regression detected — see report above")
		}
	}
}

// TestSimpleSmoke runs a single simple scenario as a quick smoke test.
//
// Usage:
//
//	LOADTEST=1 go test ./tests/loadtest/ -run TestSimpleSmoke -timeout 5m -v
func TestSimpleSmoke(t *testing.T) {
	if os.Getenv("LOADTEST") == "" {
		t.Skip("Set LOADTEST=1 to run load tests")
	}

	cfg := SystemConfig{
		Name:         "default",
		QueueWorkers: 100,
		Tick:         150 * time.Millisecond,
	}

	server := StartServer(t, cfg)

	wl := SimpleWorkload()
	lp := LoadProfile{Name: "smoke", Rate: 5, MaxEvents: 10}

	scenarioName := "smoke/simple/low"
	sr := runScenario(t, server, wl, lp, scenarioName)

	t.Logf("Smoke test result: completed=%d/%d p50=%.1fms p95=%.1fms throughput=%.1f/s",
		sr.TotalCompleted, sr.TotalEvents, sr.P50E2EMS, sr.P95E2EMS, sr.ThroughputPerSec)

	require.Greater(t, sr.TotalCompleted, 0, "expected at least one completed event")
	require.Greater(t, sr.ThroughputPerSec, 0.0, "expected positive throughput")
}

// TestCustomScenario demonstrates how to run a custom scenario with specific configuration.
func TestCustomScenario(t *testing.T) {
	if os.Getenv("LOADTEST") == "" {
		t.Skip("Set LOADTEST=1 to run load tests")
	}

	configs := []SystemConfig{
		{Name: "default", QueueWorkers: 100, Tick: 150 * time.Millisecond},
	}

	workloads := []Workload{
		SimpleWorkload(),
		MultiStepWorkload(3),
		ThrottleWorkload(10, time.Second),
	}

	profiles := []LoadProfile{
		{Name: "low", Rate: 5, MaxEvents: 20},
	}

	result := RunMatrix(t, configs, workloads, profiles)

	for _, r := range result.Results {
		t.Logf("%-40s completed=%d/%d p50=%.1fms p95=%.1fms",
			r.Scenario, r.TotalCompleted, r.TotalEvents, r.P50E2EMS, r.P95E2EMS)
	}
}
