package tracing

import (
	"testing"

	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/util"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
)

func TestIsPairedTrailing(t *testing.T) {
	key := meta.Attrs.IsPairedTrailing.Key()

	tests := []struct {
		name  string
		attrs []attribute.KeyValue
		want  bool
	}{
		{
			name:  "empty",
			attrs: nil,
			want:  false,
		},
		{
			name:  "flag present and true",
			attrs: []attribute.KeyValue{attribute.Bool(key, true)},
			want:  true,
		},
		{
			name:  "flag present and false",
			attrs: []attribute.KeyValue{attribute.Bool(key, false)},
			want:  false,
		},
		{
			name:  "flag key present but wrong type",
			attrs: []attribute.KeyValue{attribute.String(key, "true")},
			want:  false,
		},
		{
			name: "only unrelated keys",
			attrs: []attribute.KeyValue{
				attribute.Bool("some.other.flag", true),
				attribute.String(key+".suffix", "true"),
			},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, isPairedTrailing(tc.attrs))
		})
	}
}

// TestIsPairedTrailingSerializeRoundTrip guards the writer→reader contract
// across representations. The flag is written into a SerializableAttrs (as the
// checkpoint layer does via meta), but isPairedTrailing reads it from the
// []attribute.KeyValue produced by Serialize() — what OnStart actually sees. A
// key or serializer drift between the two would pass every isolated test yet
// break this round-trip.
func TestIsPairedTrailingSerializeRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		attrs *meta.SerializableAttrs
		want  bool
	}{
		{
			name:  "flag true survives serialization",
			attrs: meta.NewAttrSet(meta.Attr(meta.Attrs.IsPairedTrailing, util.ToPtr(true))),
			want:  true,
		},
		{
			name:  "flag false survives serialization",
			attrs: meta.NewAttrSet(meta.Attr(meta.Attrs.IsPairedTrailing, util.ToPtr(false))),
			want:  false,
		},
		{
			name:  "absent flag",
			attrs: meta.NewAttrSet(),
			want:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, isPairedTrailing(tc.attrs.Serialize()))
		})
	}
}
