package state

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestHistoryBinaryMarshalling(t *testing.T) {

	h := History{
		ID:        ulid.MustNew(ulid.Now(), rand.Reader),
		Type:      enums.HistoryTypeFunctionStarted,
		CreatedAt: time.Now().UTC().Truncate(time.Second),
		Data: map[string]any{
			"test": true,
		},
	}

	t.Run("JSON encoding", func(t *testing.T) {
		DefaultHistoryEncoding = HistoryEncodingJSON

		byt, err := h.MarshalBinary()
		require.NoError(t, err)
		exp, err := json.Marshal(h)
		require.NoError(t, err)
		require.Equal(t, exp, byt)
	})

	t.Run("JSON deocding, without prefix", func(t *testing.T) {
		byt, err := json.Marshal(h)
		require.NoError(t, err)
		decoded := &History{}
		err = decoded.UnmarshalBinary(byt)
		require.NoError(t, err)
		require.EqualValues(t, h, *decoded)
	})

	t.Run("JSON deocding, with prefix", func(t *testing.T) {
		byt, err := json.Marshal(h)
		require.NoError(t, err)
		input := append([]byte(HistoryEncodingJSON), byt...)
		decoded := &History{}
		err = decoded.UnmarshalBinary(input)
		require.NoError(t, err)
		require.EqualValues(t, h, *decoded)
	})

	t.Run("gzip encoding", func(t *testing.T) {
		DefaultHistoryEncoding = HistoryEncodingGZIP

		byt, err := h.MarshalBinary()
		require.NoError(t, err)
		exp, err := json.Marshal(h)
		require.NoError(t, err)
		require.NotEqual(t, exp, byt)
		require.True(t, len(byt) < len(exp))
		require.True(t, bytes.HasPrefix(byt, []byte(HistoryEncodingGZIP)))

		dec := &History{}
		err = dec.UnmarshalBinary(byt)
		require.NoError(t, err)
		require.EqualValues(t, h, *dec)
	})

	h.Type = enums.HistoryTypeStepCompleted
	h.Data = HistoryStep{
		ID:      "step-id",
		Name:    "step-name",
		Attempt: 1,
		Data: map[string]any{
			"ok": true,
		},
	}

	t.Run("History step decoding", func(t *testing.T) {
		DefaultHistoryEncoding = HistoryEncodingGZIP

		byt, err := h.MarshalBinary()
		require.NoError(t, err)

		dec := &History{}
		err = dec.UnmarshalBinary(byt)
		require.NoError(t, err)
		require.EqualValues(t, h, *dec)
		// Assert that data is of type HistoryStep for HistoryTypeStep
		// history elements.
		_, ok := dec.Data.(HistoryStep)
		require.True(t, ok)
	})

	t.Run("Step data", func(t *testing.T) {
		DefaultHistoryEncoding = HistoryEncodingJSON
		h := History{
			ID:        ulid.MustNew(ulid.Now(), rand.Reader),
			Type:      enums.HistoryTypeFunctionStarted,
			CreatedAt: time.Now().UTC().Truncate(time.Second),
			Data: HistoryStep{
				ID:   "hashid",
				Name: "Step name",
				Data: map[string]any{
					"hello": "guvna",
				},
			},
		}

		byt, err := h.MarshalBinary()
		require.NoError(t, err)
		exp, err := json.Marshal(h)
		require.NoError(t, err)
		require.Equal(t, exp, byt)
	})

}
