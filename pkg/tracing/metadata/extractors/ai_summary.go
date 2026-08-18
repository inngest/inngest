package extractors

import (
	"cmp"
	"encoding/json"
	"maps"
	"math"
	"slices"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/tracing/metadata"
)

// costPrecision is the inverse granularity summary costs are rounded to:
// 1e-8 dollars. Rounding at output makes the sum independent of the order
// entries were folded in — float addition is not associative — except for
// sums within a few ULPs of a decimal halfway boundary, which real per-token
// rates never produce. Future cost representations must keep ≤1e-8
// granularity or rounded values will visibly change.
const costPrecision = 1e8

// RoundCost canonicalizes a summed cost to costPrecision.
func RoundCost(v float64) float64 {
	return math.Round(v*costPrecision) / costPrecision
}

//tygo:generate
const (
	// KindInngestAISummary is the run-scoped rollup of all AI usage within a
	// run. It is synthesized every time a run's span tree is read and is never
	// persisted, so it can never double-count itself. It must never be added
	// to the allowedInngestKinds allowlist.
	KindInngestAISummary metadata.Kind = "inngest.ai.summary"
)

//tygo:generate
type AISummaryMetadata struct {
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
	TotalTokens  int64 `json:"total_tokens"`

	// Absent when no call reported them; cache tokens are summed raw and never
	// reconciled against InputTokens.
	CacheReadTokens     *int64 `json:"cache_read_tokens,omitempty"`
	CacheCreationTokens *int64 `json:"cache_creation_tokens,omitempty"`
	ReasoningTokens     *int64 `json:"reasoning_tokens,omitempty"`

	EstimatedCost *float64 `json:"estimated_cost,omitempty"`
	Models        []string `json:"models,omitempty"`
	Providers     []string `json:"providers,omitempty"`
	// Partial marks the summary as known-incomplete: usage from invoked child
	// runs is not folded in.
	Partial bool `json:"partial"`
}

func (ms AISummaryMetadata) Kind() metadata.Kind {
	return KindInngestAISummary
}

func (ms AISummaryMetadata) Scope() metadata.Scope {
	return enums.MetadataScopeRun
}

func (ms AISummaryMetadata) Serialize() (metadata.Values, error) {
	var rawMetadata metadata.Values
	err := rawMetadata.FromStruct(ms)
	if err != nil {
		return nil, err
	}

	return rawMetadata, nil
}

func AISummaryFromValues(values metadata.Values) (AISummaryMetadata, error) {
	var ms AISummaryMetadata
	raw, err := json.Marshal(values)
	if err != nil {
		return ms, err
	}
	err = json.Unmarshal(raw, &ms)
	return ms, err
}

// AIUsageStepScoped reports whether an inngest.ai entry at the given scope is
// executor-reported step usage. AIUsageEntryCounted needs to know whether a
// run has any such entry: only then can an extended-trace entry be the second
// report of a call already counted.
func AIUsageStepScoped(scope metadata.Scope) bool {
	return scope == enums.MetadataScopeStep || scope == enums.MetadataScopeStepAttempt
}

// AIUsageEntryCounted is the run-level definition of which inngest.ai entries
// count toward the run's AI summary, shared by OSS and Cloud. Step and legacy
// step-attempt entries are executor-reported usage and run-scoped entries are
// how users report out-of-step usage, so those always count.
//
// extended_trace is conditional in both directions. On the SDK path every
// OTel-derived LLM call made inside a step is reported twice with identical
// tokens and cost — once at step scope by the SDK's metadata processor, once
// at extended trace scope — so counting extended_trace unconditionally
// reports 2x. But a blanket exclusion reads zero for gen_ai usage that never
// reaches a counted scope at all: emitted outside any step, or from a source
// with no metadata processor, where the OTLP endpoint is the only carrier.
// Deciding per run — extended_trace counts only when the run reports no
// step-scoped AI (runHasStepScopedAI, per AIUsageStepScoped) — closes that
// gap without double-counting. The cost is that a mixed run — one step
// reporting through the SDK, another making a raw OTel LLM call — still
// undercounts; resolving that needs a per-step correlation key that only
// extended-trace metadata spans carry.
func AIUsageEntryCounted(scope metadata.Scope, runHasStepScopedAI bool) bool {
	switch scope {
	case enums.MetadataScopeStep, enums.MetadataScopeStepAttempt, enums.MetadataScopeRun:
		return true
	case enums.MetadataScopeExtendedTrace:
		return !runHasStepScopedAI
	default:
		return false
	}
}

// AISummaryBuilder accumulates inngest.ai metadata entries into a single
// AISummaryMetadata. Every counted entry is summed, including entries from
// retried attempts: spend on a retried step is real spend.
type AISummaryBuilder struct {
	sum     AISummaryMetadata
	cost    float64
	hasCost bool
	// emissions counts AddCall folds. It exists only so Empty() stays
	// meaningful — the summary deliberately exposes no call count, because the
	// SDK sums a step's calls into one emission before reporting, so no
	// emission count is a call count.
	emissions int64
	models    map[string]struct{}
	providers map[string]struct{}
	// Optional token counters, tracked separately from sum so a field no call
	// reported stays absent rather than summing to zero.
	cacheRead     optionalInt64
	cacheCreation optionalInt64
	reasoning     optionalInt64
}

// optionalInt64 accumulates a sum that is only emitted once at least one
// contributor supplied a value.
type optionalInt64 struct {
	value int64
	set   bool
}

func (o *optionalInt64) add(v int64) {
	o.value += v
	o.set = true
}

func (o optionalInt64) ptr() *int64 {
	if !o.set {
		return nil
	}
	v := o.value
	return &v
}

func NewAISummaryBuilder() *AISummaryBuilder {
	return &AISummaryBuilder{
		models:    map[string]struct{}{},
		providers: map[string]struct{}{},
	}
}

// aiUsageValues is the minimal projection of an inngest.ai entry needed to
// aggregate usage. It deliberately does not reuse AIMetadata: that struct
// types optional counts like cache_read_tokens as *int64, but producers can
// emit them as floats, so unmarshalling the full struct fails and would drop
// the entry's tokens entirely. Numerics are float64 here so both integer and
// fractional encodings parse.
type aiUsageValues struct {
	InputTokens         float64  `json:"input_tokens"`
	OutputTokens        float64  `json:"output_tokens"`
	TotalTokens         *float64 `json:"total_tokens"`
	CacheReadTokens     *float64 `json:"cache_read_tokens"`
	CacheCreationTokens *float64 `json:"cache_creation_tokens"`
	ReasoningTokens     *float64 `json:"reasoning_tokens"`
	EstimatedCost       *float64 `json:"estimated_cost"`
	Provider            string   `json:"provider"`
	RequestModel        string   `json:"request_model"`
	ResponseModel       string   `json:"response_model"`
}

// AddCall folds one inngest.ai metadata entry's values into the summary.
func (b *AISummaryBuilder) AddCall(values metadata.Values) error {
	raw, err := json.Marshal(values)
	if err != nil {
		return err
	}
	var m aiUsageValues
	if err := json.Unmarshal(raw, &m); err != nil {
		return err
	}

	b.sum.InputTokens += int64(m.InputTokens)
	b.sum.OutputTokens += int64(m.OutputTokens)
	if m.TotalTokens != nil {
		b.sum.TotalTokens += int64(*m.TotalTokens)
	} else {
		b.sum.TotalTokens += int64(m.InputTokens) + int64(m.OutputTokens)
	}
	if m.CacheReadTokens != nil {
		b.cacheRead.add(int64(*m.CacheReadTokens))
	}
	if m.CacheCreationTokens != nil {
		b.cacheCreation.add(int64(*m.CacheCreationTokens))
	}
	if m.ReasoningTokens != nil {
		b.reasoning.add(int64(*m.ReasoningTokens))
	}
	if m.EstimatedCost != nil {
		b.cost += *m.EstimatedCost
		b.hasCost = true
	}
	// Mirrors COALESCE(response_model, request_model), the label Cloud's AI
	// dashboards group every model-scoped metric by. Recording both would
	// surface request aliases that are never a dashboard category.
	if model := cmp.Or(m.ResponseModel, m.RequestModel); model != "" {
		b.models[model] = struct{}{}
	}
	if m.Provider != "" {
		b.providers[m.Provider] = struct{}{}
	}
	b.emissions++

	return nil
}

func (b *AISummaryBuilder) MarkPartial() {
	b.sum.Partial = true
}

// Empty reports whether nothing has been accumulated. The partial flag is
// deliberately ignored: an empty-but-partial summary still carries no usage.
func (b *AISummaryBuilder) Empty() bool {
	return b.emissions == 0 && !b.hasCost
}

func (b *AISummaryBuilder) Summary() AISummaryMetadata {
	out := b.sum
	out.CacheReadTokens = b.cacheRead.ptr()
	out.CacheCreationTokens = b.cacheCreation.ptr()
	out.ReasoningTokens = b.reasoning.ptr()
	if b.hasCost {
		cost := RoundCost(b.cost)
		out.EstimatedCost = &cost
	}
	if len(b.models) > 0 {
		out.Models = slices.Sorted(maps.Keys(b.models))
	}
	if len(b.providers) > 0 {
		out.Providers = slices.Sorted(maps.Keys(b.providers))
	}
	return out
}
