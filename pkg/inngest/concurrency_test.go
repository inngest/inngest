package inngest

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/stretchr/testify/require"
)

func TestConcurrencyLimits_Unmarshal(t *testing.T) {
	tests := []struct {
		input    []byte
		expected ConcurrencyLimits
	}{
		{
			input:    []byte("null"),
			expected: ConcurrencyLimits{},
		},
		{
			input: []byte("{}"),
			expected: ConcurrencyLimits{
				Step: []StepConcurrency{
					{
						Limit: 0,
						Key:   nil,
					},
				},
				Limits: []StepConcurrency{
					{
						Limit: 0,
						Key:   nil,
					},
				},
			},
		},
		{
			input: []byte(`{"key":"what"}`),
			expected: ConcurrencyLimits{
				Step: []StepConcurrency{
					{
						Limit: 0,
						Key:   strptr("what"),
						Hash:  hashConcurrencyKey("what"),
					},
				},
				Limits: []StepConcurrency{
					{
						Limit: 0,
						Key:   strptr("what"),
						Hash:  hashConcurrencyKey("what"),
					},
				},
			},
		},
		{
			input: []byte(`{"key":"what", "limit": 10}`),
			expected: ConcurrencyLimits{
				Step: []StepConcurrency{
					{
						Limit: 10,
						Key:   strptr("what"),
						Hash:  hashConcurrencyKey("what"),
					},
				},
				Limits: []StepConcurrency{
					{
						Limit: 10,
						Key:   strptr("what"),
						Hash:  hashConcurrencyKey("what"),
					},
				},
			},
		},
		{
			input: []byte(`[{"key":"what", "limit": 10}]`),
			expected: ConcurrencyLimits{
				Step: []StepConcurrency{
					{
						Limit: 10,
						Key:   strptr("what"),
						Hash:  hashConcurrencyKey("what"),
					},
				},
				Limits: []StepConcurrency{
					{
						Limit: 10,
						Key:   strptr("what"),
						Hash:  hashConcurrencyKey("what"),
					},
				},
			},
		},
		{
			input: []byte(`[{"key":"what", "limit": 10, "scope":"account"}]`),
			expected: ConcurrencyLimits{
				Step: []StepConcurrency{
					{
						Limit: 10,
						Key:   strptr("what"),
						Scope: enums.ConcurrencyScopeAccount,
						Hash:  hashConcurrencyKey("what"),
					},
				},
				Limits: []StepConcurrency{
					{
						Limit: 10,
						Key:   strptr("what"),
						Scope: enums.ConcurrencyScopeAccount,
						Hash:  hashConcurrencyKey("what"),
					},
				},
			},
		},
		{
			input: []byte(`[{"key":"what", "limit": 25, "scope": "account"}, {"key": "event.data.foo", "limit": 10}]`),
			expected: ConcurrencyLimits{
				Step: []StepConcurrency{
					// ordered low to high
					{
						Limit: 10,
						Key:   strptr("event.data.foo"),
						Hash:  hashConcurrencyKey("event.data.foo"),
					},
					{
						Limit: 25,
						Key:   strptr("what"),
						Scope: enums.ConcurrencyScopeAccount,
						Hash:  hashConcurrencyKey("what"),
					},
				},
				Limits: []StepConcurrency{
					// ordered low to high
					{
						Limit: 10,
						Key:   strptr("event.data.foo"),
						Hash:  hashConcurrencyKey("event.data.foo"),
					},
					{
						Limit: 25,
						Key:   strptr("what"),
						Scope: enums.ConcurrencyScopeAccount,
						Hash:  hashConcurrencyKey("what"),
					},
				},
			},
		},
		// New format: {"fn": [...], "step": [...]}
		{
			input: []byte(`{"fn": [{"limit": 5}], "step": [{"limit": 10, "key": "event.data.userId", "scope": "fn"}]}`),
			expected: ConcurrencyLimits{
				Fn: []FnConcurrency{
					{
						Limit: 5,
					},
				},
				Step: []StepConcurrency{
					{
						Limit: 10,
						Key:   strptr("event.data.userId"),
						Hash:  hashConcurrencyKey("event.data.userId"),
					},
				},
				Limits: []StepConcurrency{
					{
						Limit: 10,
						Key:   strptr("event.data.userId"),
						Hash:  hashConcurrencyKey("event.data.userId"),
					},
				},
			},
		},
		// New format: fn only
		{
			input: []byte(`{"fn": [{"limit": 3, "key": "event.data.customerId"}]}`),
			expected: ConcurrencyLimits{
				Fn: []FnConcurrency{
					{
						Limit: 3,
						Key:   strptr("event.data.customerId"),
						Hash:  hashConcurrencyKey("event.data.customerId"),
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			actual := &ConcurrencyLimits{}
			err := json.Unmarshal(test.input, actual)
			require.NoError(t, err, test)
			require.EqualValues(t, test.expected, *actual)
		})
	}
}

func TestConcurrencyEvaluate(t *testing.T) {
	uuidA, uuidB := uuid.MustParse("c866c44e-d49a-4577-ac1d-471ae350dead"), uuid.MustParse("a34ea1b0-b544-4738-8ac8-b6856bc506e8")
	hashA, hashB := strconv.FormatUint(xxhash.Sum64String("1"), 36), strconv.FormatUint(xxhash.Sum64String("99"), 36)

	tests := []struct {
		limit    StepConcurrency
		scopeID  uuid.UUID
		input    map[string]any
		expected string
	}{
		{
			limit: StepConcurrency{
				Limit: 10,
				Key:   strptr("event.data.user_id"),
				Scope: enums.ConcurrencyScopeFn,
			},
			scopeID: uuidA,
			input: event.Event{
				Data: map[string]any{
					"user_id": "1",
				},
			}.Map(),
			expected: fmt.Sprintf("f:%s:%s", uuidA, hashA),
		},
		// Change the ID, expect a different output
		{
			limit: StepConcurrency{
				Limit: 10,
				Key:   strptr("event.data.user_id"),
				Scope: enums.ConcurrencyScopeFn,
			},
			scopeID: uuidB,
			input: event.Event{
				Data: map[string]any{
					"user_id": "1",
				},
			}.Map(),
			expected: fmt.Sprintf("f:%s:%s", uuidB, hashA),
		},
		// Chagne the input
		{
			limit: StepConcurrency{
				Limit: 10,
				Key:   strptr("event.data.user_id"),
				Scope: enums.ConcurrencyScopeFn,
			},
			scopeID: uuidA,
			input: event.Event{
				Data: map[string]any{
					"user_id": "99",
				},
			}.Map(),
			expected: fmt.Sprintf("f:%s:%s", uuidA, hashB),
		},
		// Chagne the scope
		{
			limit: StepConcurrency{
				Limit: 10,
				Key:   strptr("event.data.user_id"),
				Scope: enums.ConcurrencyScopeAccount,
			},
			scopeID: uuidA,
			input: event.Event{
				Data: map[string]any{
					"user_id": "99",
				},
			}.Map(),
			expected: fmt.Sprintf("a:%s:%s", uuidA, hashB),
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%#v", test), func(t *testing.T) {
			actual := test.limit.EvaluatedKey(context.Background(), test.scopeID, test.input)
			require.EqualValues(t, test.expected, actual, test)
		})
	}
}
