package event

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnmarshalJSON(t *testing.T) {
	t.Run("sets all fields", func(t *testing.T) {
		r := require.New(t)

		input := `{"id":"evt-1","name":"test/event","data":{"key":"val"},"ts":1700000000000,"v":"2024-01-01","user":{"email":"a@b.com"}}`

		var evt Event
		err := json.Unmarshal([]byte(input), &evt)
		r.NoError(err)

		r.Equal("evt-1", evt.ID)
		r.Equal("test/event", evt.Name)
		r.Equal(map[string]any{"key": "val"}, evt.Data)
		r.Equal(int64(1700000000000), evt.Timestamp)
		r.Equal("2024-01-01", evt.Version)
		r.Equal(map[string]any{"email": "a@b.com"}, evt.User)
	})

	t.Run("sets size to byte length of input", func(t *testing.T) {
		r := require.New(t)

		input := `{"name":"test/event","data":{}}`

		var evt Event
		err := json.Unmarshal([]byte(input), &evt)
		r.NoError(err)

		r.Equal(len(input), evt.Size())
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		r := require.New(t)

		var evt Event
		err := json.Unmarshal([]byte(`{invalid`), &evt)
		r.Error(err)
	})

	t.Run("Size falls back to marshal when not unmarshalled", func(t *testing.T) {
		r := require.New(t)

		evt := Event{
			Name: "test/event",
			Data: map[string]any{"key": "val"},
		}

		byt, err := json.Marshal(evt)
		r.NoError(err)
		r.Equal(len(byt), evt.Size())
	})
}
