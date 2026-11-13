package driver

import (
	"context"
	"encoding/json"
	"testing"

	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/stretchr/testify/require"
)

// fakeLoader answers only what MarshalV1 reads for a run with no steps.
type fakeLoader struct {
	sv2.StateLoader
}

func (fakeLoader) LoadEvents(ctx context.Context, id sv2.ID) ([]json.RawMessage, error) {
	return []json.RawMessage{json.RawMessage(`{"name":"foo","data":{}}`)}, nil
}

func (fakeLoader) LoadSteps(ctx context.Context, id sv2.ID) (map[string]json.RawMessage, error) {
	return map[string]json.RawMessage{}, nil
}

func (fakeLoader) LoadDefersMeta(ctx context.Context, id sv2.ID) (map[string]sv2.DeferMeta, error) {
	return map[string]sv2.DeferMeta{}, nil
}

// An executor tells the SDK which features it supports on every request, so
// the SDK can refuse step.subscribe against one that does not list it.
func TestMarshalV1ListsCapabilities(t *testing.T) {
	raw, err := MarshalV1(context.Background(), fakeLoader{}, sv2.Metadata{}, inngest.Step{}, 0, "", 0, 1, "")
	require.NoError(t, err)

	var req struct {
		Ctx struct {
			Capabilities []string `json:"capabilities"`
		} `json:"ctx"`
	}
	require.NoError(t, json.Unmarshal(raw, &req))

	require.Contains(t, req.Ctx.Capabilities, CapabilitySubscribe)
	require.Equal(t, "subscribe", CapabilitySubscribe)
}
