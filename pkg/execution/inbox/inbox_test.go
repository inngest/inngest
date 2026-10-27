package inbox

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDeliverOncePerMessage(t *testing.T) {
	s := NewMemoryStore()
	ws := uuid.New()
	var got []json.RawMessage
	resume := func(signal string, data json.RawMessage) (bool, error) {
		require.Equal(t, "inbox:k", signal)
		got = append(got, data)
		return true, nil
	}

	ok, err := Deliver(s, ws, "k", resume)
	require.NoError(t, err)
	require.False(t, ok, "nothing queued")

	require.EqualValues(t, 1, s.Append(ws, "k", json.RawMessage(`"a"`)))
	require.EqualValues(t, 2, s.Append(ws, "k", json.RawMessage(`"b"`)))

	ok, _ = Deliver(s, ws, "k", resume)
	require.True(t, ok)
	require.EqualValues(t, 1, s.Acked(ws, "k"))
	ok, _ = Deliver(s, ws, "k", resume)
	require.True(t, ok)
	ok, _ = Deliver(s, ws, "k", resume)
	require.False(t, ok)
	require.Len(t, got, 2)
	require.JSONEq(t, `{"seq":2,"data":"b"}`, string(got[1]))
}

func TestDeliverRollsBackWithoutReader(t *testing.T) {
	s := NewMemoryStore()
	ws := uuid.New()
	s.Append(ws, "k", json.RawMessage(`1`))
	ok, _ := Deliver(s, ws, "k", func(string, json.RawMessage) (bool, error) { return false, nil })
	require.False(t, ok)
	require.EqualValues(t, 0, s.Acked(ws, "k"))
}
