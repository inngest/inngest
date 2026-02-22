package queue

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/util"
	"github.com/stretchr/testify/require"
)

func TestBuildBacklogID_DefaultBacklog(t *testing.T) {
	fnID := uuid.New()

	// Build via BuildBacklogID
	got := BuildBacklogID(fnID, false, nil, nil)

	// Build via ItemBacklog
	item := QueueItem{
		FunctionID: fnID,
		Data: Item{
			Kind: KindEdge,
		},
	}
	expected := ItemBacklog(context.Background(), item)

	require.Equal(t, expected.BacklogID, got)
}

func TestBuildBacklogID_StartBacklog(t *testing.T) {
	fnID := uuid.New()

	got := BuildBacklogID(fnID, true, nil, nil)

	item := QueueItem{
		FunctionID: fnID,
		Data: Item{
			Kind: KindStart,
		},
	}
	expected := ItemBacklog(context.Background(), item)

	require.Equal(t, expected.BacklogID, got)
}

func TestBuildBacklogID_StartWithThrottleNoExpression(t *testing.T) {
	fnID := uuid.New()

	// When no throttle key expression is set, the throttle key is just hashID(fnID)
	// and the expression hash is xxhash("")
	got := BuildBacklogID(fnID, true, &ThrottleKeyInput{
		Expression:     "",
		EvaluatedValue: "",
	}, nil)

	throttleKey := HashID(context.Background(), fnID.String())
	item := QueueItem{
		FunctionID: fnID,
		Data: Item{
			Kind: KindStart,
			Throttle: &Throttle{
				Key:               throttleKey,
				KeyExpressionHash: util.XXHash(""),
			},
		},
	}
	expected := ItemBacklog(context.Background(), item)

	require.Equal(t, expected.BacklogID, got)
}

func TestBuildBacklogID_StartWithThrottleCustomExpression(t *testing.T) {
	fnID := uuid.New()

	got := BuildBacklogID(fnID, true, &ThrottleKeyInput{
		Expression:     "event.data.customerId",
		EvaluatedValue: "customer-123",
	}, nil)

	// Throttle key with custom expression: hashID(fnID) + "-" + hashID(evaluatedValue)
	throttleKey := HashID(context.Background(), fnID.String()) + "-" + HashID(context.Background(), "customer-123")
	item := QueueItem{
		FunctionID: fnID,
		Data: Item{
			Kind: KindStart,
			Throttle: &Throttle{
				Key:               throttleKey,
				KeyExpressionHash: util.XXHash("event.data.customerId"),
			},
		},
	}
	expected := ItemBacklog(context.Background(), item)

	require.Equal(t, expected.BacklogID, got)
}

func TestBuildBacklogID_ThrottleIgnoredForNonStart(t *testing.T) {
	fnID := uuid.New()

	// Throttle is only applied to start items
	got := BuildBacklogID(fnID, false, &ThrottleKeyInput{
		Expression:     "event.data.customerId",
		EvaluatedValue: "customer-123",
	}, nil)

	// ItemBacklog only applies throttle when i.Data.Kind == KindStart
	item := QueueItem{
		FunctionID: fnID,
		Data: Item{
			Kind: KindEdge,
			Throttle: &Throttle{
				Key:               HashID(context.Background(), fnID.String()) + "-" + HashID(context.Background(), "customer-123"),
				KeyExpressionHash: util.XXHash("event.data.customerId"),
			},
		},
	}
	expected := ItemBacklog(context.Background(), item)

	require.Equal(t, expected.BacklogID, got)
}

func TestBuildBacklogID_SingleConcurrencyKey(t *testing.T) {
	fnID := uuid.New()

	got := BuildBacklogID(fnID, false, nil, []ConcurrencyKeyInput{
		{
			Expression:     "event.data.customerId",
			EvaluatedValue: "customer-123",
			Scope:          enums.ConcurrencyScopeFn,
			ScopeID:        fnID,
		},
	})

	canonicalKey := util.ConcurrencyKey(enums.ConcurrencyScopeFn, fnID, "customer-123")
	item := QueueItem{
		FunctionID: fnID,
		Data: Item{
			Kind: KindEdge,
			CustomConcurrencyKeys: []state.CustomConcurrency{
				{
					Key:  canonicalKey,
					Hash: util.XXHash("event.data.customerId"),
				},
			},
		},
	}
	expected := ItemBacklog(context.Background(), item)

	require.Equal(t, expected.BacklogID, got)
}

func TestBuildBacklogID_TwoConcurrencyKeys(t *testing.T) {
	fnID := uuid.New()
	acctID := uuid.New()

	got := BuildBacklogID(fnID, false, nil, []ConcurrencyKeyInput{
		{
			Expression:     "event.data.customerId",
			EvaluatedValue: "customer-123",
			Scope:          enums.ConcurrencyScopeFn,
			ScopeID:        fnID,
		},
		{
			Expression:     "event.data.region",
			EvaluatedValue: "us-east-1",
			Scope:          enums.ConcurrencyScopeAccount,
			ScopeID:        acctID,
		},
	})

	canonicalKey1 := util.ConcurrencyKey(enums.ConcurrencyScopeFn, fnID, "customer-123")
	canonicalKey2 := util.ConcurrencyKey(enums.ConcurrencyScopeAccount, acctID, "us-east-1")
	item := QueueItem{
		FunctionID: fnID,
		Data: Item{
			Kind: KindEdge,
			CustomConcurrencyKeys: []state.CustomConcurrency{
				{
					Key:  canonicalKey1,
					Hash: util.XXHash("event.data.customerId"),
				},
				{
					Key:  canonicalKey2,
					Hash: util.XXHash("event.data.region"),
				},
			},
		},
	}
	expected := ItemBacklog(context.Background(), item)

	require.Equal(t, expected.BacklogID, got)
}

func TestBuildBacklogID_StartWithThrottleAndConcurrencyKeys(t *testing.T) {
	fnID := uuid.New()
	envID := uuid.New()

	got := BuildBacklogID(fnID, true, &ThrottleKeyInput{
		Expression:     "event.data.org",
		EvaluatedValue: "org-42",
	}, []ConcurrencyKeyInput{
		{
			Expression:     "event.data.userId",
			EvaluatedValue: "user-99",
			Scope:          enums.ConcurrencyScopeEnv,
			ScopeID:        envID,
		},
	})

	throttleKey := HashID(context.Background(), fnID.String()) + "-" + HashID(context.Background(), "org-42")
	canonicalKey := util.ConcurrencyKey(enums.ConcurrencyScopeEnv, envID, "user-99")
	item := QueueItem{
		FunctionID: fnID,
		Data: Item{
			Kind: KindStart,
			Throttle: &Throttle{
				Key:               throttleKey,
				KeyExpressionHash: util.XXHash("event.data.org"),
			},
			CustomConcurrencyKeys: []state.CustomConcurrency{
				{
					Key:  canonicalKey,
					Hash: util.XXHash("event.data.userId"),
				},
			},
		},
	}
	expected := ItemBacklog(context.Background(), item)

	require.Equal(t, expected.BacklogID, got)
}

func TestBuildBacklogID_ConcurrencyKeyScopes(t *testing.T) {
	fnID := uuid.New()
	envID := uuid.New()
	acctID := uuid.New()

	tests := []struct {
		name    string
		scope   enums.ConcurrencyScope
		scopeID uuid.UUID
	}{
		{"function scope", enums.ConcurrencyScopeFn, fnID},
		{"env scope", enums.ConcurrencyScopeEnv, envID},
		{"account scope", enums.ConcurrencyScopeAccount, acctID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildBacklogID(fnID, false, nil, []ConcurrencyKeyInput{
				{
					Expression:     "event.data.key",
					EvaluatedValue: "value",
					Scope:          tt.scope,
					ScopeID:        tt.scopeID,
				},
			})

			canonicalKey := util.ConcurrencyKey(tt.scope, tt.scopeID, "value")
			item := QueueItem{
				FunctionID: fnID,
				Data: Item{
					Kind: KindEdge,
					CustomConcurrencyKeys: []state.CustomConcurrency{
						{
							Key:  canonicalKey,
							Hash: util.XXHash("event.data.key"),
						},
					},
				},
			}
			expected := ItemBacklog(context.Background(), item)

			require.Equal(t, expected.BacklogID, got)
		})
	}
}

func TestBuildBacklogID_EmptyExpressionConcurrencyKey(t *testing.T) {
	fnID := uuid.New()

	// When expression is empty, the expression hash is xxhash("")
	got := BuildBacklogID(fnID, false, nil, []ConcurrencyKeyInput{
		{
			Expression:     "",
			EvaluatedValue: "some-value",
			Scope:          enums.ConcurrencyScopeFn,
			ScopeID:        fnID,
		},
	})

	canonicalKey := util.ConcurrencyKey(enums.ConcurrencyScopeFn, fnID, "some-value")
	item := QueueItem{
		FunctionID: fnID,
		Data: Item{
			Kind: KindEdge,
			CustomConcurrencyKeys: []state.CustomConcurrency{
				{
					Key:  canonicalKey,
					Hash: util.XXHash(""),
				},
			},
		},
	}
	expected := ItemBacklog(context.Background(), item)

	require.Equal(t, expected.BacklogID, got)
}

func TestBuildBacklogID_Deterministic(t *testing.T) {
	fnID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	// Same inputs should always produce the same output
	input := func() string {
		return BuildBacklogID(fnID, true, &ThrottleKeyInput{
			Expression:     "event.data.key",
			EvaluatedValue: "value",
		}, []ConcurrencyKeyInput{
			{
				Expression:     "event.data.a",
				EvaluatedValue: "val-a",
				Scope:          enums.ConcurrencyScopeFn,
				ScopeID:        fnID,
			},
		})
	}

	first := input()
	for i := 0; i < 10; i++ {
		require.Equal(t, first, input(), "BuildBacklogID should be deterministic")
	}
}
