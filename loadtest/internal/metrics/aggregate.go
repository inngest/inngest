// Package metrics computes per-run aggregates from raw telemetry samples.
//
// All arithmetic happens in nanoseconds; results are exposed as float64
// milliseconds because UIs and humans read milliseconds.
package metrics

import (
	"database/sql"
	"sort"

	"github.com/inngest/inngest/loadtest/internal/storage"
)

// MetricNames is the set of metrics computed. Stable order makes tests and
// UI layouts easier.
var MetricNames = []string{
	"event_to_run_ms",
	"inter_step_ms",
	"step_duration_ms",
	"sdk_overhead_ms",
}

// Compute reads all samples + events for a run from the database, computes
// per-metric percentile aggregates, and returns them. The warmupEndNs cutoff
// is used to exclude samples collected before the warmup window closed.
func Compute(db *sql.DB, runID string, warmupEndNs int64) (map[string]storage.Aggregate, error) {
	eventToRun, err := computeEventToRun(db, runID, warmupEndNs)
	if err != nil {
		return nil, err
	}
	interStep, stepDur, err := computeStepDeltas(db, runID, warmupEndNs)
	if err != nil {
		return nil, err
	}

	out := map[string]storage.Aggregate{}
	out["event_to_run_ms"] = aggregateNs(eventToRun)
	out["inter_step_ms"] = aggregateNs(interStep)
	out["step_duration_ms"] = aggregateNs(stepDur)
	out["sdk_overhead_ms"] = storage.Aggregate{} // v1 placeholder — wired when workers emit sdk_request/sdk_response frames
	return out, nil
}

// computeEventToRun joins events and the first fn_start sample on the
// correlation ID the firer embeds in every event payload.
func computeEventToRun(db *sql.DB, runID string, afterNs int64) ([]int64, error) {
	q := `
SELECT s.ts_ns - e.sent_at_ns AS delta
FROM events e
JOIN (
    SELECT correlation_id, MIN(ts_ns) AS ts_ns
    FROM samples
    WHERE run_id = ? AND phase = 'fn_start' AND correlation_id IS NOT NULL AND correlation_id != ''
    GROUP BY correlation_id
) s
ON s.correlation_id = e.correlation_id
WHERE e.run_id = ? AND e.sent_at_ns > ? AND s.ts_ns > e.sent_at_ns`
	rows, err := db.Query(q, runID, runID, afterNs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var d int64
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		if d > 0 {
			out = append(out, d)
		}
	}
	return out, rows.Err()
}

// computeStepDeltas walks each (run_id, step) window to build inter-step and
// step duration samples. It's a simple streaming pass over ordered rows.
func computeStepDeltas(db *sql.DB, runID string, afterNs int64) (inter, dur []int64, err error) {
	rows, err := db.Query(`
SELECT inngest_run_id, fn, step, phase, ts_ns
FROM samples
WHERE run_id = ? AND ts_ns > ? AND phase IN ('step_start', 'step_end')
ORDER BY inngest_run_id, ts_ns ASC`, runID, afterNs)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	type key struct{ run, fn, step string }
	stepStart := map[key]int64{}
	var lastEnd struct {
		run string
		ts  int64
	}

	for rows.Next() {
		var rid, fn, step, phase string
		var ts int64
		if err := rows.Scan(&rid, &fn, &step, &phase, &ts); err != nil {
			return nil, nil, err
		}
		k := key{rid, fn, step}
		switch phase {
		case "step_start":
			stepStart[k] = ts
			if lastEnd.run == rid && lastEnd.ts > 0 {
				inter = append(inter, ts-lastEnd.ts)
			}
		case "step_end":
			if start, ok := stepStart[k]; ok {
				dur = append(dur, ts-start)
				delete(stepStart, k)
			}
			lastEnd.run = rid
			lastEnd.ts = ts
		}
	}
	return inter, dur, rows.Err()
}

func aggregateNs(xs []int64) storage.Aggregate {
	if len(xs) == 0 {
		return storage.Aggregate{}
	}
	sort.Slice(xs, func(i, j int) bool { return xs[i] < xs[j] })
	pick := func(p float64) float64 {
		idx := int(float64(len(xs)-1) * p)
		return float64(xs[idx]) / 1e6
	}
	return storage.Aggregate{
		P50:   pick(0.50),
		P95:   pick(0.95),
		P99:   pick(0.99),
		Count: int64(len(xs)),
	}
}
