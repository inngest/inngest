package event

import (
	"errors"

	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"
)

// DeferredScheduleEventID returns the deterministic event ID used for a deferred schedule event.
func DeferredScheduleEventID(parentRunID ulid.ULID, hashedID string) (ulid.ULID, error) {
	seed := []byte(parentRunID.String() + hashedID)
	return util.DeterministicULID(ulid.Time(parentRunID.Time()), seed)
}

type DeferredScheduleMetadata struct {
	FnSlug       string `json:"fn_slug"`
	ParentFnSlug string `json:"parent_fn_slug"`
	ParentRunID  string `json:"parent_run_id"`
}

func (m *DeferredScheduleMetadata) Validate() error {
	var errs []error
	if m.FnSlug == "" {
		errs = append(errs, errors.New("fn_slug is required"))
	}
	if m.ParentFnSlug == "" {
		errs = append(errs, errors.New("parent_fn_slug is required"))
	}
	if m.ParentRunID == "" {
		errs = append(errs, errors.New("parent_run_id is required"))
	}
	return errors.Join(errs...)
}
