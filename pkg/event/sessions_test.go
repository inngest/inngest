package event

import (
	"context"
	"testing"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/stretchr/testify/require"
)

// TestResolveSessions exercises the public merge entrypoint end to end: with no
// propagated layer the manual sessions pass through untouched; otherwise manual
// (user-set) keys win, remaining slots up to MaxEventSessions fill from the
// propagated layer in UTF-8 byte order (matching the SDK's aggregateMetadata and
// run-level normalizeRunSessions), and the propagated layer is always cleared so
// it never persists downstream.
//
// The byte-order cases — including the non-BMP one — are the cross-language
// parity vectors; the SDK verifies the same selection in
// packages/inngest/src/helpers/sessions.parity.test.ts. Keep the two in sync.
func TestResolveSessions(t *testing.T) {
	cases := []struct {
		name       string
		manual     Sessions
		propagated Sessions
		want       Sessions
	}{
		{
			name: "both layers empty stays nil",
			want: nil,
		},
		{
			name:   "manual only passes through (no propagated layer)",
			manual: Sessions{"a": "1"},
			want:   Sessions{"a": "1"},
		},
		{
			name:       "propagated only passes through",
			propagated: Sessions{"a": "1"},
			want:       Sessions{"a": "1"},
		},
		{
			name:       "manual wins on key collision, disjoint propagated fills",
			manual:     Sessions{"conv_id": "manual"},
			propagated: Sessions{"conv_id": "propagated", "org_id": "42"},
			want:       Sessions{"conv_id": "manual", "org_id": "42"},
		},
		{
			name:       "disjoint keys are unioned",
			manual:     Sessions{"b": "2"},
			propagated: Sessions{"a": "1"},
			want:       Sessions{"a": "1", "b": "2"},
		},
		{
			name:       "manual reserved first, propagated fills in byte order, overflow dropped",
			manual:     Sessions{"f": "9"},
			propagated: Sessions{"a": "1", "b": "1", "c": "1", "d": "1", "e": "1"},
			// f reserved; a,b,c,d fill; e is byte-last, dropped at the cap.
			want: Sessions{"f": "9", "a": "1", "b": "1", "c": "1", "d": "1"},
		},
		{
			name:       "full manual layer evicts all propagated",
			manual:     Sessions{"f": "9", "g": "9", "h": "9", "i": "9", "j": "9"},
			propagated: Sessions{"a": "1", "b": "1", "c": "1", "d": "1", "e": "1"},
			want:       Sessions{"f": "9", "g": "9", "h": "9", "i": "9", "j": "9"},
		},
		{
			name:       "shadowed propagated key consumes no slot",
			manual:     Sessions{"a": "2", "f": "9"},
			propagated: Sessions{"a": "1", "b": "1", "c": "1", "d": "1", "e": "1"},
			// a,f reserved (a shadows propagated a, freeing a slot); b,c,d fill.
			want: Sessions{"a": "2", "f": "9", "b": "1", "c": "1", "d": "1"},
		},
		{
			name: "oversized manual layer is preserved for post-merge Validate to reject",
			// Manual alone exceeds the cap; it is never truncated here and no
			// propagated key is added.
			manual:     Sessions{"a": "1", "b": "1", "c": "1", "d": "1", "e": "1", "f": "1"},
			propagated: Sessions{"z": "1"},
			want:       Sessions{"a": "1", "b": "1", "c": "1", "d": "1", "e": "1", "f": "1"},
		},
		{
			name: "fill order is UTF-8 byte order, not locale/rune order",
			// 'Z' (0x5A) < 'a' (0x61) < 'z' (0x7A) < 'é' (0xC3 0xA9). Six keys,
			// cap 5 → 'é' is dropped.
			propagated: Sessions{"Z": "1", "a": "1", "b": "1", "c": "1", "z": "1", "é": "1"},
			want:       Sessions{"Z": "1", "a": "1", "b": "1", "c": "1", "z": "1"},
		},
		{
			name: "non-BMP truncation follows UTF-8 bytes, not UTF-16 code units",
			// U+E000 encodes as 0xEE.. and 🍎 (U+1F34E) as 0xF0.., so by UTF-8 bytes
			// U+E000 sorts first and 🍎 is dropped at the cap. A UTF-16 code-unit
			// sort (JavaScript's default) would order them the other way — this
			// vector pins the SDK/server agreement.
			propagated: Sessions{
				"a": "1", "b": "1", "c": "1", "d": "1", "\ue000": "5", "\U0001F34E": "6",
			},
			want: Sessions{"a": "1", "b": "1", "c": "1", "d": "1", "\ue000": "5"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := require.New(t)
			m := &EventMeta{Sessions: tc.manual, PropagatedSessions: tc.propagated}

			m.ResolveSessions()

			r.Equal(tc.want, m.Sessions)
			r.Nil(m.PropagatedSessions, "propagated layer is always cleared")
		})
	}
}

// TestResolveSessionsThenValidate ensures that we resolve sessions before
// validation.
func TestResolveSessionsThenValidate(t *testing.T) {
	r := require.New(t)
	manual := Sessions{"a": "1", "b": "1", "c": "1"}               // 3
	propagated := Sessions{"d": "1", "e": "1", "f": "1", "g": "1"} // 4 → raw union 7 > cap
	r.Greater(len(manual)+len(propagated), consts.MaxEventSessions)

	evt := Event{
		Name: "test/session-prop",
		Meta: EventMeta{Sessions: manual, PropagatedSessions: propagated},
	}
	evt.Meta.ResolveSessions()

	r.Len(evt.Meta.Sessions, consts.MaxEventSessions) // capped
	for k := range manual {
		r.Contains(evt.Meta.Sessions, k, "manual keys always survive the merge")
	}
	r.NoError(evt.Validate(context.Background()))
}
