package event

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
)

type DeferredScheduleMetadata struct {
	FnSlug            string `json:"fn_slug"`
	ParentFnSlug      string `json:"parent_fn_slug"`
	ParentRunID       string `json:"parent_run_id"`
	ParentFunctionID  string `json:"parent_function_id"`
	HashedDeferID     string `json:"hashed_defer_id"`
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
	if m.ParentFunctionID == "" {
		errs = append(errs, errors.New("parent_function_id is required"))
	} else if parsed, err := uuid.Parse(m.ParentFunctionID); err != nil {
		errs = append(errs, fmt.Errorf("parent_function_id is invalid: %w", err))
	} else if parsed == uuid.Nil {
		// uuid.Parse accepts the zero string; reject it explicitly so a
		// zero-value FunctionID can't slip through and stamp uuid.Nil onto
		// the persisted span (where it would disappear from per-function
		// indexes).
		errs = append(errs, errors.New("parent_function_id must not be the zero uuid"))
	}
	if m.HashedDeferID == "" {
		errs = append(errs, errors.New("hashed_defer_id is required"))
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
