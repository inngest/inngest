package event

import "errors"

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
