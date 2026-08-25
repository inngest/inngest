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

// XXX: this package otherwise holds write-time extractors producing
// []metadata.Structured, but AISummaryMetadata is a read-time aggregate that
// is never written — it deliberately omits Op() and Scope(), so it cannot
// implement metadata.Structured or be passed to tracing.CreateMetadataSpan.
// It lives here because Cloud imports the counting rules; usage.go has the
// same shape already.

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
}

func (ms AISummaryMetadata) Kind() metadata.Kind {
	return KindInngestAISummary
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
// An OTel LLM call inside a step arrives twice with identical usage: summed
// into a step-scoped entry by the SDK's metadata processor, and re-extracted
// at extended_trace scope from the OTLP export. Counting extended_trace
// unconditionally therefore reports 2x, but excluding it always reads zero
// when it is the only carrier (calls outside any step, or sources with no
// metadata processor). Counting it only when the run reports no step-scoped
// AI closes that gap without double-counting. A mixed run, where one step
// reports via the SDK and another via raw OTel, still undercounts; fixing
// that needs the per-step correlation the read path currently discards.
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
// aggregate usage. It deliberately does not reuse AIMetadata: producers emit
// values as arbitrary JSON and may encode counts like cache_read_tokens as
// floats, which fail to unmarshal into *int64 and would drop the entry's
// tokens entirely. Loosening AIMetadata's types would not help — it is a
// producer-side struct that is never decoded from stored JSON, so this read
// path must tolerate float encodings regardless. Numerics are float64 here
// so both integer and fractional encodings parse.
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
	return b.AddCallJSON(raw)
}

// AddCallJSON folds one inngest.ai metadata entry's raw JSON values into the
// summary.
func (b *AISummaryBuilder) AddCallJSON(raw []byte) error {
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
