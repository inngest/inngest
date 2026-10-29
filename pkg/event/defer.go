package event

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"
)

// DeferEventID returns the deterministic ID for an inngest/deferred.schedule
// event. The ID is derived from (parent run ID, hashedID) so a duplicate
// publish path produces the same event.ID and the runner dedupes on it.
func DeferEventID(parent ulid.ULID, hashedID string) (ulid.ULID, error) {
	return util.DeterministicULID(
		ulid.Time(parent.Time()),

		// "defer-event:" prefix namespaces the seed to prevent collisions with
		// other `(parent, hashedID)` derived seeds.
		fmt.Appendf(nil, "defer-event:%s:%s", parent, hashedID),
	)
}

type DeferredScheduleMetadata struct {
	FnSlug      string    `json:"fn_slug"`
	ParentAppID uuid.UUID `json:"parent_app_id"`

	// Defer span in the parent run. This is used to update the span when
	// scheduling the deferred run.
	ParentDeferSpan *meta.SpanReference `json:"parent_defer_span,omitempty"`

	ParentFnID   uuid.UUID `json:"parent_fn_id"`
	ParentFnSlug string    `json:"parent_fn_slug"`
	ParentRunID  ulid.ULID `json:"parent_run_id"`

	// ParentStatus is how the parent run ended: the string form of
	// enums.StepStatus, one of Completed, Failed or Cancelled.
	ParentStatus string `json:"parent_status,omitempty"`

	// ParentError is the parent run's error, in the same shape as the `error`
	// field of inngest/function.finished. Only set when the parent failed with
	// an error, and reduced to a name and message when over
	// consts.MaxDeferParentErrorSize.
	ParentError json.RawMessage `json:"parent_error,omitempty"`

	// ParentEndedAt is when the parent run ended, in Unix milliseconds.
	ParentEndedAt int64 `json:"parent_ended_at,omitempty"`
}

// validParentStatuses are the run outcomes a parent can report.
var validParentStatuses = map[string]struct{}{
	enums.StepStatusCompleted.String(): {},
	enums.StepStatusFailed.String():    {},
	enums.StepStatusCancelled.String(): {},
}

func (m *DeferredScheduleMetadata) Validate() error {
	var errs []error
	if m.FnSlug == "" {
		errs = append(errs, errors.New("fn_slug is required"))
	}
	if m.ParentAppID == uuid.Nil {
		errs = append(errs, errors.New("parent_app_id is required"))
	}
	if m.ParentFnID == uuid.Nil {
		errs = append(errs, errors.New("parent_fn_id is required"))
	}
	if m.ParentFnSlug == "" {
		errs = append(errs, errors.New("parent_fn_slug is required"))
	}
	if m.ParentRunID == (ulid.ULID{}) {
		errs = append(errs, errors.New("parent_run_id is required"))
	}
	if m.ParentDeferSpan == nil {
		errs = append(errs, errors.New("parent_defer_span is required"))
	}
	// Empty is allowed so events published before parent outcomes existed still
	// validate.
	if m.ParentStatus != "" {
		if _, ok := validParentStatuses[m.ParentStatus]; !ok {
			errs = append(errs, fmt.Errorf("parent_status %q is not Completed, Failed or Cancelled", m.ParentStatus))
		}
	}
	return errors.Join(errs...)
}

// DeferredScheduleMetadata extracts the parent-run linkage from the
// `_inngest` data prefix on an inngest/deferred.schedule event.
func (e Event) DeferredScheduleMetadata() (*DeferredScheduleMetadata, error) {
	raw, ok := e.Data[consts.InngestEventDataPrefix]
	if !ok {
		return nil, fmt.Errorf("no data found in prefix '%s'", consts.InngestEventDataPrefix)
	}
	if v, ok := raw.(DeferredScheduleMetadata); ok {
		return &v, nil
	}
	var m DeferredScheduleMetadata
	byt, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(byt, &m); err != nil {
		return nil, err
	}
	return &m, nil
}
