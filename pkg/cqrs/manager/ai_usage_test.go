package manager

import (
	"crypto/rand"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/tracing/metadata/extractors"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func metadataSpanAttrs(kind, scope, values string) []byte {
	return fmt.Appendf(nil,
		`{"_inngest.metadata.kind":%q,"_inngest.metadata.scope":%q,"_inngest.metadata.op":"merge","_inngest.metadata.values":%s}`,
		kind, scope, strconv.Quote(values),
	)
}

func findAISummary(t *testing.T, span *cqrs.OtelSpan) []*cqrs.SpanMetadata {
	t.Helper()
	var out []*cqrs.SpanMetadata
	for _, md := range span.Metadata {
		if md.Kind == extractors.KindInngestAISummary {
			out = append(out, md)
		}
	}
	return out
}

func TestCQRSAISummaryMetadata(t *testing.T) {
	runAttr := []byte(`{"_inngest.dynamic.status":"Running"}`)
	stepAttrs := func(id string, attempt int) []byte {
		return fmt.Appendf(nil, `{"_inngest.step.id":%q,"_inngest.step.attempt":%d}`, id, attempt)
	}

	t.Run("sums counted scopes, excludes extended_trace, strips spoofed summaries", func(t *testing.T) {
		cm, cleanup := initCQRS(t)
		defer cleanup()

		runID := ulid.MustNew(ulid.Now(), rand.Reader).String()
		spans := []testSpanFields{
			{DynamicSpanID: "root", Name: meta.SpanNameRun, Attributes: runAttr},
			{DynamicSpanID: "step1", ParentSpanID: "root", Name: meta.SpanNameStep, Attributes: stepAttrs("a", 0)},
			{DynamicSpanID: "step2", ParentSpanID: "root", Name: meta.SpanNameStep, Attributes: stepAttrs("b", 0)},
			// Executor-reported usage at legacy step_attempt scope.
			{DynamicSpanID: "md1", ParentSpanID: "step1", Name: meta.SpanNameMetadata, Attributes: metadataSpanAttrs(
				"inngest.ai", "step_attempt",
				`{"input_tokens":100,"output_tokens":20,"request_model":"gpt-4o","response_model":"gpt-4o-mini","estimated_cost":0.05}`,
			)},
			// Step-scoped usage with an explicit total that beats input+output.
			{DynamicSpanID: "md2", ParentSpanID: "step2", Name: meta.SpanNameMetadata, Attributes: metadataSpanAttrs(
				"inngest.ai", "step",
				`{"input_tokens":10,"output_tokens":5,"total_tokens":40,"request_model":"claude-3-5"}`,
			)},
			// Extended-trace entries can duplicate step-level reporting and must
			// not be counted.
			{DynamicSpanID: "md3", ParentSpanID: "step1", Name: meta.SpanNameMetadata, Attributes: metadataSpanAttrs(
				"inngest.ai", "extended_trace",
				`{"input_tokens":1000,"output_tokens":1000}`,
			)},
			// User-written run-scoped usage is additive.
			{DynamicSpanID: "md4", ParentSpanID: "root", Name: meta.SpanNameMetadata, Attributes: metadataSpanAttrs(
				"inngest.ai", "run",
				`{"input_tokens":1,"output_tokens":2}`,
			)},
			// A stored summary must be stripped and recomputed, never trusted.
			{DynamicSpanID: "md5", ParentSpanID: "root", Name: meta.SpanNameMetadata, Attributes: metadataSpanAttrs(
				"inngest.ai.summary", "run",
				`{"input_tokens":99999,"partial":true}`,
			)},
		}
		for _, s := range spans {
			s.RunID = runID
			insertTestSpan(t, cm, s)
		}

		root, err := cm.GetSpansByRunID(t.Context(), ulid.MustParse(runID))
		require.NoError(t, err)
		require.NotNil(t, root)

		summaries := findAISummary(t, root)
		require.Len(t, summaries, 1)
		require.Equal(t, enums.MetadataScopeRun, summaries[0].Scope)

		sum, err := extractors.AISummaryFromValues(summaries[0].Values)
		require.NoError(t, err)
		require.Equal(t, int64(111), sum.InputTokens)
		require.Equal(t, int64(27), sum.OutputTokens)
		require.Equal(t, int64(163), sum.TotalTokens)
		require.NotNil(t, sum.EstimatedCost)
		require.InDelta(t, 0.05, *sum.EstimatedCost, 1e-9)
		// md1 reported both models, so the response model wins; md2 reported
		// only a request model, so that is used as the fallback.
		require.Equal(t, []string{"claude-3-5", "gpt-4o-mini"}, sum.Models)
		require.False(t, sum.Partial)

		// The user's own run-scoped entry stays visible alongside the summary.
		userEntries := 0
		for _, md := range root.Metadata {
			if md.Kind == extractors.KindInngestAI {
				userEntries++
			}
		}
		require.Equal(t, 1, userEntries)
	})

	// Producers emit optional token counts as floats, which the step-level
	// AIMetadata struct types as *int64. Parsing must not be so strict that
	// such an entry's tokens are dropped from the summary.
	t.Run("counts entries whose optional fields are fractional", func(t *testing.T) {
		cm, cleanup := initCQRS(t)
		defer cleanup()

		runID := ulid.MustNew(ulid.Now(), rand.Reader).String()
		spans := []testSpanFields{
			{DynamicSpanID: "root", Name: meta.SpanNameRun, Attributes: runAttr},
			{DynamicSpanID: "step1", ParentSpanID: "root", Name: meta.SpanNameStep, Attributes: stepAttrs("a", 0)},
			{DynamicSpanID: "md1", ParentSpanID: "step1", Name: meta.SpanNameMetadata, Attributes: metadataSpanAttrs(
				"inngest.ai", "step",
				`{"input_tokens":37,"output_tokens":6,"total_tokens":43,"cache_read_tokens":7.5,"request_model":"gpt-5.4-mini","estimated_cost":0.000055}`,
			)},
		}
		for _, s := range spans {
			s.RunID = runID
			insertTestSpan(t, cm, s)
		}

		root, err := cm.GetSpansByRunID(t.Context(), ulid.MustParse(runID))
		require.NoError(t, err)

		summaries := findAISummary(t, root)
		require.Len(t, summaries, 1)
		sum, err := extractors.AISummaryFromValues(summaries[0].Values)
		require.NoError(t, err)
		require.Equal(t, int64(37), sum.InputTokens)
		require.Equal(t, int64(43), sum.TotalTokens)
		require.NotNil(t, sum.CacheReadTokens)
		require.Equal(t, int64(7), *sum.CacheReadTokens)
	})

	t.Run("sums granular tokens and providers, omitting unreported fields", func(t *testing.T) {
		cm, cleanup := initCQRS(t)
		defer cleanup()

		runID := ulid.MustNew(ulid.Now(), rand.Reader).String()
		spans := []testSpanFields{
			{DynamicSpanID: "root", Name: meta.SpanNameRun, Attributes: runAttr},
			{DynamicSpanID: "step1", ParentSpanID: "root", Name: meta.SpanNameStep, Attributes: stepAttrs("a", 0)},
			{DynamicSpanID: "step2", ParentSpanID: "root", Name: meta.SpanNameStep, Attributes: stepAttrs("b", 0)},
			{DynamicSpanID: "md1", ParentSpanID: "step1", Name: meta.SpanNameMetadata, Attributes: metadataSpanAttrs(
				"inngest.ai", "step",
				`{"input_tokens":10,"output_tokens":2,"cache_read_tokens":7,"reasoning_tokens":4,"provider":"openai"}`,
			)},
			// No cache_read_tokens and no cache_creation_tokens anywhere but md1's
			// read count, so creation stays absent while read is still summed.
			{DynamicSpanID: "md2", ParentSpanID: "step2", Name: meta.SpanNameMetadata, Attributes: metadataSpanAttrs(
				"inngest.ai", "step",
				`{"input_tokens":3,"output_tokens":1,"reasoning_tokens":6,"provider":"anthropic"}`,
			)},
		}
		for _, s := range spans {
			s.RunID = runID
			insertTestSpan(t, cm, s)
		}

		root, err := cm.GetSpansByRunID(t.Context(), ulid.MustParse(runID))
		require.NoError(t, err)

		summaries := findAISummary(t, root)
		require.Len(t, summaries, 1)
		sum, err := extractors.AISummaryFromValues(summaries[0].Values)
		require.NoError(t, err)

		require.NotNil(t, sum.CacheReadTokens)
		require.Equal(t, int64(7), *sum.CacheReadTokens)
		require.NotNil(t, sum.ReasoningTokens)
		require.Equal(t, int64(10), *sum.ReasoningTokens)
		require.Nil(t, sum.CacheCreationTokens)
		require.Equal(t, []string{"anthropic", "openai"}, sum.Providers)

		// Absent fields must not serialize as a misleading zero.
		require.NotContains(t, summaries[0].Values, "cache_creation_tokens")
	})

	t.Run("partial when the run invokes a child run", func(t *testing.T) {
		cm, cleanup := initCQRS(t)
		defer cleanup()

		runID := ulid.MustNew(ulid.Now(), rand.Reader).String()
		childRunID := ulid.MustNew(ulid.Now(), rand.Reader).String()
		invokeAttrs := fmt.Appendf(nil,
			`{"_inngest.step.id":"inv","_inngest.step.attempt":0,"_inngest.step.invoke.run.id":%q}`,
			childRunID,
		)

		spans := []testSpanFields{
			{DynamicSpanID: "root", Name: meta.SpanNameRun, Attributes: runAttr},
			{DynamicSpanID: "step1", ParentSpanID: "root", Name: meta.SpanNameStep, Attributes: invokeAttrs},
			{DynamicSpanID: "md1", ParentSpanID: "root", Name: meta.SpanNameMetadata, Attributes: metadataSpanAttrs(
				"inngest.ai", "run", `{"input_tokens":5,"output_tokens":5}`,
			)},
		}
		for _, s := range spans {
			s.RunID = runID
			insertTestSpan(t, cm, s)
		}

		root, err := cm.GetSpansByRunID(t.Context(), ulid.MustParse(runID))
		require.NoError(t, err)

		summaries := findAISummary(t, root)
		require.Len(t, summaries, 1)
		sum, err := extractors.AISummaryFromValues(summaries[0].Values)
		require.NoError(t, err)
		require.True(t, sum.Partial)
		require.Equal(t, int64(10), sum.TotalTokens)
	})

	// The OTLP endpoint can be the only carrier of gen_ai usage — emitted
	// outside any step, or from a source with no metadata processor. With no
	// step-scoped entry an extended-trace entry cannot be a second report of
	// an already-counted call, so it counts.
	t.Run("counts extended_trace when the run has no step-scoped AI", func(t *testing.T) {
		cm, cleanup := initCQRS(t)
		defer cleanup()

		runID := ulid.MustNew(ulid.Now(), rand.Reader).String()
		spans := []testSpanFields{
			{DynamicSpanID: "root", Name: meta.SpanNameRun, Attributes: runAttr},
			{DynamicSpanID: "step1", ParentSpanID: "root", Name: meta.SpanNameStep, Attributes: stepAttrs("a", 0)},
			{DynamicSpanID: "md1", ParentSpanID: "step1", Name: meta.SpanNameMetadata, Attributes: metadataSpanAttrs(
				"inngest.ai", "extended_trace",
				`{"input_tokens":250,"output_tokens":50,"request_model":"gpt-4o"}`,
			)},
		}
		for _, s := range spans {
			s.RunID = runID
			insertTestSpan(t, cm, s)
		}

		root, err := cm.GetSpansByRunID(t.Context(), ulid.MustParse(runID))
		require.NoError(t, err)

		summaries := findAISummary(t, root)
		require.Len(t, summaries, 1)
		sum, err := extractors.AISummaryFromValues(summaries[0].Values)
		require.NoError(t, err)
		require.Equal(t, int64(250), sum.InputTokens)
		require.Equal(t, int64(300), sum.TotalTokens)
		require.Equal(t, []string{"gpt-4o"}, sum.Models)
	})

	// A zero-valued partial summary would put an AI card on every
	// invoke-bearing run, so invoke steps alone attach nothing.
	t.Run("no summary without AI usage, even with invokes", func(t *testing.T) {
		cm, cleanup := initCQRS(t)
		defer cleanup()

		runID := ulid.MustNew(ulid.Now(), rand.Reader).String()
		childRunID := ulid.MustNew(ulid.Now(), rand.Reader).String()
		invokeAttrs := fmt.Appendf(nil,
			`{"_inngest.step.id":"inv","_inngest.step.attempt":0,"_inngest.step.invoke.run.id":%q}`,
			childRunID,
		)
		spans := []testSpanFields{
			{DynamicSpanID: "root", Name: meta.SpanNameRun, Attributes: runAttr},
			{DynamicSpanID: "step1", ParentSpanID: "root", Name: meta.SpanNameStep, Attributes: stepAttrs("a", 0)},
			{DynamicSpanID: "step2", ParentSpanID: "root", Name: meta.SpanNameStep, Attributes: invokeAttrs},
		}
		for _, s := range spans {
			s.RunID = runID
			insertTestSpan(t, cm, s)
		}

		root, err := cm.GetSpansByRunID(t.Context(), ulid.MustParse(runID))
		require.NoError(t, err)
		require.Empty(t, findAISummary(t, root))
	})
}

// Every inngest.ai emission under a given parent shares one dynamic_span_id,
// and the kind's merge opcode is last-write-wins — so rolling the group up
// into a single entry would discard all but the final emission's usage.
func TestCQRSAISummaryCountsEveryEmissionUnderOneParent(t *testing.T) {
	cm, cleanup := initCQRS(t)
	defer cleanup()

	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	traceID := ulid.MustNew(ulid.Now(), rand.Reader).String()
	aiAttrs := func(in, out int) []byte {
		return metadataSpanAttrs("inngest.ai", "run", fmt.Sprintf(
			`{"input_tokens":%d,"output_tokens":%d}`, in, out,
		))
	}

	spans := []testSpanFields{
		{DynamicSpanID: "root", Name: meta.SpanNameRun, Attributes: []byte(`{"_inngest.dynamic.status":"Completed"}`)},
		// Two separate emissions of the same kind under the same parent, as
		// CreateMetadataSpanFromValues produces them.
		{DynamicSpanID: "md-ai", ParentSpanID: "root", Name: meta.SpanNameMetadata, Attributes: aiAttrs(100, 10), StartTime: time.Now()},
		{DynamicSpanID: "md-ai", ParentSpanID: "root", Name: meta.SpanNameMetadata, Attributes: aiAttrs(200, 20), StartTime: time.Now().Add(time.Millisecond)},
	}
	for _, s := range spans {
		s.RunID = runID.String()
		s.TraceID = traceID
		insertTestSpan(t, cm, s)
	}

	root, err := cm.GetSpansByRunID(t.Context(), runID)
	require.NoError(t, err)

	summaries := findAISummary(t, root)
	require.Len(t, summaries, 1)
	tree, err := extractors.AISummaryFromValues(summaries[0].Values)
	require.NoError(t, err)
	require.Equal(t, int64(300), tree.InputTokens, "the earlier emission must not be overwritten by the later one")
	require.Equal(t, int64(30), tree.OutputTokens)
	require.Equal(t, int64(330), tree.TotalTokens)
}

// Metadata spans reference their parent by ID with no existence check, so a
// dangling reference must still count toward the run's usage rather than
// silently vanishing from the tree-assembled summary.
func TestCQRSAISummaryCountsMetadataWithMissingParentSpan(t *testing.T) {
	cm, cleanup := initCQRS(t)
	defer cleanup()

	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	spans := []testSpanFields{
		{DynamicSpanID: "root", Name: meta.SpanNameRun, Attributes: []byte(`{"_inngest.dynamic.status":"Completed"}`)},
		{DynamicSpanID: "step1", ParentSpanID: "root", Name: meta.SpanNameStep, Attributes: []byte(`{"_inngest.step.id":"a","_inngest.step.attempt":0}`)},
		{DynamicSpanID: "md1", ParentSpanID: "step1", Name: meta.SpanNameMetadata, Attributes: metadataSpanAttrs(
			"inngest.ai", "step", `{"input_tokens":10,"output_tokens":5}`,
		)},
		// Parented on a step span that was never written.
		{DynamicSpanID: "md2", ParentSpanID: "does-not-exist", Name: meta.SpanNameMetadata, Attributes: metadataSpanAttrs(
			"inngest.ai", "step", `{"input_tokens":100,"output_tokens":50}`,
		)},
	}
	for _, s := range spans {
		s.RunID = runID.String()
		insertTestSpan(t, cm, s)
	}

	root, err := cm.GetSpansByRunID(t.Context(), runID)
	require.NoError(t, err)

	summaries := findAISummary(t, root)
	require.Len(t, summaries, 1)
	tree, err := extractors.AISummaryFromValues(summaries[0].Values)
	require.NoError(t, err)
	require.Equal(t, int64(110), tree.InputTokens)
	require.Equal(t, int64(55), tree.OutputTokens)
}

// findSpanByID returns the span with the given dynamic span ID from an
// assembled tree.
func findSpanByID(t *testing.T, span *cqrs.OtelSpan, dynamicSpanID string) *cqrs.OtelSpan {
	t.Helper()
	if span.SpanID == dynamicSpanID {
		return span
	}
	for _, child := range span.Children {
		if found := findSpanByID(t, child, dynamicSpanID); found != nil {
			return found
		}
	}
	return nil
}

func findAIEntries(span *cqrs.OtelSpan) []*cqrs.SpanMetadata {
	var out []*cqrs.SpanMetadata
	for _, md := range span.Metadata {
		if md.Kind == extractors.KindInngestAI {
			out = append(out, md)
		}
	}
	return out
}

// Every inngest.ai emission under one parent shares a dynamic_span_id, so the
// group's own end time is a single value common to all of them. Stamping each
// entry with that value would erase the order of the calls a step made, so
// each entry carries its own fragment's start time instead. Cloud's ClickHouse
// rollup stamps the same value; the two must agree.
func TestCQRSAIEntriesCarryPerEmissionTimestamps(t *testing.T) {
	cm, cleanup := initCQRS(t)
	defer cleanup()

	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	traceID := ulid.MustNew(ulid.Now(), rand.Reader).String()
	first := time.Now().UTC().Truncate(time.Millisecond)
	second := first.Add(50 * time.Millisecond)

	aiAttrs := func(in int) []byte {
		return metadataSpanAttrs("inngest.ai", "step", fmt.Sprintf(`{"input_tokens":%d,"output_tokens":1}`, in))
	}

	spans := []testSpanFields{
		{DynamicSpanID: "root", Name: meta.SpanNameRun, Attributes: []byte(`{"_inngest.dynamic.status":"Completed"}`), StartTime: first},
		{DynamicSpanID: "step1", ParentSpanID: "root", Name: meta.SpanNameStep, Attributes: []byte(`{"_inngest.step.id":"a","_inngest.step.attempt":0}`), StartTime: first},
		{DynamicSpanID: "md-ai", ParentSpanID: "step1", Name: meta.SpanNameMetadata, Attributes: aiAttrs(100), StartTime: first},
		{DynamicSpanID: "md-ai", ParentSpanID: "step1", Name: meta.SpanNameMetadata, Attributes: aiAttrs(200), StartTime: second},
	}
	for _, s := range spans {
		s.RunID = runID.String()
		s.TraceID = traceID
		insertTestSpan(t, cm, s)
	}

	root, err := cm.GetSpansByRunID(t.Context(), runID)
	require.NoError(t, err)

	step := findSpanByID(t, root, "step1")
	require.NotNil(t, step)

	entries := findAIEntries(step)
	require.Len(t, entries, 2, "each emission must survive as its own entry")

	// Entries stay in emission order, each stamped with its own start time
	// rather than the group's end time (start + 100ms, per insertTestSpan).
	require.WithinDuration(t, first, entries[0].UpdatedAt, time.Millisecond)
	require.WithinDuration(t, second, entries[1].UpdatedAt, time.Millisecond)
	require.True(t, entries[0].UpdatedAt.Before(entries[1].UpdatedAt), "per-emission timestamps must be distinct")
	assert.JSONEq(t, "100", string(entries[0].Values["input_tokens"]))
	assert.JSONEq(t, "200", string(entries[1].Values["input_tokens"]))
}

// Float addition is not associative and entries are folded in map order, so
// the raw cost sum can differ by ULPs between reads. RoundCost at output must
// absorb that: the same run reports the same cost on every read regardless of
// fold order.
func TestCQRSAISummaryCostSumIsOrderDeterministic(t *testing.T) {
	cm, cleanup := initCQRS(t)
	defer cleanup()

	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	traceID := ulid.MustNew(ulid.Now(), rand.Reader).String()
	// Same millisecond for every emission, so nothing distinguishes fold
	// orders.
	emittedAt := time.Now().UTC().Truncate(time.Millisecond)

	// Costs whose raw sum differs by an ULP depending on the order applied:
	// (0.1+0.2)+0.3 != 0.1+(0.2+0.3). Rounded, both orders are exactly 0.6.
	costs := []float64{0.1, 0.2, 0.3}
	aiAttrs := func(cost float64) []byte {
		return metadataSpanAttrs("inngest.ai", "step", fmt.Sprintf(
			`{"input_tokens":1,"output_tokens":1,"estimated_cost":%v}`, cost,
		))
	}

	spans := []testSpanFields{
		{DynamicSpanID: "root", Name: meta.SpanNameRun, Attributes: []byte(`{"_inngest.dynamic.status":"Completed"}`), StartTime: emittedAt},
	}
	// Insert in an order unrelated to the span IDs the sum must follow.
	for _, i := range []int{2, 0, 1} {
		stepID := fmt.Sprintf("step%d", i)
		spans = append(spans,
			testSpanFields{DynamicSpanID: stepID, ParentSpanID: "root", Name: meta.SpanNameStep, StartTime: emittedAt, Attributes: fmt.Appendf(nil,
				`{"_inngest.step.id":%q,"_inngest.step.attempt":0}`, stepID,
			)},
			testSpanFields{DynamicSpanID: "md-" + stepID, ParentSpanID: stepID, Name: meta.SpanNameMetadata, StartTime: emittedAt, Attributes: aiAttrs(costs[i])},
		)
	}
	for _, s := range spans {
		s.RunID = runID.String()
		s.TraceID = traceID
		insertTestSpan(t, cm, s)
	}

	root, err := cm.GetSpansByRunID(t.Context(), runID)
	require.NoError(t, err)

	summaries := findAISummary(t, root)
	require.Len(t, summaries, 1)
	sum, err := extractors.AISummaryFromValues(summaries[0].Values)
	require.NoError(t, err)
	require.NotNil(t, sum.EstimatedCost)
	require.Equal(t, 0.6, *sum.EstimatedCost, "rounded cost must not depend on fold order")
}

// A step's per-emission entries are unbounded in count on the extended trace
// ingest path (no run state, so consts.MaxRunMetadataSize never applies), so
// what a span carries is capped. The run's summary is summed before the cap is
// applied and must stay exact past it.
func TestCQRSAIEntriesAreCappedPerSpanWithExactSummary(t *testing.T) {
	cm, cleanup := initCQRS(t)
	defer cleanup()

	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	traceID := ulid.MustNew(ulid.Now(), rand.Reader).String()
	emissions := consts.MaxSpanMetadataEntries + 10
	start := time.Now().UTC().Truncate(time.Millisecond)

	spans := []testSpanFields{
		{DynamicSpanID: "root", Name: meta.SpanNameRun, Attributes: []byte(`{"_inngest.dynamic.status":"Completed"}`), StartTime: start},
		{DynamicSpanID: "step1", ParentSpanID: "root", Name: meta.SpanNameStep, StartTime: start, Attributes: []byte(`{"_inngest.step.id":"a","_inngest.step.attempt":0}`)},
	}
	for i := range emissions {
		spans = append(spans, testSpanFields{
			DynamicSpanID: "md-ai",
			ParentSpanID:  "step1",
			Name:          meta.SpanNameMetadata,
			StartTime:     start.Add(time.Duration(i) * time.Millisecond),
			Attributes:    metadataSpanAttrs("inngest.ai", "step", `{"input_tokens":1,"output_tokens":1}`),
		})
	}
	for _, s := range spans {
		s.RunID = runID.String()
		s.TraceID = traceID
		insertTestSpan(t, cm, s)
	}

	root, err := cm.GetSpansByRunID(t.Context(), runID)
	require.NoError(t, err)

	step := findSpanByID(t, root, "step1")
	require.NotNil(t, step)
	require.Len(t, step.Metadata, consts.MaxSpanMetadataEntries, "attached entries are capped")

	summaries := findAISummary(t, root)
	require.Len(t, summaries, 1)
	sum, err := extractors.AISummaryFromValues(summaries[0].Values)
	require.NoError(t, err)
	require.Equal(t, int64(emissions), sum.InputTokens, "the summary sums every emission, not just the attached ones")
}
