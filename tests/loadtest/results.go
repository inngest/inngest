package loadtest

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"
)

// ComputeResult aggregates raw timing samples into a ScenarioResult.
func ComputeResult(scenarioName string, samples []TimingSample, totalEvents, totalErrors int, elapsed time.Duration) ScenarioResult {
	r := ScenarioResult{
		Scenario:       scenarioName,
		TotalEvents:    totalEvents,
		TotalCompleted: len(samples),
		TotalErrors:    totalErrors,
		DurationMS:     float64(elapsed.Milliseconds()),
	}

	if len(samples) == 0 {
		return r
	}

	if elapsed.Seconds() > 0 {
		r.ThroughputPerSec = float64(len(samples)) / elapsed.Seconds()
	}

	// Compute first-hit latencies.
	var firstHits []time.Duration
	var e2es []time.Duration
	var sumFirstHit, sumE2E time.Duration

	for _, s := range samples {
		if s.FirstHit > 0 {
			firstHits = append(firstHits, s.FirstHit)
			sumFirstHit += s.FirstHit
		}
		if s.E2E > 0 {
			e2es = append(e2es, s.E2E)
			sumE2E += s.E2E
		}
	}

	if len(firstHits) > 0 {
		sort.Slice(firstHits, func(i, j int) bool { return firstHits[i] < firstHits[j] })
		r.P50FirstHitMS = durationToMS(percentile(firstHits, 0.50))
		r.P95FirstHitMS = durationToMS(percentile(firstHits, 0.95))
		r.P99FirstHitMS = durationToMS(percentile(firstHits, 0.99))
		r.MeanFirstHitMS = durationToMS(sumFirstHit / time.Duration(len(firstHits)))
	}

	if len(e2es) > 0 {
		sort.Slice(e2es, func(i, j int) bool { return e2es[i] < e2es[j] })
		r.P50E2EMS = durationToMS(percentile(e2es, 0.50))
		r.P95E2EMS = durationToMS(percentile(e2es, 0.95))
		r.P99E2EMS = durationToMS(percentile(e2es, 0.99))
		r.MeanE2EMS = durationToMS(sumE2E / time.Duration(len(e2es)))
	}

	return r
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)-1) * p)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func durationToMS(d time.Duration) float64 {
	return float64(d.Microseconds()) / 1000.0
}

// WriteJSON writes the MatrixResult to the given writer in JSON format.
func WriteJSON(w io.Writer, result MatrixResult) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

// LoadBaseline loads a previous MatrixResult from a JSON file.
func LoadBaseline(path string) (*MatrixResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open baseline: %w", err)
	}
	defer f.Close()

	var result MatrixResult
	if err := json.NewDecoder(f).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode baseline: %w", err)
	}
	return &result, nil
}

// CompareResults compares current results against a baseline and returns a
// human-readable report. It flags regressions where current values exceed
// the baseline by more than thresholdPct percent.
func CompareResults(current, baseline MatrixResult, thresholdPct float64) (report string, regressed bool) {
	// Build a map of baseline results by scenario name.
	baselineMap := make(map[string]ScenarioResult)
	for _, r := range baseline.Results {
		baselineMap[r.Scenario] = r
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Comparison: current (%s) vs baseline (%s)\n",
		current.RunAt.Format(time.RFC3339), baseline.RunAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("Regression threshold: %.1f%%\n\n", thresholdPct))

	sb.WriteString(fmt.Sprintf("%-50s %12s %12s %12s %10s\n",
		"Scenario", "Metric", "Baseline", "Current", "Change"))
	sb.WriteString(strings.Repeat("-", 96) + "\n")

	for _, cr := range current.Results {
		br, ok := baselineMap[cr.Scenario]
		if !ok {
			sb.WriteString(fmt.Sprintf("%-50s (no baseline)\n", cr.Scenario))
			continue
		}

		metrics := []struct {
			name     string
			baseline float64
			current  float64
		}{
			{"P50 FirstHit", br.P50FirstHitMS, cr.P50FirstHitMS},
			{"P95 FirstHit", br.P95FirstHitMS, cr.P95FirstHitMS},
			{"P99 FirstHit", br.P99FirstHitMS, cr.P99FirstHitMS},
			{"P50 E2E", br.P50E2EMS, cr.P50E2EMS},
			{"P95 E2E", br.P95E2EMS, cr.P95E2EMS},
			{"P99 E2E", br.P99E2EMS, cr.P99E2EMS},
			{"Throughput", br.ThroughputPerSec, cr.ThroughputPerSec},
		}

		for _, m := range metrics {
			if m.baseline == 0 {
				continue
			}
			changePct := ((m.current - m.baseline) / m.baseline) * 100
			marker := ""

			// For latency metrics, higher is worse (regression).
			// For throughput, lower is worse (regression).
			isThroughput := m.name == "Throughput"
			isRegression := false
			if isThroughput {
				isRegression = changePct < -thresholdPct
			} else {
				isRegression = changePct > thresholdPct
			}

			if isRegression {
				marker = " REGRESSION"
				regressed = true
			}

			sb.WriteString(fmt.Sprintf("%-50s %12s %10.1fms %10.1fms %+8.1f%%%s\n",
				cr.Scenario, m.name, m.baseline, m.current, changePct, marker))
		}
	}

	return sb.String(), regressed
}
