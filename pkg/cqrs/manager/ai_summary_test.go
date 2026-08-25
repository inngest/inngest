package manager

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	"github.com/inngest/inngest/pkg/tracing/metadata/extractors"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

const (
	testTraceID = "trace-ai-summary"
	testRunID   = "01HQ4T3Z8B0000000000000000"
)

var testStart = time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC)

// attrEncoding selects how a fragment's attributes field is delivered by the
// backend: sqlite's json_object embeds the JSON column as a quoted string,
// while postgres' json_build_object embeds jsonb as a nested object.
type attrEncoding string

const (
	encodingSQLite   attrEncoding = "sqlite"
	encodingPostgres attrEncoding = "postgres"
)

func eachEncoding(t *testing.T, fn func(t *testing.T, enc attrEncoding)) {
	t.Helper()
	for _, enc := range []attrEncoding{encodingSQLite, encodingPostgres} {
		t.Run(string(enc), func(t *testing.T) { fn(t, enc) })
	}
}

func encodeFragmentAttributes(t *testing.T, enc attrEncoding, rows []testSpanRow) {
	t.Helper()
	if enc != encodingSQLite {
		return
	}
	for _, row := range rows {
		for _, fragment := range row.fragments {
			attrs, ok := fragment["attributes"].(map[string]any)
			if !ok {
				continue
			}
			raw, err := json.Marshal(attrs)
			require.NoError(t, err)
			fragment["attributes"] = string(raw)
		}
	}
}

type testSpanRow struct {
	dynamicSpanID string
	parentSpanID  string
	fragments     []map[string]any
}

func (r testSpanRow) GetTraceID() string { return testTraceID }
func (r testSpanRow) GetRunID() string   { return testRunID }
func (r testSpanRow) GetDynamicSpanID() sql.NullString {
	return sql.NullString{String: r.dynamicSpanID, Valid: r.dynamicSpanID != ""}
}
func (r testSpanRow) GetParentSpanID() sql.NullString {
	return sql.NullString{String: r.parentSpanID, Valid: r.parentSpanID != ""}
}
func (r testSpanRow) GetStartTime() interface{} { return testStart }
func (r testSpanRow) GetEndTime() interface{}   { return testStart.Add(time.Second) }
func (r testSpanRow) GetSpanFragments() any {
	raw, err := json.Marshal(r.fragments)
	if err != nil {
		panic(err)
	}
	return json.RawMessage(raw)
}

func runRow() testSpanRow {
	return testSpanRow{
		dynamicSpanID: "root",
		fragments: []map[string]any{{
			"span_id": "otel-root",
			"name":    meta.SpanNameRun,
		}},
	}
}

func stepRow(id string, attrs map[string]any) testSpanRow {
	fragment := map[string]any{
		"span_id": "otel-" + id,
		"name":    meta.SpanNameStep,
	}
	if attrs != nil {
		fragment["attributes"] = attrs
	}
	return testSpanRow{
		dynamicSpanID: id,
		parentSpanID:  "root",
		fragments:     []map[string]any{fragment},
	}
}

// aiMetadataFragment mirrors the fragment shape a metadata span row carries in
// the DB: the four _inngest.metadata.* attribute keys written by
// tracing.RawMetadataAttrs, with values as a JSON-encoded string.
func aiMetadataFragment(t *testing.T, scope metadata.Scope, values map[string]any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(values)
	require.NoError(t, err)
	scopeText, err := scope.MarshalText()
	require.NoError(t, err)
	opText, err := enums.MetadataOpcodeMerge.MarshalText()
	require.NoError(t, err)
	return map[string]any{
		"name": meta.SpanNameMetadata,
		"attributes": map[string]any{
			"_inngest.metadata.kind":   extractors.KindInngestAI.String(),
			"_inngest.metadata.scope":  string(scopeText),
			"_inngest.metadata.op":     string(opText),
			"_inngest.metadata.values": string(raw),
		},
	}
}

func aiMetadataRow(id, parent string, fragments ...map[string]any) testSpanRow {
	return testSpanRow{
		dynamicSpanID: id,
		parentSpanID:  parent,
		fragments:     fragments,
	}
}

var (
	stepCallA = map[string]any{
		"input_tokens": 37, "output_tokens": 10, "total_tokens": 47,
		"estimated_cost": 0.000073, "latency_ms": 2246.002685546875,
		"provider": "openai", "request_model": "gpt-5.4-mini",
		"response_model": "gpt-5.4-mini-2026-03-17",
	}
	stepCallB = map[string]any{
		"input_tokens": 30, "output_tokens": 19, "total_tokens": 49,
		"estimated_cost": 0.000108, "latency_ms": 1737.404052734375,
		"provider": "openai", "request_model": "gpt-5.4-mini",
		"response_model": "gpt-5.4-mini-2026-03-17",
	}
	inferCall = map[string]any{
		"input_tokens": 42, "output_tokens": 17, "total_tokens": 59,
		"estimated_cost": 0.000275, "latency_ms": 661,
		"provider": "openai-chat", "request_model": "gpt-4o",
		"response_model": "gpt-4o-2024-08-06",
	}
	runCall = map[string]any{
		"input_tokens": 11, "output_tokens": 3,
		"estimated_cost": 0.000012,
		"provider":       "anthropic", "request_model": "claude-opus-5",
	}
)

func buildRoot(t *testing.T, enc attrEncoding, rows []testSpanRow) *cqrs.OtelSpan {
	t.Helper()
	encodeFragmentAttributes(t, enc, rows)
	root, err := mapRootSpansFromRows(context.Background(), rows)
	require.NoError(t, err)
	return root
}

func findAISummaries(root *cqrs.OtelSpan) []*cqrs.SpanMetadata {
	var found []*cqrs.SpanMetadata
	for _, md := range root.Metadata {
		if md.Kind == extractors.KindInngestAISummary {
			found = append(found, md)
		}
	}
	return found
}

func requireSummary(t *testing.T, root *cqrs.OtelSpan) extractors.AISummaryMetadata {
	t.Helper()
	found := findAISummaries(root)
	require.Len(t, found, 1)
	require.Equal(t, enums.MetadataScopeRun, found[0].Scope)
	summary, err := extractors.AISummaryFromValues(found[0].Values)
	require.NoError(t, err)
	return summary
}

func TestAISummarySumsAcrossScopes(t *testing.T) {
	eachEncoding(t, func(t *testing.T, enc attrEncoding) {
		rows := []testSpanRow{
			runRow(),
			stepRow("step-a", nil),
			stepRow("step-b", nil),
			aiMetadataRow("md-step-a", "step-a", aiMetadataFragment(t, enums.MetadataScopeStep, stepCallA)),
			aiMetadataRow("md-step-b", "step-b", aiMetadataFragment(t, enums.MetadataScopeStep, stepCallB)),
			// The extended_trace entries are the same calls re-reported, and must
			// not be counted twice.
			aiMetadataRow("md-ext-a", "step-a", aiMetadataFragment(t, enums.MetadataScopeExtendedTrace, stepCallA)),
			aiMetadataRow("md-ext-b", "step-b", aiMetadataFragment(t, enums.MetadataScopeExtendedTrace, stepCallB)),
			aiMetadataRow("md-attempt", "step-b", aiMetadataFragment(t, enums.MetadataScopeStepAttempt, inferCall)),
			// Run-scoped usage on a metadata span with no parent still counts.
			aiMetadataRow("md-run", "", aiMetadataFragment(t, enums.MetadataScopeRun, runCall)),
		}

		summary := requireSummary(t, buildRoot(t, enc, rows))

		require.Equal(t, int64(37+30+42+11), summary.InputTokens)
		require.Equal(t, int64(10+19+17+3), summary.OutputTokens)
		require.Equal(t, int64(47+49+59+14), summary.TotalTokens)
		require.NotNil(t, summary.EstimatedCost)
		require.InDelta(t, 0.000073+0.000108+0.000275+0.000012, *summary.EstimatedCost, 1e-12)
		require.Equal(t, []string{"claude-opus-5", "gpt-4o-2024-08-06", "gpt-5.4-mini-2026-03-17"}, summary.Models)
		require.Equal(t, []string{"anthropic", "openai", "openai-chat"}, summary.Providers)
	})
}

func TestAISummaryExtendedTraceOnlyCounts(t *testing.T) {
	eachEncoding(t, func(t *testing.T, enc attrEncoding) {
		rows := []testSpanRow{
			runRow(),
			stepRow("step-a", nil),
			aiMetadataRow("md-ext", "step-a", aiMetadataFragment(t, enums.MetadataScopeExtendedTrace, inferCall)),
		}

		summary := requireSummary(t, buildRoot(t, enc, rows))

		require.Equal(t, int64(42), summary.InputTokens)
		require.Equal(t, int64(17), summary.OutputTokens)
		require.Equal(t, int64(59), summary.TotalTokens)
		require.Equal(t, []string{"gpt-4o-2024-08-06"}, summary.Models)
		require.Equal(t, []string{"openai-chat"}, summary.Providers)
	})
}

func TestAISummaryMultipleEmissionsOneDynamicSpanMerge(t *testing.T) {
	// Two emissions share one dynamic span ID. The merge opcode makes the
	// later one a per-key update of the same entry — not a new call — so the
	// summary counts the merged entry once, matching the state the span
	// itself carries.
	eachEncoding(t, func(t *testing.T, enc attrEncoding) {
		rows := []testSpanRow{
			runRow(),
			stepRow("step-a", nil),
			aiMetadataRow("md-multi", "step-a",
				aiMetadataFragment(t, enums.MetadataScopeStep, stepCallA),
				aiMetadataFragment(t, enums.MetadataScopeStep, stepCallB),
			),
		}

		summary := requireSummary(t, buildRoot(t, enc, rows))

		require.Equal(t, int64(30), summary.InputTokens)
		require.Equal(t, int64(19), summary.OutputTokens)
		require.Equal(t, int64(49), summary.TotalTokens)
		require.NotNil(t, summary.EstimatedCost)
		require.InDelta(t, 0.000108, *summary.EstimatedCost, 1e-12)
	})
}

func TestAISummaryPartialUpdateNotDoubleCounted(t *testing.T) {
	// A later emission carrying only a corrected cost updates the entry's
	// cost in place; its tokens and original cost must not be summed again.
	eachEncoding(t, func(t *testing.T, enc attrEncoding) {
		rows := []testSpanRow{
			runRow(),
			stepRow("step-a", nil),
			aiMetadataRow("md-update", "step-a",
				aiMetadataFragment(t, enums.MetadataScopeStep, stepCallA),
				aiMetadataFragment(t, enums.MetadataScopeStep, map[string]any{"estimated_cost": 0.0002}),
			),
		}

		summary := requireSummary(t, buildRoot(t, enc, rows))

		require.Equal(t, int64(37), summary.InputTokens)
		require.Equal(t, int64(10), summary.OutputTokens)
		require.Equal(t, int64(47), summary.TotalTokens)
		require.NotNil(t, summary.EstimatedCost)
		require.InDelta(t, 0.0002, *summary.EstimatedCost, 1e-12)
	})
}

func TestAISummaryAbsentWithoutAI(t *testing.T) {
	eachEncoding(t, func(t *testing.T, enc attrEncoding) {
		rows := []testSpanRow{
			runRow(),
			stepRow("step-a", nil),
		}

		root := buildRoot(t, enc, rows)
		require.Empty(t, findAISummaries(root))
	})
}

// aiMetadataAttributes builds the stored attributes blob for an inngest.ai
// metadata span the way tracer_sqlc.go does, so a rename of any
// _inngest.metadata.* key breaks these tests rather than silently passing.
func aiMetadataAttributes(t *testing.T, scope metadata.Scope, values map[string]any) []byte {
	t.Helper()

	var v metadata.Values
	require.NoError(t, v.FromStruct(values))

	attrs := tracing.RawMetadataAttrs(extractors.KindInngestAI, v, enums.MetadataOpcodeMerge)
	meta.AddAttr(attrs, meta.Attrs.MetadataScope, &scope)

	out := map[string]any{}
	for _, kv := range attrs.Serialize() {
		out[string(kv.Key)] = kv.Value.AsInterface()
	}

	raw, err := json.Marshal(out)
	require.NoError(t, err)
	return raw
}

// insertRunScopedAI writes a root span plus one metadata span per call, all
// sharing a single dynamic span ID. Two run-scoped AddRunMetadata POSTs collide
// this way: the parent comes from the run and the dynamic span ID from
// (parent, kind), so distinct calls differ only in span_id and merge
// last-write-wins into one entry.
func insertRunScopedAI(t *testing.T, cm cqrs.Manager, calls ...map[string]any) *cqrs.OtelSpan {
	t.Helper()

	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	traceID := ulid.MustNew(ulid.Now(), rand.Reader).String()

	insertTestSpan(t, cm, testSpanFields{
		RunID:         runID.String(),
		TraceID:       traceID,
		DynamicSpanID: "dyn-root",
		Name:          meta.SpanNameRun,
	})

	for _, call := range calls {
		insertTestSpan(t, cm, testSpanFields{
			RunID:         runID.String(),
			TraceID:       traceID,
			DynamicSpanID: "dyn-ai",
			ParentSpanID:  "dyn-root",
			Name:          meta.SpanNameMetadata,
			Attributes:    aiMetadataAttributes(t, enums.MetadataScopeRun, call),
		})
	}

	root, err := cm.GetSpansByRunID(t.Context(), runID)
	require.NoError(t, err)
	require.NotNil(t, root)
	return root
}

func TestAISummaryRunScopedCallsMergeLastWriteWins(t *testing.T) {
	cm, cleanup := initCQRS(t)
	defer cleanup()

	root := insertRunScopedAI(t, cm, stepCallA, stepCallB)
	summary := requireSummary(t, root)

	// The summary must agree with the merged entry the span itself carries.
	// Which call wins is left to fragment order, which the sqlite query does
	// not pin.
	var stored []*cqrs.SpanMetadata
	for _, md := range root.Metadata {
		if md.Kind == extractors.KindInngestAI {
			stored = append(stored, md)
		}
	}
	require.Len(t, stored, 1)
	var storedInput int64
	require.NoError(t, json.Unmarshal(stored[0].Values["input_tokens"], &storedInput))
	require.Contains(t, []int64{37, 30}, storedInput)
	require.Equal(t, storedInput, summary.InputTokens)
}
