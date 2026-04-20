package storage

import (
	"path/filepath"
	"testing"

	"github.com/inngest/inngest/loadtest/internal/config"
	"github.com/inngest/inngest/loadtest/internal/telemetry"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestCreateAndGetRun(t *testing.T) {
	s := newStore(t)
	cfg := config.Defaults()
	if err := s.CreateRun("r1", cfg, "host-a"); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := s.GetRun("r1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != "r1" || got.Status != "pending" {
		t.Errorf("got %+v", got)
	}
	if got.Config.Target.URL != cfg.Target.URL {
		t.Errorf("config not persisted: %+v", got.Config)
	}
}

func TestInsertSamplesAndLive(t *testing.T) {
	s := newStore(t)
	_ = s.CreateRun("r1", config.Defaults(), "host-a")
	frames := []telemetry.Frame{
		{WorkerID: "w-0", Seq: 1, Phase: telemetry.PhaseFnStart, TSNanos: 100, CorrelationID: "c1"},
		{WorkerID: "w-0", Seq: 2, Phase: telemetry.PhaseStepStart, TSNanos: 200, InngestRunID: "ir1"},
		{WorkerID: "w-0", Seq: 3, Phase: telemetry.PhaseStepEnd, TSNanos: 300, InngestRunID: "ir1"},
	}
	if err := s.InsertSamples("r1", frames); err != nil {
		t.Fatalf("insert: %v", err)
	}
	live, err := s.ReadLiveSamples("r1", 0, 100)
	if err != nil {
		t.Fatalf("live: %v", err)
	}
	if len(live) != 3 {
		t.Fatalf("samples: got %d, want 3", len(live))
	}

	// Cursor semantics: after > latest.ts should return zero samples.
	live, err = s.ReadLiveSamples("r1", 300, 100)
	if err != nil {
		t.Fatalf("live cursor: %v", err)
	}
	if len(live) != 0 {
		t.Errorf("cursor should filter out stale samples, got %d", len(live))
	}
}

func TestMarkRunSetsEndedAt(t *testing.T) {
	s := newStore(t)
	_ = s.CreateRun("r1", config.Defaults(), "h")
	if err := s.MarkRun("r1", "completed", map[string]any{"k": 1}); err != nil {
		t.Fatalf("mark: %v", err)
	}
	got, _ := s.GetRun("r1")
	if got.Status != "completed" {
		t.Errorf("status: got %q", got.Status)
	}
	if got.EndedAt == nil {
		t.Errorf("endedAt not set")
	}
	if got.Summary["k"] != 1.0 { // JSON unmarshal yields float64
		t.Errorf("summary: got %+v", got.Summary)
	}
}
