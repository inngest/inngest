package driver

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestStack(t *testing.T) {
	fnStack := FunctionStack{Stack: nil}

	marshaled, err := json.Marshal(fnStack)
	require.NoError(t, err)

	assert.Equal(t, "{\"stack\":[],\"current\":0}", string(marshaled))
}

func TestSDKRequestLoop(t *testing.T) {
	t.Run("omitted for a plain function", func(t *testing.T) {
		byt, err := json.Marshal(SDKRequest{})
		require.NoError(t, err)
		assert.NotContains(t, string(byt), `"loop"`)
	})

	t.Run("round trips", func(t *testing.T) {
		req := SDKRequest{
			Loop: &SDKLoopRequest{
				State:   json.RawMessage(`{"n":1}`),
				Version: 3,
				Key:     "app_123",
				Input: SDKLoopInput{
					Kind: "wake",
					ID:   "soak",
				},
			},
		}
		byt, err := json.Marshal(req)
		require.NoError(t, err)
		assert.Contains(t, string(byt), `"loop":{"state":{"n":1},"version":3,"key":"app_123","input":{"kind":"wake","id":"soak"}}`)

		out := SDKRequest{}
		require.NoError(t, json.Unmarshal(byt, &out))
		require.NotNil(t, out.Loop)
		assert.Equal(t, int64(3), out.Loop.Version)
		assert.Equal(t, "app_123", out.Loop.Key)
		assert.Equal(t, "wake", out.Loop.Input.Kind)
		assert.Equal(t, "soak", out.Loop.Input.ID)
		assert.JSONEq(t, `{"n":1}`, string(out.Loop.State))
	})

	t.Run("null state decodes", func(t *testing.T) {
		out := SDKRequest{}
		require.NoError(t, json.Unmarshal([]byte(`{"loop":{"state":null,"version":0,"input":{"kind":"event","id":""}}}`), &out))
		require.NotNil(t, out.Loop)
		assert.Equal(t, "null", string(out.Loop.State))
		assert.Empty(t, out.Loop.Key)
		assert.Equal(t, "event", out.Loop.Input.Kind)
	})
}
