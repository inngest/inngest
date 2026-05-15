package event

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/inngest/inngest/pkg/consts"
)

type DeferredScheduleMetadata struct {
	FnSlug       string `json:"fn_slug"`
	ParentFnSlug string `json:"parent_fn_slug"`
	ParentRunID  string `json:"parent_run_id"`
	// HashedDeferID is the hashed per-run defer ID; the SDK-supplied ID is
	// hashed inside SaveDefer before reaching this field.
	HashedDeferID string `json:"defer_id"`
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
	if m.HashedDeferID == "" {
		errs = append(errs, errors.New("defer_id is required"))
	}
	return errors.Join(errs...)
}

func (m *DeferredScheduleMetadata) Decode(data any) error {
	byt, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(byt, m)
}

// DeferredScheduleMetadata extracts the parent-run linkage carried inside the
// `_inngest` data prefix of an inngest/deferred.schedule event. Returns an
// error when the prefix is absent or the payload doesn't decode.
func (e Event) DeferredScheduleMetadata() (*DeferredScheduleMetadata, error) {
	raw, ok := e.Data[consts.InngestEventDataPrefix]
	if !ok {
		return nil, fmt.Errorf("no data found in prefix '%s'", consts.InngestEventDataPrefix)
	}

	switch v := raw.(type) {
	case DeferredScheduleMetadata:
		return &v, nil
	case *DeferredScheduleMetadata:
		return v, nil
	default:
		var metadata DeferredScheduleMetadata
		if err := metadata.Decode(raw); err != nil {
			return nil, err
		}
		return &metadata, nil
	}
}
