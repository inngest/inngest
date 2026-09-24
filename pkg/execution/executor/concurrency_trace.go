package executor

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/util"
)

const maxConcurrencyTraceKeyChars = 512

func truncateConcurrencyTraceKey(value string) (string, bool) {
	chars := 0
	for i := range value {
		if chars == maxConcurrencyTraceKeyChars {
			return value[:i], true
		}
		chars++
	}
	return value, false
}

// customConcurrencyTraceKeys only pairs an expression with a queue key when
// both hashes agree. A function may have changed since the item was queued.
func customConcurrencyTraceKeys(ctx context.Context, fn *inngest.Function, keys []state.CustomConcurrency, rawEvent json.RawMessage) []meta.CustomConcurrencyKey {
	if fn == nil || fn.Concurrency == nil || len(keys) == 0 {
		return nil
	}

	var result []meta.CustomConcurrencyKey
	var input map[string]any
	inputLoaded := false
	for _, key := range keys {
		if key.Validate() != nil {
			continue
		}
		scope, _, valueHash, err := key.ParseKey()
		if err != nil {
			continue
		}

		for _, limit := range fn.Concurrency.Limits {
			if limit.Key == nil || limit.Scope != scope || limit.Hash != key.Hash {
				continue
			}
			value := key.UnhashedEvaluatedKeyValue
			if util.XXHash(value) != valueHash {
				if !inputLoaded {
					inputLoaded = true
					var evt event.Event
					if json.Unmarshal(rawEvent, &evt) == nil {
						input = evt.Map()
					}
				}
				if input == nil {
					continue
				}
				value = limit.Evaluate(ctx, input)
				if util.XXHash(value) != valueHash {
					continue
				}
			}
			expression, expressionTruncated := truncateConcurrencyTraceKey(*limit.Key)
			value, valueTruncated := truncateConcurrencyTraceKey(value)
			result = append(result, meta.CustomConcurrencyKey{
				Scope:               strings.ToLower(scope.String()),
				Expression:          expression,
				Value:               value,
				ExpressionTruncated: expressionTruncated,
				ValueTruncated:      valueTruncated,
			})
			break
		}
	}
	return result
}
