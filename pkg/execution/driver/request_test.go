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

func TestSDKRequestContextGenerationID(t *testing.T) {
	t.Run("present GenerationID is serialized as generation_id", func(t *testing.T) {
		ctx := SDKRequestContext{GenerationID: 7}

		b, err := json.Marshal(ctx)
		require.NoError(t, err)
		assert.Contains(t, string(b), `"generation_id":7`)

		var rt SDKRequestContext
		require.NoError(t, json.Unmarshal(b, &rt))
		assert.Equal(t, 7, rt.GenerationID)
	})

	t.Run("zero GenerationID is omitted from the wire payload", func(t *testing.T) {
		ctx := SDKRequestContext{}

		b, err := json.Marshal(ctx)
		require.NoError(t, err)
		assert.NotContains(t, string(b), "generation_id",
			"omitempty must keep the field off the wire so old SDKs see no change")
	})
}
