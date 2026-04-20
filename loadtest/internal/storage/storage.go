// Package storage wraps the SQLite database the harness uses to persist
// runs, samples, events, and aggregates. Pure Go via modernc.org/sqlite so
// the harness builds without CGO.
package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/inngest/inngest/loadtest/internal/config"
	"github.com/inngest/inngest/loadtest/internal/telemetry"
)

const schema = `
CREATE TABLE IF NOT EXISTS runs (
    id TEXT PRIMARY KEY,
    created_at INTEGER NOT NULL,
    ended_at INTEGER,
    status TEXT NOT NULL,
    config_json TEXT NOT NULL,
    summary_json TEXT,
    samples_dropped INTEGER NOT NULL DEFAULT 0,
    host_id TEXT
);
CREATE TABLE IF NOT EXISTS samples (
    run_id TEXT NOT NULL,
    worker_id TEXT NOT NULL,
    fn TEXT,
    step TEXT,
    phase TEXT NOT NULL,
    attempt INTEGER NOT NULL,
    ts_ns INTEGER NOT NULL,
    seq INTEGER NOT NULL,
    inngest_run_id TEXT,
    correlation_id TEXT
);
CREATE INDEX IF NOT EXISTS samples_run_phase ON samples(run_id, phase);
CREATE INDEX IF NOT EXISTS samples_inngest_run ON samples(inngest_run_id);
CREATE INDEX IF NOT EXISTS samples_corr ON samples(correlation_id);

CREATE TABLE IF NOT EXISTS events (
    run_id TEXT NOT NULL,
    correlation_id TEXT NOT NULL,
    batch_id TEXT,
    sent_at_ns INTEGER NOT NULL,
    PRIMARY KEY (run_id, correlation_id)
);

CREATE TABLE IF NOT EXISTS aggregates (
    run_id TEXT NOT NULL,
    metric TEXT NOT NULL,
    p50 REAL, p95 REAL, p99 REAL,
    count INTEGER NOT NULL,
    PRIMARY KEY (run_id, metric)
);
`

// Store owns the DB handle. Safe for concurrent use — SQLite with WAL allows
// many readers and a single writer; we serialize writes with a mutex at the
// call sites that batch inserts.
type Store struct {
	db *sql.DB
}

// Open opens or creates the SQLite database at path and applies schema +
// PRAGMAs for WAL mode and relaxed sync.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return &Store{db: db}, nil
}

// Close closes the underlying database.
func (s *Store) Close() error { return s.db.Close() }

// DB returns the underlying *sql.DB for tests and advanced queries.
func (s *Store) DB() *sql.DB { return s.db }

// RunRow is the serialization of a persisted run.
type RunRow struct {
	ID             string          `json:"id"`
	CreatedAt      int64           `json:"createdAt"`
	EndedAt        *int64          `json:"endedAt,omitempty"`
	Status         string          `json:"status"`
	Config         config.RunConfig `json:"config"`
	Summary        map[string]any  `json:"summary,omitempty"`
	SamplesDropped int64           `json:"samplesDropped"`
}

// CreateRun inserts a new run row in status="pending" and returns its ID.
func (s *Store) CreateRun(id string, c config.RunConfig, hostID string) error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		`INSERT INTO runs(id, created_at, status, config_json, host_id) VALUES (?, ?, ?, ?, ?)`,
		id, time.Now().UnixNano(), "pending", string(b), hostID,
	)
	return err
}

// MarkRun transitions a run's status and (optionally) writes a summary blob.
func (s *Store) MarkRun(id string, status string, summary map[string]any) error {
	var summaryJSON any
	if summary != nil {
		b, err := json.Marshal(summary)
		if err != nil {
			return err
		}
		summaryJSON = string(b)
	}
	var ended any
	if status == "completed" || status == "failed" || status == "stopped" {
		ended = time.Now().UnixNano()
	}
	_, err := s.db.Exec(
		`UPDATE runs SET status=?, ended_at=COALESCE(?, ended_at), summary_json=COALESCE(?, summary_json) WHERE id=?`,
		status, ended, summaryJSON, id,
	)
	return err
}

// BumpDropped adds n to the samples_dropped counter for a run.
func (s *Store) BumpDropped(runID string, n int64) error {
	_, err := s.db.Exec(`UPDATE runs SET samples_dropped = samples_dropped + ? WHERE id = ?`, n, runID)
	return err
}

// ListRuns returns the most recent runs, newest first.
func (s *Store) ListRuns(limit int) ([]RunRow, error) {
	rows, err := s.db.Query(
		`SELECT id, created_at, ended_at, status, config_json, summary_json, samples_dropped FROM runs ORDER BY created_at DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []RunRow{}
	for rows.Next() {
		var r RunRow
		var endedAt sql.NullInt64
		var cfgJSON, summaryJSON sql.NullString
		if err := rows.Scan(&r.ID, &r.CreatedAt, &endedAt, &r.Status, &cfgJSON, &summaryJSON, &r.SamplesDropped); err != nil {
			return nil, err
		}
		if endedAt.Valid {
			v := endedAt.Int64
			r.EndedAt = &v
		}
		if cfgJSON.Valid {
			_ = json.Unmarshal([]byte(cfgJSON.String), &r.Config)
		}
		if summaryJSON.Valid {
			_ = json.Unmarshal([]byte(summaryJSON.String), &r.Summary)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetRun fetches one run by id.
func (s *Store) GetRun(id string) (RunRow, error) {
	row := s.db.QueryRow(
		`SELECT id, created_at, ended_at, status, config_json, summary_json, samples_dropped FROM runs WHERE id = ?`,
		id,
	)
	var r RunRow
	var endedAt sql.NullInt64
	var cfgJSON, summaryJSON sql.NullString
	if err := row.Scan(&r.ID, &r.CreatedAt, &endedAt, &r.Status, &cfgJSON, &summaryJSON, &r.SamplesDropped); err != nil {
		return RunRow{}, err
	}
	if endedAt.Valid {
		v := endedAt.Int64
		r.EndedAt = &v
	}
	if cfgJSON.Valid {
		_ = json.Unmarshal([]byte(cfgJSON.String), &r.Config)
	}
	if summaryJSON.Valid {
		_ = json.Unmarshal([]byte(summaryJSON.String), &r.Summary)
	}
	return r, nil
}

// InsertSamples writes a batch of telemetry frames for a run in a single
// transaction. Callers are expected to batch at ~1k frames per call.
func (s *Store) InsertSamples(runID string, frames []telemetry.Frame) error {
	if len(frames) == 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO samples(run_id, worker_id, fn, step, phase, attempt, ts_ns, seq, inngest_run_id, correlation_id) VALUES (?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	for _, f := range frames {
		if _, err := stmt.Exec(runID, f.WorkerID, f.FunctionSlug, f.StepID, string(f.Phase), f.Attempt, f.TSNanos, f.Seq, f.InngestRunID, f.CorrelationID); err != nil {
			_ = stmt.Close()
			_ = tx.Rollback()
			return err
		}
	}
	_ = stmt.Close()
	return tx.Commit()
}

// EventRow describes a fired event for persistence.
type EventRow struct {
	CorrelationID string
	BatchID       string
	SentAtNanos   int64
}

// InsertEvents writes fired-event records.
func (s *Store) InsertEvents(runID string, events []EventRow) error {
	if len(events) == 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO events(run_id, correlation_id, batch_id, sent_at_ns) VALUES (?,?,?,?)`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	for _, e := range events {
		if _, err := stmt.Exec(runID, e.CorrelationID, e.BatchID, e.SentAtNanos); err != nil {
			_ = stmt.Close()
			_ = tx.Rollback()
			return err
		}
	}
	_ = stmt.Close()
	return tx.Commit()
}

// WriteAggregates replaces existing aggregate rows for a run.
func (s *Store) WriteAggregates(runID string, aggs map[string]Aggregate) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM aggregates WHERE run_id = ?`, runID); err != nil {
		_ = tx.Rollback()
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO aggregates(run_id, metric, p50, p95, p99, count) VALUES (?,?,?,?,?,?)`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	for name, a := range aggs {
		if _, err := stmt.Exec(runID, name, a.P50, a.P95, a.P99, a.Count); err != nil {
			_ = stmt.Close()
			_ = tx.Rollback()
			return err
		}
	}
	_ = stmt.Close()
	return tx.Commit()
}

// Aggregate is a summary row for one metric of one run.
type Aggregate struct {
	P50   float64 `json:"p50"`
	P95   float64 `json:"p95"`
	P99   float64 `json:"p99"`
	Count int64   `json:"count"`
}

// ReadAggregates returns all aggregate rows for a run keyed by metric.
func (s *Store) ReadAggregates(runID string) (map[string]Aggregate, error) {
	rows, err := s.db.Query(`SELECT metric, p50, p95, p99, count FROM aggregates WHERE run_id = ?`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]Aggregate{}
	for rows.Next() {
		var m string
		var a Aggregate
		if err := rows.Scan(&m, &a.P50, &a.P95, &a.P99, &a.Count); err != nil {
			return nil, err
		}
		out[m] = a
	}
	return out, rows.Err()
}

// LiveSample is a lightweight projection of one telemetry sample for the UI's
// live dashboard feed.
type LiveSample struct {
	WorkerID     string `json:"workerId"`
	Phase        string `json:"phase"`
	Fn           string `json:"fn"`
	Step         string `json:"step"`
	TSNanos      int64  `json:"ts"`
	InngestRunID string `json:"runId,omitempty"`
}

// ReadLiveSamples returns samples for a run with ts_ns > afterNs. The UI polls
// this endpoint on a 1–2s cadence for the live dashboard.
func (s *Store) ReadLiveSamples(runID string, afterNs int64, limit int) ([]LiveSample, error) {
	rows, err := s.db.Query(
		`SELECT worker_id, phase, fn, step, ts_ns, inngest_run_id FROM samples WHERE run_id = ? AND ts_ns > ? ORDER BY ts_ns ASC LIMIT ?`,
		runID, afterNs, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []LiveSample{}
	for rows.Next() {
		var ls LiveSample
		var fn, step, rid sql.NullString
		if err := rows.Scan(&ls.WorkerID, &ls.Phase, &fn, &step, &ls.TSNanos, &rid); err != nil {
			return nil, err
		}
		ls.Fn = fn.String
		ls.Step = step.String
		ls.InngestRunID = rid.String
		out = append(out, ls)
	}
	return out, rows.Err()
}
