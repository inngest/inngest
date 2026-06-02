package event

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
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
