package metadata

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitLegacy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		kind   Kind
		values Values
		want   []KindValues
		wantOK bool
	}{
		{
			name: "legacy scores split per name",
			kind: KindInngestScore,
			values: Values{
				"b": json.RawMessage(`{"value":true}`),
				"a": json.RawMessage(`{"value":0.5}`),
			},
			want: []KindValues{
				{Kind: "inngest.score.a", Values: Values{"value": json.RawMessage(`0.5`)}},
				{Kind: "inngest.score.b", Values: Values{"value": json.RawMessage(`true`)}},
			},
			wantOK: true,
		},
		{
			name:   "legacy score that isn't an object is kept under value",
			kind:   KindInngestScore,
			values: Values{"a": json.RawMessage(`1`)},
			want: []KindValues{
				{Kind: "inngest.score.a", Values: Values{"value": json.RawMessage(`1`)}},
			},
			wantOK: true,
		},
		{
			name: "legacy warnings split per code",
			kind: KindInngestWarnings,
			values: Values{
				"size": json.RawMessage(`"too big"`),
				"auth": json.RawMessage(`"nope"`),
			},
			want: []KindValues{
				{Kind: "inngest.warning.auth", Values: Values{"auth": json.RawMessage(`"nope"`)}},
				{Kind: "inngest.warning.size", Values: Values{"size": json.RawMessage(`"too big"`)}},
			},
			wantOK: true,
		},
		{
			name:   "empty legacy write has nothing to write",
			kind:   KindInngestScore,
			values: Values{},
			want:   []KindValues{},
			wantOK: true,
		},
		{
			name:   "per name score kind isn't split",
			kind:   ScoreKind("a"),
			values: Values{"value": json.RawMessage(`1`)},
		},
		{
			name:   "other kinds aren't split",
			kind:   "userland.foo",
			values: Values{"a": json.RawMessage(`1`), "b": json.RawMessage(`2`)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := SplitLegacy(tt.kind, tt.values)
			require.Equal(t, tt.wantOK, ok)
			require.Equal(t, tt.want, got)
		})
	}
}
