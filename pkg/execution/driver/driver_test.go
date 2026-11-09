package driver

import (
	"context"
	"encoding/json"
	"testing"

	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/stretchr/testify/require"
)

// marshalStateLoader serves one event and no steps.
type marshalStateLoader struct{}

func (marshalStateLoader) LoadMetadata(context.Context, sv2.ID, ...sv2.LoadMetadataOption) (sv2.Metadata, error) {
	return sv2.Metadata{}, nil
}

func (marshalStateLoader) LoadEvents(context.Context, sv2.ID) ([]json.RawMessage, error) {
	return []json.RawMessage{json.RawMessage(`{"name":"test","data":{}}`)}, nil
}

func (marshalStateLoader) LoadSteps(context.Context, sv2.ID) (map[string]json.RawMessage, error) {
	return map[string]json.RawMessage{}, nil
}

func (marshalStateLoader) LoadStepInputs(context.Context, sv2.ID) (map[string]json.RawMessage, error) {
	return nil, nil
}

func (marshalStateLoader) LoadStepsWithIDs(context.Context, sv2.ID, []string) (map[string]json.RawMessage, error) {
	return nil, nil
}

func (marshalStateLoader) LoadStack(context.Context, sv2.ID) ([]string, error) {
	return nil, nil
}

func (marshalStateLoader) LoadPending(context.Context, sv2.ID) ([]string, error) {
	return nil, nil
}

func (marshalStateLoader) LoadState(context.Context, sv2.ID) (sv2.State, error) {
	return sv2.State{}, nil
}

func (marshalStateLoader) LoadDefers(context.Context, sv2.ID) (map[string]sv2.Defer, error) {
	return nil, nil
}

func (marshalStateLoader) LoadDefersMeta(context.Context, sv2.ID) (map[string]sv2.DeferMeta, error) {
	return nil, nil
}

func TestMarshalV1Loop(t *testing.T) {
	md := sv2.Metadata{
		Config: *sv2.InitConfig(&sv2.Config{}),
	}
	step := inngest.Step{
		ID: "step",
	}

	t.Run("loop on ctx fills the block", func(t *testing.T) {
		ctx := WithLoop(context.Background(), &SDKLoopRequest{
			State:   json.RawMessage(`{"n":1}`),
			Version: 2,
			Key:     "user-1",
			Input: SDKLoopInput{
				Kind: "wake",
				ID:   "soak",
			},
		})
		byt, err := MarshalV1(ctx, marshalStateLoader{}, md, step, 0, "env", 0, 1, "")
		require.NoError(t, err)

		req := SDKRequest{}
		require.NoError(t, json.Unmarshal(byt, &req))
		require.NotNil(t, req.Loop)
		require.JSONEq(t, `{"n":1}`, string(req.Loop.State))
		require.Equal(t, int64(2), req.Loop.Version)
		require.Equal(t, "user-1", req.Loop.Key)
		require.Equal(t, SDKLoopInput{
			Kind: "wake",
			ID:   "soak",
		}, req.Loop.Input)
	})

	t.Run("no loop on ctx leaves it nil", func(t *testing.T) {
		byt, err := MarshalV1(context.Background(), marshalStateLoader{}, md, step, 0, "env", 0, 1, "")
		require.NoError(t, err)
		require.NotContains(t, string(byt), `"loop"`)

		req := SDKRequest{}
		require.NoError(t, json.Unmarshal(byt, &req))
		require.Nil(t, req.Loop)
	})
}
