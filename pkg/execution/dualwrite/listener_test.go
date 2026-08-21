package dualwrite

import (
	"context"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/queue"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/stretchr/testify/require"
)

func TestListenerOnEventReceivedNonBlockingWhenBufferFull(t *testing.T) {
	l := newListenerWithChannels(1, 1, 1) // capacity 1 for each of runs/spans/events

	evt := event.NewBaseTrackedEvent(event.Event{Name: "test"}, nil)

	// Fill the events channel to capacity.
	l.OnEventReceived(context.Background(), evt)
	require.Equal(t, int64(0), l.droppedEvents.Load())

	// This second call must return immediately (not block) even though the
	// channel is full, and must increment the dropped counter.
	done := make(chan struct{})
	go func() {
		l.OnEventReceived(context.Background(), evt)
		close(done)
	}()

	select {
	case <-done:
	case <-timeAfter():
		t.Fatal("OnEventReceived blocked on a full channel")
	}
	require.Equal(t, int64(1), l.droppedEvents.Load())
}

func TestListenerOnFunctionScheduledEnqueuesRunRow(t *testing.T) {
	l := newListenerWithChannels(10, 10, 10)

	l.OnFunctionScheduled(context.Background(), sv2.Metadata{}, queue.Item{}, nil)

	select {
	case row := <-l.runs:
		require.Equal(t, "function_scheduled", row["event_type"])
	default:
		t.Fatal("expected a row on the runs channel")
	}
}

// TestNewListenerEndToEndEventFlow proves a row makes it from a hook call,
// through the non-blocking channel send, through a real batcher flush,
// into the events_staging table via a real duckdb subprocess. Compaction
// (rolling staged rows to Parquet) is out of scope for this task — see
// listener.go's NewListener doc comment — so this test stops at the
// staging-table insert.
func TestNewListenerEndToEndEventFlow(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()

	l := NewListener(db, func(o *setupOpts) { o.batchInterval = 20 * time.Millisecond })

	evt := event.NewBaseTrackedEvent(event.Event{Name: "e2e/test"}, nil)
	l.OnEventReceived(context.Background(), evt)

	require.Eventually(t, func() bool {
		var count int
		row := db.QueryRowContext(context.Background(), "SELECT count(*) FROM events_staging;")
		_ = row.Scan(&count)
		return count == 1
	}, 2*time.Second, 20*time.Millisecond, "row should land in staging after a batch flush")
}
