package executor

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/exechttp"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	"github.com/stretchr/testify/require"
)

// mockRunContext is a minimal RunContext implementation for testing createMetadataSpan.
type mockRunContext struct {
	md sv2.Metadata
}

func (m *mockRunContext) Metadata() *sv2.Metadata          { return &m.md }
func (m *mockRunContext) DriverResponse() *state.DriverResponse { return nil }
func (m *mockRunContext) Events() []json.RawMessage         { return nil }
func (m *mockRunContext) HTTPClient() exechttp.RequestExecutor { return nil }
func (m *mockRunContext) ExecutionSpan() *meta.SpanReference { return &meta.SpanReference{} }
func (m *mockRunContext) ParentSpan() *meta.SpanReference   { return &meta.SpanReference{} }
func (m *mockRunContext) GroupID() string                    { return "" }
func (m *mockRunContext) AttemptCount() int                  { return 0 }
func (m *mockRunContext) MaxAttempts() *int                  { return nil }
func (m *mockRunContext) ShouldRetry() bool                 { return false }
func (m *mockRunContext) IncrementAttempt()                  {}
func (m *mockRunContext) PriorityFactor() *int64             { return nil }
func (m *mockRunContext) ConcurrencyKeys() []state.CustomConcurrency { return nil }
func (m *mockRunContext) ParallelMode() enums.ParallelMode  { return 0 }
func (m *mockRunContext) LifecycleItem() queue.Item         { return queue.Item{} }
func (m *mockRunContext) SetStatusCode(code int)             {}
func (m *mockRunContext) UpdateOpcodeError(op *state.GeneratorOpcode, err state.UserError) {}
func (m *mockRunContext) UpdateOpcodeOutput(op *state.GeneratorOpcode, output json.RawMessage) {}
func (m *mockRunContext) SetError(err error)                 {}

// Compile-time check that mockRunContext implements RunContext.
var _ execution.RunContext = (*mockRunContext)(nil)

// mockStructured implements metadata.Structured with configurable behavior.
type mockStructured struct {
	kind      metadata.Kind
	values    metadata.Values
	serializeErr error
}

func (m *mockStructured) Kind() metadata.Kind             { return m.kind }
func (m *mockStructured) Op() enums.MetadataOpcode        { return enums.MetadataOpcodeMerge }
func (m *mockStructured) Serialize() (metadata.Values, error) {
	if m.serializeErr != nil {
		return nil, m.serializeErr
	}
	return m.values, nil
}

// makeValues creates a Values map with a single key whose total size (key + value) equals targetSize.
func makeValues(targetSize int) metadata.Values {
	if targetSize <= 0 {
		return metadata.Values{}
	}
	key := "k"
	valSize := targetSize - len(key)
	if valSize < 0 {
		// If target is smaller than key, just use the key alone
		return metadata.Values{key[:targetSize]: json.RawMessage{}}
	}
	return metadata.Values{
		key: json.RawMessage(strings.Repeat("x", valSize)),
	}
}

func newTestExecutor() *executor {
	return &executor{
		log:            logger.From(context.Background()),
		tracerProvider: tracing.NewNoopTracerProvider(),
	}
}

func newTestRunContext() *mockRunContext {
	return &mockRunContext{
		md: sv2.Metadata{},
	}
}

func TestCreateMetadataSpan_SpanExactlyAtLimit(t *testing.T) {
	e := newTestExecutor()
	rc := newTestRunContext()

	md := &mockStructured{
		kind:   "test.kind",
		values: makeValues(consts.MaxMetadataSpanSize), // exactly 64 KB
	}

	ref, err := e.createMetadataSpan(context.Background(), rc, "test.location", md, enums.MetadataScopeStep)
	require.NoError(t, err)
	require.NotNil(t, ref)
}

func TestCreateMetadataSpan_SpanOverLimit(t *testing.T) {
	e := newTestExecutor()
	rc := newTestRunContext()

	md := &mockStructured{
		kind:   "test.kind",
		values: makeValues(consts.MaxMetadataSpanSize + 1), // 64 KB + 1
	}

	ref, err := e.createMetadataSpan(context.Background(), rc, "test.location", md, enums.MetadataScopeStep)
	require.ErrorIs(t, err, metadata.ErrMetadataSpanTooLarge)
	require.Nil(t, ref)
}

func TestCreateMetadataSpan_CumulativeWithinLimit(t *testing.T) {
	e := newTestExecutor()
	rc := newTestRunContext()

	md := &mockStructured{
		kind:   "test.kind",
		values: makeValues(1000),
	}

	ref, err := e.createMetadataSpan(context.Background(), rc, "test.location", md, enums.MetadataScopeStep)
	require.NoError(t, err)
	require.NotNil(t, ref)
}

func TestCreateMetadataSpan_CumulativeOverLimit(t *testing.T) {
	e := newTestExecutor()
	rc := newTestRunContext()
	// Set current size just below limit
	rc.md.Metrics.MetadataSize = consts.MaxRunMetadataSize - 100

	md := &mockStructured{
		kind:   "test.kind",
		values: makeValues(101), // would push over 1 MB
	}

	ref, err := e.createMetadataSpan(context.Background(), rc, "test.location", md, enums.MetadataScopeStep)
	require.ErrorIs(t, err, metadata.ErrRunMetadataSizeExceeded)
	require.Nil(t, ref)
}

func TestCreateMetadataSpan_CumulativeExactlyAtLimit(t *testing.T) {
	e := newTestExecutor()
	rc := newTestRunContext()
	// Set current size so that adding the span reaches exactly the limit (should be accepted)
	rc.md.Metrics.MetadataSize = consts.MaxRunMetadataSize - 500

	md := &mockStructured{
		kind:   "test.kind",
		values: makeValues(500), // currentSize + spanSize == MaxRunMetadataSize
	}

	ref, err := e.createMetadataSpan(context.Background(), rc, "test.location", md, enums.MetadataScopeStep)
	require.NoError(t, err)
	require.NotNil(t, ref)
}

func TestCreateMetadataSpan_CumulativeAtMaxRejectsNonZero(t *testing.T) {
	e := newTestExecutor()
	rc := newTestRunContext()
	// MetadataSize is exactly at the max
	rc.md.Metrics.MetadataSize = consts.MaxRunMetadataSize

	md := &mockStructured{
		kind:   "test.kind",
		values: makeValues(1), // any non-zero span should be rejected
	}

	ref, err := e.createMetadataSpan(context.Background(), rc, "test.location", md, enums.MetadataScopeStep)
	require.ErrorIs(t, err, metadata.ErrRunMetadataSizeExceeded)
	require.Nil(t, ref)
}

func TestCreateMetadataSpan_SequentialAccumulation(t *testing.T) {
	e := newTestExecutor()
	rc := newTestRunContext()

	spanSize := 100_000 // 100 KB per span, under 64 KB per-span limit? No, 100KB > 64KB.
	// Use a span size under the per-span limit (64 KB)
	spanSize = 50_000 // 50 KB per span

	// We can fit floor(1MB / 50KB) = 20 spans, and the 21st should be rejected
	md := &mockStructured{
		kind:   "test.kind",
		values: makeValues(spanSize),
	}

	expectedFits := consts.MaxRunMetadataSize / spanSize // 1048576 / 50000 = 20

	for i := 0; i < expectedFits; i++ {
		ref, err := e.createMetadataSpan(context.Background(), rc, "test.location", md, enums.MetadataScopeStep)
		require.NoError(t, err, "span %d should succeed", i)
		require.NotNil(t, ref)
	}

	// Verify cumulative size accumulated correctly
	require.Equal(t, expectedFits*spanSize, rc.md.Metrics.MetadataSize)

	// The next span should exceed the cumulative limit
	ref, err := e.createMetadataSpan(context.Background(), rc, "test.location", md, enums.MetadataScopeStep)
	require.ErrorIs(t, err, metadata.ErrRunMetadataSizeExceeded)
	require.Nil(t, ref)

	// MetadataSize should not have changed after rejection
	require.Equal(t, expectedFits*spanSize, rc.md.Metrics.MetadataSize)
}

func TestCreateMetadataSpan_SerializationError(t *testing.T) {
	e := newTestExecutor()
	rc := newTestRunContext()

	md := &mockStructured{
		kind:         "test.kind",
		serializeErr: errors.New("marshal failed"),
	}

	ref, err := e.createMetadataSpan(context.Background(), rc, "test.location", md, enums.MetadataScopeStep)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to serialize metadata")
	require.Nil(t, ref)
}

func TestCreateMetadataSpan_EmptyValues(t *testing.T) {
	e := newTestExecutor()
	rc := newTestRunContext()

	md := &mockStructured{
		kind:   "test.kind",
		values: metadata.Values{}, // empty, size = 0
	}

	ref, err := e.createMetadataSpan(context.Background(), rc, "test.location", md, enums.MetadataScopeStep)
	require.NoError(t, err)
	require.NotNil(t, ref)
	// MetadataSize should still be 0 since span size was 0
	require.Equal(t, 0, rc.md.Metrics.MetadataSize)
}
