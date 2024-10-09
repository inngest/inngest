package redis_state

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSerializeGuaranteedCapacity(t *testing.T) {
	cases := []struct {
		name string
		got  GuaranteedCapacity
		want json.RawMessage
	}{
		{
			name: "leased gc",
			got: GuaranteedCapacity{
				Scope:              enums.GuaranteedCapacityScopeAccount,
				AccountID:          uuid.MustParse("c06e5559-74fd-4404-8754-d06b6f342d10"),
				Priority:           0,
				GuaranteedCapacity: 1,
				Leases:             []ulid.ULID{ulid.MustParse("01J9S37W106HK23TGK5MNPY09J")},
			},
			want: json.RawMessage(`{"s":"Account","a":"c06e5559-74fd-4404-8754-d06b6f342d10","p":0,"gc":1,"leases":["01J9S37W106HK23TGK5MNPY09J"]}`),
		},
		{
			name: "non-leased gc",
			got: GuaranteedCapacity{
				Scope:              enums.GuaranteedCapacityScopeAccount,
				AccountID:          uuid.MustParse("c06e5559-74fd-4404-8754-d06b6f342d10"),
				Priority:           0,
				GuaranteedCapacity: 1,
			},
			want: json.RawMessage(`{"s":"Account","a":"c06e5559-74fd-4404-8754-d06b6f342d10","p":0,"gc":1,"leases":null}`),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := json.Marshal(tc.got)
			require.NoError(t, err)
			require.JSONEq(t, string(tc.want), string(got))
		})
	}
}

func TestDeserializeGuaranteedCapacity(t *testing.T) {
	cases := []struct {
		name string
		got  json.RawMessage
		want GuaranteedCapacity
	}{
		{
			name: "empty gc",
			got:  json.RawMessage(`{}`),
			want: GuaranteedCapacity{},
		},
		{
			name: "gc with empty leases obj",
			got: json.RawMessage(`{
				"s": "Account",
				"a": "c06e5559-74fd-4404-8754-d06b6f342d10",
				"p": 0,
				"gc": 1,
				"leases": {}
			}`),
			want: GuaranteedCapacity{
				Scope:              enums.GuaranteedCapacityScopeAccount,
				AccountID:          uuid.MustParse("c06e5559-74fd-4404-8754-d06b6f342d10"),
				Priority:           0,
				GuaranteedCapacity: 1,
				Leases:             nil,
			},
		},
		{
			name: "gc with empty leases obj",
			got: json.RawMessage(`{
				"s": "Account",
				"a": "c06e5559-74fd-4404-8754-d06b6f342d10",
				"p": 0,
				"gc": 1,
				"leases": []
			}`),
			want: GuaranteedCapacity{
				Scope:              enums.GuaranteedCapacityScopeAccount,
				AccountID:          uuid.MustParse("c06e5559-74fd-4404-8754-d06b6f342d10"),
				Priority:           0,
				GuaranteedCapacity: 1,
				Leases:             []ulid.ULID{},
			},
		},
		{
			name: "gc with non-empty leases obj",
			got: json.RawMessage(`{
				"s": "Account",
				"a": "c06e5559-74fd-4404-8754-d06b6f342d10",
				"p": 0,
				"gc": 2,
				"leases": ["01J9S37W106HK23TGK5MNPY09J", "01J9S37YW0HHABTVCJ7WNFAV5N"]
			}`),
			want: GuaranteedCapacity{
				Scope:              enums.GuaranteedCapacityScopeAccount,
				AccountID:          uuid.MustParse("c06e5559-74fd-4404-8754-d06b6f342d10"),
				Priority:           0,
				GuaranteedCapacity: 2,
				Leases: []ulid.ULID{
					ulid.MustParse("01J9S37W106HK23TGK5MNPY09J"),
					ulid.MustParse("01J9S37YW0HHABTVCJ7WNFAV5N"),
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got GuaranteedCapacity
			err := json.Unmarshal(tc.got, &got)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
