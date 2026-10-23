package statev2

import (
	"testing"

	"google.golang.org/protobuf/proto"
)

func TestLookupIdempotencyResponseRoundTrip(t *testing.T) {
	for _, tt := range []struct {
		name  string
		entry *IdempotencyEntry
	}{
		{name: "miss"},
		{name: "existing run", entry: &IdempotencyEntry{RunId: "01ARZ3NDEKTSV4RRFFQ69G5FAV"}},
		{name: "tombstone", entry: &IdempotencyEntry{RunId: "01ARZ3NDEKTSV4RRFFQ69G5FAV", IsTombstone: true}},
		{name: "empty tombstone", entry: &IdempotencyEntry{IsTombstone: true}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			want := &LookupIdempotencyResponse{Entry: tt.entry}
			wire, err := proto.Marshal(want)
			if err != nil {
				t.Fatal(err)
			}
			got := &LookupIdempotencyResponse{}
			if err := proto.Unmarshal(wire, got); err != nil {
				t.Fatal(err)
			}
			if !proto.Equal(want, got) {
				t.Fatalf("round trip changed response: got %v, want %v", got, want)
			}
		})
	}
}
