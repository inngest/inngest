package executor

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/oklog/ulid/v2"
)

// Invoke pauses have deterministic IDs, so a lost correlation index need not
// prevent delivery of a completion while the pause itself still exists.
func (e *executor) invokePauseByID(ctx context.Context, evt event.TrackedEvent, correlationID string) (*state.Pause, error) {
	outcome := "error"
	defer func() {
		metrics.RecordCounterMetric(ctx, 1, metrics.CounterOpt{
			PkgName:     pkgName,
			MetricName:  "invoke_pause_lookup_fallback_total",
			Description: "Invoke pause lookups after a missing correlation index",
			Tags:        map[string]any{"outcome": outcome},
		})
	}()

	runIDString, stepID, ok := strings.Cut(correlationID, ".")
	runID, err := ulid.ParseStrict(runIDString)
	if !ok || err != nil || stepID == "" || !evt.GetEvent().IsFinishedEvent() {
		outcome = "invalid"
		return nil, state.ErrInvokePauseNotFound
	}

	// Keep this derivation in sync with handleGeneratorInvokeFunction.
	pauseID := inngest.DeterministicSha1UUID(runIDString + stepID)
	pause, err := e.pm.PauseByID(ctx, pauses.Index{
		WorkspaceID: evt.GetWorkspaceID(),
		EventName:   event.FnFinishedName,
	}, pauseID)
	if errors.Is(err, state.ErrPauseNotFound) {
		outcome = "not_found"
		return nil, state.ErrInvokePauseNotFound
	}
	if err != nil {
		return nil, err
	}

	if pause == nil || pause.ID != pauseID ||
		pause.Identifier.AccountID != evt.GetAccountID() ||
		pause.WorkspaceID != evt.GetWorkspaceID() ||
		pause.Identifier.RunID != runID || pause.DataKey != stepID ||
		pause.InvokeCorrelationID == nil || *pause.InvokeCorrelationID != correlationID ||
		!pause.IsInvoke() || pause.Event == nil || *pause.Event != event.FnFinishedName {
		outcome = "invalid"
		return nil, fmt.Errorf("invoke pause fallback identity mismatch for pause %s", pauseID)
	}

	outcome = "found"
	e.log.Warn("invoke pause found without correlation index",
		"account_id", evt.GetAccountID(),
		"workspace_id", evt.GetWorkspaceID(),
		"event_id", evt.GetInternalID(),
		"run_id", runID,
		"pause_id", pauseID,
	)
	return pause, nil
}
