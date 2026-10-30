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

func (e *executor) invokePause(ctx context.Context, evt event.TrackedEvent, correlationID string) (*state.Pause, error) {
	outcome := "error"
	lookup := "id"
	defer func() {
		metrics.RecordCounterMetric(ctx, 1, metrics.CounterOpt{
			PkgName:     pkgName,
			MetricName:  "invoke_pause_lookup_total",
			Description: "Invoke pause lookups by method and outcome",
			Tags:        map[string]any{"lookup": lookup, "outcome": outcome},
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
		// Older executors created invoke pauses with random IDs.
		lookup = "correlation"
		pause, err = e.pm.PauseByInvokeCorrelationID(ctx, evt.GetWorkspaceID(), correlationID)
	}
	if errors.Is(err, state.ErrPauseNotFound) || errors.Is(err, state.ErrInvokePauseNotFound) {
		outcome = "not_found"
		return nil, state.ErrInvokePauseNotFound
	}
	if err != nil {
		return nil, err
	}

	if pause == nil || (lookup == "id" && pause.ID != pauseID) ||
		pause.Identifier.AccountID != evt.GetAccountID() ||
		pause.WorkspaceID != evt.GetWorkspaceID() ||
		pause.Identifier.RunID != runID || pause.DataKey != stepID ||
		pause.InvokeCorrelationID == nil || *pause.InvokeCorrelationID != correlationID ||
		!pause.IsInvoke() || pause.Event == nil || *pause.Event != event.FnFinishedName {
		outcome = "invalid"
		return nil, fmt.Errorf("invoke pause lookup identity mismatch for pause %s", pauseID)
	}

	outcome = "found"
	return pause, nil
}
