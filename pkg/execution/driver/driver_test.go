package driver

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/inngest/inngest/pkg/enums"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/stretchr/testify/require"
)

type stubStateLoader struct {
	sv2.StateLoader
	defers map[string]sv2.DeferMeta
}

func (l stubStateLoader) LoadEvents(ctx context.Context, id sv2.ID) ([]json.RawMessage, error) {
	return []json.RawMessage{[]byte(`{}`)}, nil
}

func (l stubStateLoader) LoadSteps(ctx context.Context, id sv2.ID) (map[string]json.RawMessage, error) {
	return map[string]json.RawMessage{}, nil
}

func (l stubStateLoader) LoadDefersMeta(ctx context.Context, id sv2.ID) (map[string]sv2.DeferMeta, error) {
	return l.defers, nil
}

func TestMarshalV1DefersAbortableOnlyForAfterRun(t *testing.T) {
	sl := stubStateLoader{defers: map[string]sv2.DeferMeta{
		"after-run": {ScheduleStatus: enums.DeferStatusAfterRun},
		"scheduled": {ScheduleStatus: enums.DeferStatusScheduled},
		"aborted":   {ScheduleStatus: enums.DeferStatusAborted},
		"rejected":  {ScheduleStatus: enums.DeferStatusRejected},
	}}

	b, err := MarshalV1(context.Background(), sl, sv2.Metadata{}, inngest.Step{}, 0, "test", 0, 1, "")
	require.NoError(t, err)

	req := SDKRequest{}
	require.NoError(t, json.Unmarshal(b, &req))

	require.Equal(t, map[string]SDKDeferEntry{
		"after-run": {Abortable: true},
		"scheduled": {Abortable: false},
		"aborted":   {Abortable: false},
		"rejected":  {Abortable: false},
	}, req.Defers)
}
