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

// concurrencyTraceEventInput provides a read-only view of event data for
// concurrency evaluation. It reuses decoded data when available and otherwise
// decodes raw JSON lazily, at most once.
type concurrencyTraceEventInput struct {
	decoded map[string]any
	raw     json.RawMessage
	loaded  bool
}

func (i *concurrencyTraceEventInput) load() map[string]any {
	if i.decoded != nil || i.loaded {
		return i.decoded
	}
	i.loaded = true

	var evt event.Event
	if json.Unmarshal(i.raw, &evt) == nil {
		i.decoded = evt.Map()
	}
	return i.decoded
}

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
func customConcurrencyTraceKeys(ctx context.Context, fn *inngest.Function, keys []state.CustomConcurrency, eventInput concurrencyTraceEventInput) []meta.CustomConcurrencyKey {
	if fn == nil || fn.Concurrency == nil || len(keys) == 0 {
		return nil
	}

	var result []meta.CustomConcurrencyKey
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
				input := eventInput.load()
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
