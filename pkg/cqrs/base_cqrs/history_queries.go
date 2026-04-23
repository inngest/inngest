package base_cqrs

import (
	"context"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/oklog/ulid/v2"
)

func (w wrapper) CountRuns(ctx context.Context, opts cqrs.CountRunOpts) (int, error) {
	return (&reader{q: w.q}).CountRuns(ctx, opts)
}

func (w wrapper) CountReplayRuns(ctx context.Context, opts cqrs.CountReplayRunsOpts) (cqrs.ReplayRunCounts, error) {
	return (&reader{q: w.q}).CountReplayRuns(ctx, opts)
}

func (w wrapper) GetHistoryRun(ctx context.Context, runID ulid.ULID, opts cqrs.GetRunOpts) (cqrs.Run, error) {
	return (&reader{q: w.q}).GetRun(ctx, runID, opts)
}

func (w wrapper) GetHistoryRuns(ctx context.Context, opts cqrs.GetRunsOpts) ([]cqrs.Run, error) {
	return (&reader{q: w.q}).GetRuns(ctx, opts)
}

func (w wrapper) GetReplayRuns(ctx context.Context, opts cqrs.GetReplayRunsOpts) ([]cqrs.ReplayRun, error) {
	return (&reader{q: w.q}).GetReplayRuns(ctx, opts)
}

func (w wrapper) GetRunHistory(ctx context.Context, runID ulid.ULID, opts cqrs.GetRunOpts) ([]*cqrs.RunHistory, error) {
	return (&reader{q: w.q}).GetRunHistory(ctx, runID, opts)
}

func (w wrapper) GetRunHistoryItemOutput(ctx context.Context, historyID ulid.ULID, opts cqrs.GetHistoryOutputOpts) (*string, error) {
	return (&reader{q: w.q}).GetRunHistoryItemOutput(ctx, historyID, opts)
}

func (w wrapper) GetRunsByEventID(ctx context.Context, eventID ulid.ULID, opts cqrs.GetRunsByEventIDOpts) ([]cqrs.Run, error) {
	return (&reader{q: w.q}).GetRunsByEventID(ctx, eventID, opts)
}

func (w wrapper) GetSkippedRunsByEventID(ctx context.Context, eventID ulid.ULID, opts cqrs.GetRunsByEventIDOpts) ([]cqrs.SkippedRun, error) {
	return (&reader{q: w.q}).GetSkippedRunsByEventID(ctx, eventID, opts)
}

func (w wrapper) GetUsage(ctx context.Context, opts cqrs.GetUsageOpts) ([]cqrs.HistoryUsage, error) {
	return (&reader{q: w.q}).GetUsage(ctx, opts)
}

func (w wrapper) GetActiveRunIDs(ctx context.Context, opts cqrs.GetActiveRunIDsOpts) ([]ulid.ULID, error) {
	return (&reader{q: w.q}).GetActiveRunIDs(ctx, opts)
}

func (w wrapper) CountActiveRuns(ctx context.Context, opts cqrs.CountActiveRunsOpts) (int, error) {
	return (&reader{q: w.q}).CountActiveRuns(ctx, opts)
}
