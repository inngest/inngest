package executor

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/util"
	"github.com/stretchr/testify/require"
)

func TestCustomConcurrencyTraceKeys(t *testing.T) {
	fnID := uuid.New()
	envID := uuid.New()
	fnExpr := "event.data.customer"
	envExpr := "event.data.region"
	fnHash := util.XXHash(fnExpr)
	envHash := util.XXHash(envExpr)
	fn := &inngest.Function{Concurrency: &inngest.ConcurrencyLimits{Limits: []inngest.StepConcurrency{
		{Key: &fnExpr, Hash: fnHash, Scope: enums.ConcurrencyScopeFn, Limit: 2},
		{Key: &envExpr, Hash: envHash, Scope: enums.ConcurrencyScopeEnv, Limit: 3},
	}}}

	key := func(scope enums.ConcurrencyScope, id uuid.UUID, hash, value string) state.CustomConcurrency {
		return state.CustomConcurrency{
			Key:                       util.ConcurrencyKey(scope, id, value),
			Hash:                      hash,
			UnhashedEvaluatedKeyValue: value,
		}
	}
	validFn := key(enums.ConcurrencyScopeFn, fnID, fnHash, "customer-a")
	validEnv := key(enums.ConcurrencyScopeEnv, envID, envHash, "eu")
	missingValue := validFn
	missingValue.UnhashedEvaluatedKeyValue = ""
	wrongExpression := validFn
	wrongExpression.Hash = "old-expression"
	wrongScope := key(enums.ConcurrencyScopeEnv, envID, fnHash, "customer-a")
	emptyValue := key(enums.ConcurrencyScopeFn, fnID, fnHash, "")
	rawEvent, err := json.Marshal(event.Event{Data: map[string]any{"customer": "customer-a"}})
	require.NoError(t, err)
	otherEvent, err := json.Marshal(event.Event{Data: map[string]any{"customer": "customer-b"}})
	require.NoError(t, err)

	tests := []struct {
		name  string
		keys  []state.CustomConcurrency
		event json.RawMessage
		want  []meta.CustomConcurrencyKey
	}{
		{name: "no keys"},
		{name: "one key", keys: []state.CustomConcurrency{validFn}, want: []meta.CustomConcurrencyKey{
			{Scope: "fn", Expression: fnExpr, Value: "customer-a"},
		}},
		{name: "two keys", keys: []state.CustomConcurrency{validFn, validEnv}, want: []meta.CustomConcurrencyKey{
			{Scope: "fn", Expression: fnExpr, Value: "customer-a"},
			{Scope: "env", Expression: envExpr, Value: "eu"},
		}},
		{name: "missing raw value", keys: []state.CustomConcurrency{missingValue}},
		{name: "recover missing raw value", keys: []state.CustomConcurrency{missingValue}, event: rawEvent, want: []meta.CustomConcurrencyKey{
			{Scope: "fn", Expression: fnExpr, Value: "customer-a"},
		}},
		{name: "reject mismatched event", keys: []state.CustomConcurrency{missingValue}, event: otherEvent},
		{name: "changed expression", keys: []state.CustomConcurrency{wrongExpression}},
		{name: "different scope", keys: []state.CustomConcurrency{wrongScope}},
		{name: "malformed key", keys: []state.CustomConcurrency{{Key: "not-a-key", Hash: fnHash, UnhashedEvaluatedKeyValue: "customer-a"}}},
		{name: "empty evaluated value", keys: []state.CustomConcurrency{emptyValue}, want: []meta.CustomConcurrencyKey{
			{Scope: "fn", Expression: fnExpr, Value: ""},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, customConcurrencyTraceKeys(context.Background(), fn, tt.keys, tt.event))
		})
	}
}
