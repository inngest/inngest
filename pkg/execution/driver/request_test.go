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

func TestSDKRequestContextDispatchID(t *testing.T) {
	t.Run("present DispatchID is serialized as dispatch_id", func(t *testing.T) {
		ctx := SDKRequestContext{DispatchID: "01ABC"}

		b, err := json.Marshal(ctx)
		require.NoError(t, err)
		assert.Contains(t, string(b), `"dispatch_id":"01ABC"`)

		var rt SDKRequestContext
		require.NoError(t, json.Unmarshal(b, &rt))
		assert.Equal(t, "01ABC", rt.DispatchID)
	})

	t.Run("empty DispatchID is omitted from the wire payload", func(t *testing.T) {
		ctx := SDKRequestContext{}

		b, err := json.Marshal(ctx)
		require.NoError(t, err)
		assert.NotContains(t, string(b), "dispatch_id",
			"omitempty must keep the field off the wire so old SDKs see no change")
	})
}
