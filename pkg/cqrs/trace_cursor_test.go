package cqrs

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTraceCursorPreservesLegacyWireFormats(t *testing.T) {
	for _, test := range []struct {
		name   string
		cursor TraceCursor
		json   string
	}{
		// Cloud event-run cursors carry only a run ID; cloud metrics use milliseconds.
		{"event run marker", TraceCursor{Field: "run_id"}, `{"id":"01J00000000000000000000001","c":{"run_id":{"f":"run_id","v":0}}}`},
		{"milliseconds", TraceCursor{Field: "started_at", Value: 1767268801000}, `{"id":"01J00000000000000000000001","c":{"started_at":{"f":"started_at","v":1767268801000}}}`},
		{"microseconds", TraceCursor{Field: "start_time", Value: 1767268801000000}, `{"id":"01J00000000000000000000001","c":{"start_time":{"f":"start_time","v":1767268801000000}}}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			page := TracePageCursor{ID: "01J00000000000000000000001", Cursors: map[string]TraceCursor{test.cursor.Field: test.cursor}}
			encoded, err := page.Encode()
			require.NoError(t, err)
			require.Equal(t, base64.StdEncoding.EncodeToString([]byte(test.json)), encoded)
			var decoded TracePageCursor
			require.NoError(t, decoded.Decode(encoded))
			require.Equal(t, page, decoded)
			require.Zero(t, decoded.Find(test.cursor.Field).NanosecondRemainder)
		})
	}
}
