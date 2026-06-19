package event

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnmarshalJSON(t *testing.T) {
	t.Run("sets all fields", func(t *testing.T) {
		r := require.New(t)

		input := `{"id":"evt-1","name":"test/event","data":{"key":"val"},"ts":1700000000000,"v":"2024-01-01","meta":{"sessions":{"conversation_id":"conversation_1234","priority":1}},"user":{"email":"a@b.com"}}`

		var evt Event
		err := json.Unmarshal([]byte(input), &evt)
		r.NoError(err)

		r.Equal("evt-1", evt.ID)
		r.Equal("test/event", evt.Name)
		r.Equal(map[string]any{"key": "val"}, evt.Data)
		r.Equal(int64(1700000000000), evt.Timestamp)
		r.Equal("2024-01-01", evt.Version)
		r.Equal(Sessions{"conversation_id": "conversation_1234", "priority": "1"}, evt.Meta.Sessions)
		r.Equal(map[string]any{"email": "a@b.com"}, evt.User)
	})

	t.Run("ignores top-level sessions", func(t *testing.T) {
		r := require.New(t)

		var evt Event
		err := json.Unmarshal([]byte(`{"name":"test/event","data":{},"sessions":{"conversation_id":"conversation_1234"}}`), &evt)
		r.NoError(err)

		r.Empty(evt.Meta.Sessions)
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

	t.Run("invalid session value type returns error", func(t *testing.T) {
		r := require.New(t)

		var evt Event
		err := json.Unmarshal([]byte(`{"name":"test/event","data":{},"meta":{"sessions":{"conversation_id":null}}}`), &evt)
		r.EqualError(err, `event session "conversation_id" must be a string or number`)
	})

	t.Run("boolean session value returns error", func(t *testing.T) {
		r := require.New(t)

		var evt Event
		err := json.Unmarshal([]byte(`{"name":"test/event","data":{},"meta":{"sessions":{"active":true}}}`), &evt)
		r.EqualError(err, `event session "active" must be a string or number`)
	})

	t.Run("preserves numeric session precision", func(t *testing.T) {
		r := require.New(t)

		var evt Event
		err := json.Unmarshal([]byte(`{"name":"test/event","data":{},"meta":{"sessions":{"conversation_id":9007199254740993}}}`), &evt)
		r.NoError(err)

		r.Equal(Sessions{"conversation_id": "9007199254740993"}, evt.Meta.Sessions)
	})

	t.Run("omits empty meta when marshalled", func(t *testing.T) {
		r := require.New(t)

		byt, err := json.Marshal(Event{Name: "test/event", Data: map[string]any{}})
		r.NoError(err)

		r.NotContains(string(byt), `"meta"`)
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

func TestEventValidateSessions(t *testing.T) {
	t.Run("allows valid sessions", func(t *testing.T) {
		evt := Event{
			Name: "test/event",
			Meta: EventMeta{
				Sessions: Sessions{"conversation_id": "conversation_1234"},
			},
		}

		require.NoError(t, evt.Validate(t.Context()))
	})

	t.Run("rejects too many sessions", func(t *testing.T) {
		evt := Event{
			Name: "test/event",
			Meta: EventMeta{
				Sessions: Sessions{
					"a": "1",
					"b": "2",
					"c": "3",
					"d": "4",
					"e": "5",
					"f": "6",
				},
			},
		}

		require.EqualError(t, evt.Validate(t.Context()), "event sessions can include at most 5 entries")
	})

	t.Run("rejects empty session key", func(t *testing.T) {
		evt := Event{
			Name: "test/event",
			Meta: EventMeta{
				Sessions: Sessions{"": "conversation_1234"},
			},
		}

		require.EqualError(t, evt.Validate(t.Context()), "event session keys cannot be empty")
	})
}
