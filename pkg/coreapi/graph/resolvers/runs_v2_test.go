package resolvers

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// fakeRunTriggerManager is a minimal cqrs.Manager stub for
// TestRunTriggerFallsBackToQueuedAtWhenNoTriggerIDs — only the two methods
// RunTrigger actually calls are implemented; everything else falls through
// to the embedded nil interface and would panic if ever invoked.
type fakeRunTriggerManager struct {
	cqrs.Manager
	run *cqrs.TraceRun
}

func (f *fakeRunTriggerManager) GetTraceRun(ctx context.Context, id cqrs.TraceRunIdentifier) (*cqrs.TraceRun, error) {
	return f.run, nil
}

func (f *fakeRunTriggerManager) GetEventsByInternalIDs(ctx context.Context, ids []ulid.ULID) ([]*cqrs.Event, error) {
	return nil, nil
}

// TestRunTriggerFallsBackToQueuedAtWhenNoTriggerIDs is a regression test
// for a real bug: RunTrigger's timestamp starts at the zero time.Time and
// is only ever set inside a loop over run.TriggerIDs. TriggerIDs is empty
// for every DuckDB-backed run today (a known gap: TriggerIDs isn't
// captured yet), so the loop never ran and the zero value flowed straight
// into models.RunTraceTrigger's non-nullable `timestamp: Time!` GQL field —
// which fails to serialize and crashes the whole runTrigger query with "the
// requested element is null which the schema does not allow", rather than
// just leaving a field blank.
func TestRunTriggerFallsBackToQueuedAtWhenNoTriggerIDs(t *testing.T) {
	queuedAt := time.Now().Add(-time.Hour).UTC()
	run := &cqrs.TraceRun{
		RunID:    ulid.MustNew(ulid.Now(), nil).String(),
		QueuedAt: queuedAt,
		Status:   enums.RunStatusCompleted,
		// TriggerIDs intentionally left empty, matching DuckDB-backed runs.
	}

	r := &Resolver{Data: &fakeRunTriggerManager{run: run}}
	qr := &queryResolver{r}

	trigger, err := qr.RunTrigger(context.Background(), run.RunID)
	require.NoError(t, err)
	require.False(t, trigger.Timestamp.IsZero(), "timestamp must never be the zero value — the schema declares it non-nullable")
	require.Equal(t, queuedAt, trigger.Timestamp)
}

func TestRunFunctionRunV2ConversionMatchesMakeFunctionRunV2(t *testing.T) {
	run := &cqrs.TraceRun{
		RunID:      ulid.MustNew(ulid.Now(), nil).String(),
		AppID:      uuid.New(),
		FunctionID: uuid.New(),
		QueuedAt:   time.Now(),
		Status:     enums.RunStatusCompleted,
		StartedAt:  time.Now(),
		EndedAt:    time.Now().Add(time.Second),
	}

	want, err := models.MakeFunctionRunV2(run)
	require.NoError(t, err)
	require.NotNil(t, want)
	require.Equal(t, run.AppID, want.AppID)
	require.Equal(t, models.FunctionRunStatusCompleted, want.Status)
}
