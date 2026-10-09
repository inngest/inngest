package tracing

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// recordingMetadataListener implements execution.SyncLifecycleListener,
// recording every OnMetadataEntry call for assertions.
type recordingMetadataListener struct {
	execution.NoopSyncLifecycleListener
	entries []execution.MetadataEntry
}

func (r *recordingMetadataListener) OnMetadataEntry(ctx context.Context, entry execution.MetadataEntry) {
	r.entries = append(r.entries, entry)
}

// mockStructured implements metadata.Structured with configurable behavior.
type mockStructured struct {
	kind         metadata.Kind
	values       metadata.Values
	serializeErr error
}

func (m *mockStructured) Kind() metadata.Kind      { return m.kind }
func (m *mockStructured) Op() enums.MetadataOpcode { return enums.MetadataOpcodeMerge }
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
		return metadata.Values{key[:targetSize]: json.RawMessage{}}
	}
	return metadata.Values{
		key: json.RawMessage(strings.Repeat("x", valSize)),
	}
}

func TestCreateMetadataSpan_SpanExactlyAtLimit(t *testing.T) {
	tp := NewNoopTracerProvider()

	md := &mockStructured{
		kind:   "test.kind",
		values: makeValues(consts.MaxMetadataSpanSize),
	}

	ref, err := CreateMetadataSpan(
		context.Background(), tp, &meta.SpanReference{},
		"test.location", "test", nil, md, enums.MetadataScopeStep,
	)
	require.NoError(t, err)
	require.NotNil(t, ref)
}

func TestCreateMetadataSpan_SpanOverLimit(t *testing.T) {
	md := &mockStructured{
		kind:   "test.kind",
		values: makeValues(consts.MaxMetadataSpanSize + 1),
	}

	ref, err := CreateMetadataSpan(
		context.Background(), nil, nil,
		"test.location", "test", nil, md, enums.MetadataScopeStep,
	)
	require.True(t, errors.Is(err, metadata.ErrMetadataSpanTooLarge))
	require.Nil(t, ref)
}

func TestCreateMetadataSpan_EmptyValues(t *testing.T) {
	tp := NewNoopTracerProvider()

	md := &mockStructured{
		kind:   "test.kind",
		values: metadata.Values{},
	}

	ref, err := CreateMetadataSpan(
		context.Background(), tp, &meta.SpanReference{},
		"test.location", "test", nil, md, enums.MetadataScopeStep,
	)
	require.NoError(t, err)
	require.NotNil(t, ref)
}

func TestCreateMetadataSpanFromValues_CumulativeLimitExceeded(t *testing.T) {
	spanSize := 50000
	stateMd := &statev2.Metadata{
		Metrics: statev2.RunMetrics{
			MetadataSize:       consts.MaxRunMetadataSize - spanSize + 1, // just over with new span
			MetadataSizeLoaded: consts.MaxRunMetadataSize - spanSize + 1,
		},
	}

	values := makeValues(spanSize)
	ref, err := CreateMetadataSpanFromValues(
		context.Background(), nil, nil,
		"test.location", "test", stateMd,
		"test.kind", enums.MetadataOpcodeMerge, values, enums.MetadataScopeStep,
	)
	require.ErrorIs(t, err, metadata.ErrRunMetadataSizeExceeded)
	require.Nil(t, ref)
	// In-memory counter should NOT have been incremented
	require.Equal(t, consts.MaxRunMetadataSize-spanSize+1, stateMd.Metrics.MetadataSize)
}

func TestCreateMetadataSpanFromValues_CumulativeLimitAccepted(t *testing.T) {
	tp := NewNoopTracerProvider()

	previousSize := consts.MaxRunMetadataSize - 50000
	stateMd := &statev2.Metadata{
		Metrics: statev2.RunMetrics{
			MetadataSize:       previousSize,
			MetadataSizeLoaded: previousSize,
		},
	}

	spanSize := 40000 // fits within remaining budget
	values := makeValues(spanSize)
	ref, err := CreateMetadataSpanFromValues(
		context.Background(), tp, &meta.SpanReference{},
		"test.location", "test", stateMd,
		"test.kind", enums.MetadataOpcodeMerge, values, enums.MetadataScopeStep,
	)
	require.NoError(t, err)
	require.NotNil(t, ref)
	// In-memory counter should be incremented
	require.Equal(t, previousSize+spanSize, stateMd.Metrics.MetadataSize)
}

func TestCreateMetadataSpanFromValues_CumulativeIncrementAcrossMultipleSpans(t *testing.T) {
	tp := NewNoopTracerProvider()

	// Start near the cumulative limit so a second small span pushes over it
	initialSize := consts.MaxRunMetadataSize - 50000
	stateMd := &statev2.Metadata{
		Metrics: statev2.RunMetrics{
			MetadataSize:       initialSize,
			MetadataSizeLoaded: initialSize,
		},
	}

	spanSize := 40000 // fits within remaining 50000 budget
	values := makeValues(spanSize)

	// First span — accepted
	ref, err := CreateMetadataSpanFromValues(
		context.Background(), tp, &meta.SpanReference{},
		"test.location", "test", stateMd,
		"test.kind", enums.MetadataOpcodeMerge, values, enums.MetadataScopeStep,
	)
	require.NoError(t, err)
	require.NotNil(t, ref)
	require.Equal(t, initialSize+spanSize, stateMd.Metrics.MetadataSize)

	// Second span of same size pushes over the cumulative limit (only 10000 remaining)
	ref, err = CreateMetadataSpanFromValues(
		context.Background(), tp, &meta.SpanReference{},
		"test.location", "test", stateMd,
		"test.kind2", enums.MetadataOpcodeMerge, values, enums.MetadataScopeStep,
	)
	require.ErrorIs(t, err, metadata.ErrRunMetadataSizeExceeded)
	require.Nil(t, ref)
	// Counter should still reflect only the first span
	require.Equal(t, initialSize+spanSize, stateMd.Metrics.MetadataSize)

	// Delta should be the first span only
	delta := stateMd.Metrics.MetadataSize - stateMd.Metrics.MetadataSizeLoaded
	require.Equal(t, spanSize, delta)
}

func TestCreateMetadataSpanFromValues_NotifiesSyncListenersWithStateMetadata(t *testing.T) {
	tp := NewNoopTracerProvider()
	rec := &recordingMetadataListener{}

	accountID, envID, appID, functionID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	stateMd := &statev2.Metadata{ID: statev2.ID{
		RunID:      runID,
		FunctionID: functionID,
		Tenant:     statev2.Tenant{AccountID: accountID, EnvID: envID, AppID: appID},
	}}

	values := metadata.Values{"foo": json.RawMessage(`"bar"`)}
	ref, err := CreateMetadataSpanFromValues(
		context.Background(), tp, &meta.SpanReference{},
		"test.location", "test", stateMd,
		"test.kind", enums.MetadataOpcodeMerge, values, enums.MetadataScopeStep,
		WithMetadataSyncListeners(rec),
	)
	require.NoError(t, err)
	require.NotNil(t, ref)
	require.Len(t, rec.entries, 1)

	got := rec.entries[0]
	require.Equal(t, accountID, got.AccountID)
	require.Equal(t, envID, got.EnvID)
	require.Equal(t, appID, got.AppID)
	require.Equal(t, functionID, got.FunctionID)
	require.Equal(t, runID, got.RunID)
	require.Equal(t, metadata.Kind("test.kind"), got.Kind)
	require.Equal(t, enums.MetadataScopeStep, got.Scope)
	require.Equal(t, values, got.Values)
}

// TestCreateMetadataSpanFromValues_NotifiesSyncListenersWithStepIdentity
// proves step identity (added via an attrs option, mirroring
// pkg/execution/executor's createMetadataSpanOnParent) is picked up
// alongside stateMetadata-sourced tenant/run identity -- the two are
// populated independently in buildSyncMetadataEntry. It also proves
// MetadataEntry.StepID prefers the SDK-facing userland ID
// (meta.Attrs.StepUserlandID) when present, while StepHashedID always
// carries the internal hashed ID (meta.Attrs.StepID) independently.
func TestCreateMetadataSpanFromValues_NotifiesSyncListenersWithStepIdentity(t *testing.T) {
	tp := NewNoopTracerProvider()
	rec := &recordingMetadataListener{}

	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	stateMd := &statev2.Metadata{ID: statev2.ID{RunID: runID}}

	hashedStepID := "hashed-step-1"
	userlandStepID := "my-step"
	stepAttempt := 2
	withStepIdentity := func(cfg *MetadataSpanConfig) {
		meta.AddAttr(cfg.Attrs, meta.Attrs.StepID, &hashedStepID)
		meta.AddAttr(cfg.Attrs, meta.Attrs.StepUserlandID, &userlandStepID)
		meta.AddAttr(cfg.Attrs, meta.Attrs.StepAttempt, &stepAttempt)
	}

	values := metadata.Values{"foo": json.RawMessage(`"bar"`)}
	ref, err := CreateMetadataSpanFromValues(
		context.Background(), tp, &meta.SpanReference{},
		"test.location", "test", stateMd,
		"test.kind", enums.MetadataOpcodeMerge, values, enums.MetadataScopeStep,
		withStepIdentity,
		WithMetadataSyncListeners(rec),
	)
	require.NoError(t, err)
	require.NotNil(t, ref)
	require.Len(t, rec.entries, 1)

	got := rec.entries[0]
	require.Equal(t, runID, got.RunID)
	require.Equal(t, userlandStepID, got.StepID)
	require.Equal(t, hashedStepID, got.StepHashedID)
	require.NotNil(t, got.StepAttempt)
	require.Equal(t, stepAttempt, *got.StepAttempt)
	require.Nil(t, got.StepIndex)
}

// TestCreateMetadataSpanFromValues_StepIDFallsBackToHashedID proves
// entry.StepID falls back to the internal hashed step ID when the userland
// step ID isn't present on attrs -- mirroring opcodes/SDK versions where
// op.Userland is nil, so generatorAttrs never sets meta.Attrs.StepUserlandID.
func TestCreateMetadataSpanFromValues_StepIDFallsBackToHashedID(t *testing.T) {
	tp := NewNoopTracerProvider()
	rec := &recordingMetadataListener{}

	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	stateMd := &statev2.Metadata{ID: statev2.ID{RunID: runID}}

	hashedStepID := "hashed-step-1"
	withHashedOnly := func(cfg *MetadataSpanConfig) {
		meta.AddAttr(cfg.Attrs, meta.Attrs.StepID, &hashedStepID)
	}

	values := metadata.Values{"foo": json.RawMessage(`"bar"`)}
	_, err := CreateMetadataSpanFromValues(
		context.Background(), tp, &meta.SpanReference{},
		"test.location", "test", stateMd,
		"test.kind", enums.MetadataOpcodeMerge, values, enums.MetadataScopeStep,
		withHashedOnly,
		WithMetadataSyncListeners(rec),
	)
	require.NoError(t, err)
	require.Len(t, rec.entries, 1)
	require.Equal(t, hashedStepID, rec.entries[0].StepID)
	require.Equal(t, hashedStepID, rec.entries[0].StepHashedID)
}

// TestCreateMetadataSpanFromValues_StepIdentityEmptyWithoutEitherAttr proves
// both StepID and StepHashedID stay empty when neither the userland nor the
// hashed step ID is present on attrs (run-scoped metadata).
func TestCreateMetadataSpanFromValues_StepIdentityEmptyWithoutEitherAttr(t *testing.T) {
	tp := NewNoopTracerProvider()
	rec := &recordingMetadataListener{}

	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	stateMd := &statev2.Metadata{ID: statev2.ID{RunID: runID}}

	values := metadata.Values{"foo": json.RawMessage(`"bar"`)}
	_, err := CreateMetadataSpanFromValues(
		context.Background(), tp, &meta.SpanReference{},
		"test.location", "test", stateMd,
		"test.kind", enums.MetadataOpcodeMerge, values, enums.MetadataScopeRun,
		WithMetadataSyncListeners(rec),
	)
	require.NoError(t, err)
	require.Len(t, rec.entries, 1)
	require.Empty(t, rec.entries[0].StepID)
	require.Empty(t, rec.entries[0].StepHashedID)
}

// TestCreateMetadataSpanFromValues_NotifiesSyncListenersFromAttrsWhenStateMetadataNil
// proves the fallback path buildSyncMetadataEntry needs for
// pkg/api/apiv1/traces.go's commitSpanMetadata, the one caller with no
// statev2.Metadata at all -- identity there arrives only via an attrs-set
// option (addTenantIDs), mimicked here.
func TestCreateMetadataSpanFromValues_NotifiesSyncListenersFromAttrsWhenStateMetadataNil(t *testing.T) {
	tp := NewNoopTracerProvider()
	rec := &recordingMetadataListener{}

	accountID, envID, appID, functionID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	runID := ulid.MustNew(ulid.Now(), rand.Reader)

	addTenantIDs := func(cfg *MetadataSpanConfig) {
		meta.AddAttr(cfg.Attrs, meta.Attrs.AccountID, &accountID)
		meta.AddAttr(cfg.Attrs, meta.Attrs.EnvID, &envID)
		meta.AddAttr(cfg.Attrs, meta.Attrs.AppID, &appID)
		meta.AddAttr(cfg.Attrs, meta.Attrs.FunctionID, &functionID)
		meta.AddAttr(cfg.Attrs, meta.Attrs.RunID, &runID)
	}

	values := metadata.Values{"foo": json.RawMessage(`"bar"`)}
	ref, err := CreateMetadataSpanFromValues(
		context.Background(), tp, &meta.SpanReference{},
		"test.location", "test", nil,
		"test.kind", enums.MetadataOpcodeMerge, values, enums.MetadataScopeExtendedTrace,
		addTenantIDs,
		WithMetadataSyncListeners(rec),
	)
	require.NoError(t, err)
	require.NotNil(t, ref)
	require.Len(t, rec.entries, 1)
	require.Equal(t, runID, rec.entries[0].RunID)
	require.Equal(t, accountID, rec.entries[0].AccountID)
}

// TestCreateMetadataSpanFromValues_SkipsSyncListenersWithNoRunID proves a
// missing run ID (neither stateMetadata nor attrs supply one) is a no-op,
// not a panic or a garbage row -- there's nothing for such a row to join
// onto.
func TestCreateMetadataSpanFromValues_SkipsSyncListenersWithNoRunID(t *testing.T) {
	tp := NewNoopTracerProvider()
	rec := &recordingMetadataListener{}

	values := metadata.Values{"foo": json.RawMessage(`"bar"`)}
	ref, err := CreateMetadataSpanFromValues(
		context.Background(), tp, &meta.SpanReference{},
		"test.location", "test", nil,
		"test.kind", enums.MetadataOpcodeMerge, values, enums.MetadataScopeExtendedTrace,
		WithMetadataSyncListeners(rec),
	)
	require.NoError(t, err)
	require.NotNil(t, ref)
	require.Empty(t, rec.entries)
}

// orderedTracerProvider wraps NewNoopTracerProvider, recording "create" in a
// shared order log so tests can assert the span is created before any
// listener is dispatched.
type orderedTracerProvider struct {
	TracerProvider
	order *[]string
}

func (o *orderedTracerProvider) CreateSpan(ctx context.Context, name string, opts *CreateSpanOptions) (*meta.SpanReference, error) {
	*o.order = append(*o.order, "create")
	return o.TracerProvider.CreateSpan(ctx, name, opts)
}

// failingTracerProvider fails every CreateSpan call, for proving listeners
// are never notified when span creation itself fails.
type failingTracerProvider struct {
	TracerProvider
	err error
}

func (f *failingTracerProvider) CreateSpan(context.Context, string, *CreateSpanOptions) (*meta.SpanReference, error) {
	return nil, f.err
}

// orderedListener records OnMetadataEntry calls plus their arrival order in
// a shared log, for asserting inline (not concurrent/reordered) dispatch.
type orderedListener struct {
	execution.NoopSyncLifecycleListener
	name    string
	order   *[]string
	entries []execution.MetadataEntry
}

func (o *orderedListener) OnMetadataEntry(_ context.Context, entry execution.MetadataEntry) {
	o.entries = append(o.entries, entry)
	*o.order = append(*o.order, "listener:"+o.name)
}

// TestCreateMetadataSpanFromValues_ScopeTable exercises every
// enums.MetadataScope value (other than Unknown) through
// CreateMetadataSpanFromValues with two registered listeners and
// deliberately different internal/userland step IDs, asserting: the exact
// Parent reference, scope, values, userland step ID (preferred) alongside
// the independently-carried hashed ID, optional index/attempt, a shared
// CreatedAt across both listeners, inline dispatch order (both listeners
// fire, in registration order), and that span creation happens before
// either listener is notified.
func TestCreateMetadataSpanFromValues_ScopeTable(t *testing.T) {
	scopes := []enums.MetadataScope{
		enums.MetadataScopeRun,
		enums.MetadataScopeStep,
		enums.MetadataScopeStepAttempt,
		enums.MetadataScopeExtendedTrace,
		enums.MetadataScopeRequest,
	}

	for _, scope := range scopes {
		t.Run(scope.String(), func(t *testing.T) {
			order := []string{}
			tp := &orderedTracerProvider{TracerProvider: NewNoopTracerProvider(), order: &order}
			l1 := &orderedListener{name: "l1", order: &order}
			l2 := &orderedListener{name: "l2", order: &order}

			accountID, envID, appID, functionID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
			runID := ulid.MustNew(ulid.Now(), rand.Reader)
			stateMd := &statev2.Metadata{ID: statev2.ID{
				RunID:      runID,
				FunctionID: functionID,
				Tenant:     statev2.Tenant{AccountID: accountID, EnvID: envID, AppID: appID},
			}}

			hashedStepID := "hashed-" + scope.String()
			userlandStepID := "userland-" + scope.String()
			stepIndex := 3
			stepAttempt := 1
			withStepIdentity := func(cfg *MetadataSpanConfig) {
				meta.AddAttr(cfg.Attrs, meta.Attrs.StepID, &hashedStepID)
				meta.AddAttr(cfg.Attrs, meta.Attrs.StepUserlandID, &userlandStepID)
				meta.AddAttr(cfg.Attrs, meta.Attrs.StepUserlandIndex, &stepIndex)
				meta.AddAttr(cfg.Attrs, meta.Attrs.StepAttempt, &stepAttempt)
			}

			parent := &meta.SpanReference{TraceParent: "00-" + scope.String() + "-parent-00"}
			values := metadata.Values{"foo": json.RawMessage(`"bar"`)}
			ref, err := CreateMetadataSpanFromValues(
				context.Background(), tp, parent,
				"test.location", "test", stateMd,
				"test.kind", enums.MetadataOpcodeMerge, values, scope,
				withStepIdentity,
				WithMetadataSyncListeners(l1, l2),
			)
			require.NoError(t, err)
			require.NotNil(t, ref)

			require.Len(t, l1.entries, 1)
			require.Len(t, l2.entries, 1)
			require.Equal(t, []string{"create", "listener:l1", "listener:l2"}, order)

			for _, entry := range []execution.MetadataEntry{l1.entries[0], l2.entries[0]} {
				require.Same(t, parent, entry.Parent)
				require.Equal(t, scope, entry.Scope)
				require.Equal(t, values, entry.Values)
				require.Equal(t, runID, entry.RunID)
				require.Equal(t, accountID, entry.AccountID)
				require.Equal(t, userlandStepID, entry.StepID)
				require.Equal(t, hashedStepID, entry.StepHashedID)
				require.NotNil(t, entry.StepIndex)
				require.Equal(t, stepIndex, *entry.StepIndex)
				require.NotNil(t, entry.StepAttempt)
				require.Equal(t, stepAttempt, *entry.StepAttempt)
			}
			require.Equal(t, l1.entries[0].CreatedAt, l2.entries[0].CreatedAt)
		})
	}
}

// TestCreateMetadataSpanFromValues_SpanTooLargeSkipsListeners proves the
// per-span size gate rejects before ever creating a span or notifying a
// listener.
func TestCreateMetadataSpanFromValues_SpanTooLargeSkipsListeners(t *testing.T) {
	tp := NewNoopTracerProvider()
	rec := &recordingMetadataListener{}
	stateMd := &statev2.Metadata{ID: statev2.ID{RunID: ulid.MustNew(ulid.Now(), rand.Reader)}}

	values := makeValues(consts.MaxMetadataSpanSize + 1)
	ref, err := CreateMetadataSpanFromValues(
		context.Background(), tp, &meta.SpanReference{},
		"test.location", "test", stateMd,
		"test.kind", enums.MetadataOpcodeMerge, values, enums.MetadataScopeStep,
		WithMetadataSyncListeners(rec),
	)
	require.ErrorIs(t, err, metadata.ErrMetadataSpanTooLarge)
	require.Nil(t, ref)
	require.Empty(t, rec.entries)
}

// TestCreateMetadataSpanFromValues_CumulativeLimitSkipsListeners proves the
// per-run cumulative size gate rejects without creating a span or notifying
// a listener.
func TestCreateMetadataSpanFromValues_CumulativeLimitSkipsListeners(t *testing.T) {
	tp := NewNoopTracerProvider()
	rec := &recordingMetadataListener{}
	spanSize := 50000
	stateMd := &statev2.Metadata{
		ID: statev2.ID{RunID: ulid.MustNew(ulid.Now(), rand.Reader)},
		Metrics: statev2.RunMetrics{
			MetadataSize:       consts.MaxRunMetadataSize - spanSize + 1,
			MetadataSizeLoaded: consts.MaxRunMetadataSize - spanSize + 1,
		},
	}

	values := makeValues(spanSize)
	ref, err := CreateMetadataSpanFromValues(
		context.Background(), tp, &meta.SpanReference{},
		"test.location", "test", stateMd,
		"test.kind", enums.MetadataOpcodeMerge, values, enums.MetadataScopeStep,
		WithMetadataSyncListeners(rec),
	)
	require.ErrorIs(t, err, metadata.ErrRunMetadataSizeExceeded)
	require.Nil(t, ref)
	require.Empty(t, rec.entries)
}

// TestCreateMetadataSpan_SerializeFailureSkipsListeners proves a
// metadata.Structured that fails to serialize never reaches span creation
// or listener dispatch.
func TestCreateMetadataSpan_SerializeFailureSkipsListeners(t *testing.T) {
	tp := NewNoopTracerProvider()
	rec := &recordingMetadataListener{}
	stateMd := &statev2.Metadata{ID: statev2.ID{RunID: ulid.MustNew(ulid.Now(), rand.Reader)}}

	md := &mockStructured{kind: "test.kind", serializeErr: errors.New("bad json")}
	ref, err := CreateMetadataSpan(
		context.Background(), tp, &meta.SpanReference{},
		"test.location", "test", stateMd, md, enums.MetadataScopeStep,
		WithMetadataSyncListeners(rec),
	)
	require.Error(t, err)
	require.Nil(t, ref)
	require.Empty(t, rec.entries)
}

// TestCreateMetadataSpanFromValues_TracerFailureSkipsListeners proves a
// tracer-provider failure creating the span skips listener dispatch
// entirely -- a row must never be reported for a span that doesn't exist.
func TestCreateMetadataSpanFromValues_TracerFailureSkipsListeners(t *testing.T) {
	tp := &failingTracerProvider{err: errors.New("tracer backend unavailable")}
	rec := &recordingMetadataListener{}
	stateMd := &statev2.Metadata{ID: statev2.ID{RunID: ulid.MustNew(ulid.Now(), rand.Reader)}}

	values := metadata.Values{"foo": json.RawMessage(`"bar"`)}
	ref, err := CreateMetadataSpanFromValues(
		context.Background(), tp, &meta.SpanReference{},
		"test.location", "test", stateMd,
		"test.kind", enums.MetadataOpcodeMerge, values, enums.MetadataScopeStep,
		WithMetadataSyncListeners(rec),
	)
	require.Error(t, err)
	require.Nil(t, ref)
	require.Empty(t, rec.entries)
}

// capturingTracerProvider records the attrs of every created span.
type capturingTracerProvider struct {
	TracerProvider
	attrs []*meta.SerializableAttrs
}

func (c *capturingTracerProvider) CreateSpan(ctx context.Context, name string, opts *CreateSpanOptions) (*meta.SpanReference, error) {
	c.attrs = append(c.attrs, opts.Attributes)
	return c.TracerProvider.CreateSpan(ctx, name, opts)
}

func newCapturingTracerProvider() *capturingTracerProvider {
	return &capturingTracerProvider{TracerProvider: NewNoopTracerProvider()}
}

// capturedMetadata is the kind, op & values written to a captured span.
type capturedMetadata struct {
	Kind   metadata.Kind
	Op     metadata.Opcode
	Values metadata.Values
}

func (c *capturingTracerProvider) metadata(t *testing.T) []capturedMetadata {
	t.Helper()

	ret := make([]capturedMetadata, 0, len(c.attrs))
	for _, attrs := range c.attrs {
		kind, ok := meta.GetAttr(attrs, meta.Attrs.MetadataKind)
		require.True(t, ok)
		op, ok := meta.GetAttr(attrs, meta.Attrs.MetadataOp)
		require.True(t, ok)
		values, ok := meta.GetAttr(attrs, meta.Attrs.Metadata)
		require.True(t, ok)
		ret = append(ret, capturedMetadata{Kind: *kind, Op: *op, Values: *values})
	}
	return ret
}

func TestCreateMetadataSpanFromValues_AlwaysWritesSet(t *testing.T) {
	for _, op := range enums.MetadataOpcodeValues() {
		t.Run(op.String(), func(t *testing.T) {
			tp := newCapturingTracerProvider()
			values := metadata.Values{"foo": json.RawMessage(`"bar"`)}

			ref, err := CreateMetadataSpanFromValues(
				context.Background(), tp, &meta.SpanReference{},
				"test.location", "test", nil,
				"userland.thing", op, values, enums.MetadataScopeStep,
			)
			require.NoError(t, err)
			require.NotNil(t, ref)
			require.Equal(t, []capturedMetadata{
				{Kind: "userland.thing", Op: enums.MetadataOpcodeSet, Values: values},
			}, tp.metadata(t))
		})
	}
}

func TestCreateMetadataSpan_AlwaysWritesSet(t *testing.T) {
	tp := newCapturingTracerProvider()
	md := &mockStructured{kind: "test.kind", values: metadata.Values{"foo": json.RawMessage(`1`)}}

	_, err := CreateMetadataSpan(
		context.Background(), tp, &meta.SpanReference{},
		"test.location", "test", nil, md, enums.MetadataScopeStep,
	)
	require.NoError(t, err)
	got := tp.metadata(t)
	require.Len(t, got, 1)
	require.Equal(t, enums.MetadataOpcodeSet, got[0].Op)
}

func TestIncomingOp(t *testing.T) {
	require.Equal(t, enums.MetadataOpcodeMerge, incomingOp(metadata.Update{RawUpdate: metadata.RawUpdate{Op: enums.MetadataOpcodeMerge}}))
	require.Equal(t, enums.MetadataOpcodeDelete, incomingOp(metadata.ScopedUpdate{Update: metadata.Update{RawUpdate: metadata.RawUpdate{Op: enums.MetadataOpcodeDelete}}}))
	require.Equal(t, enums.MetadataOpcodeSet, incomingOp(structuredWithoutOp{}))
}

// structuredWithoutOp is server built metadata, which has no op.
type structuredWithoutOp struct{}

func (structuredWithoutOp) Kind() metadata.Kind                 { return "inngest.test" }
func (structuredWithoutOp) Serialize() (metadata.Values, error) { return metadata.Values{}, nil }

func TestMetadataKindTag(t *testing.T) {
	tests := map[metadata.Kind]string{
		"userland.foo":                 "userland.*",
		"inngest.score.accuracy":       "inngest.score.*",
		"inngest.warning.size":         "inngest.warning.*",
		metadata.KindInngestScore:      "inngest.score",
		metadata.KindInngestWarnings:   "inngest.warnings",
		metadata.KindInngestExperiment: "inngest.experiment",
	}
	for kind, want := range tests {
		require.Equal(t, want, metadataKindTag(kind), kind)
	}
}

func TestCreateMetadataSpanFromValues_SplitsLegacyScores(t *testing.T) {
	tp := newCapturingTracerProvider()
	rec := &recordingMetadataListener{}
	stateMd := &statev2.Metadata{ID: statev2.ID{RunID: ulid.MustNew(ulid.Now(), rand.Reader)}}

	values := metadata.Values{
		"accuracy": json.RawMessage(`{"value":0.95}`),
		"passed":   json.RawMessage(`{"value":true}`),
	}
	ref, err := CreateMetadataSpanFromValues(
		context.Background(), tp, &meta.SpanReference{},
		"test.location", "test", stateMd,
		metadata.KindInngestScore, enums.MetadataOpcodeMerge, values, enums.MetadataScopeRun,
		WithMetadataSyncListeners(rec),
	)
	require.NoError(t, err)
	require.NotNil(t, ref)

	require.Equal(t, []capturedMetadata{
		{Kind: "inngest.score.accuracy", Op: enums.MetadataOpcodeSet, Values: metadata.Values{"value": json.RawMessage(`0.95`)}},
		{Kind: "inngest.score.passed", Op: enums.MetadataOpcodeSet, Values: metadata.Values{"value": json.RawMessage(`true`)}},
	}, tp.metadata(t))

	// The sync listeners fire once per split span.
	require.Len(t, rec.entries, 2)
	require.Equal(t, metadata.Kind("inngest.score.accuracy"), rec.entries[0].Kind)
	require.Equal(t, metadata.Kind("inngest.score.passed"), rec.entries[1].Kind)

	// The cumulative size counts every split span.
	require.Equal(t, len("value")*2+len("0.95")+len("true"), stateMd.Metrics.MetadataSize)
}

func TestCreateMetadataSpanFromValues_SplitsLegacyWarnings(t *testing.T) {
	tp := newCapturingTracerProvider()

	values := metadata.Values{
		"size": json.RawMessage(`"too big"`),
		"auth": json.RawMessage(`"nope"`),
	}
	_, err := CreateMetadataSpanFromValues(
		context.Background(), tp, &meta.SpanReference{},
		"test.location", "test", nil,
		metadata.KindInngestWarnings, enums.MetadataOpcodeMerge, values, enums.MetadataScopeStep,
	)
	require.NoError(t, err)

	require.Equal(t, []capturedMetadata{
		{Kind: "inngest.warning.auth", Op: enums.MetadataOpcodeSet, Values: metadata.Values{"auth": json.RawMessage(`"nope"`)}},
		{Kind: "inngest.warning.size", Op: enums.MetadataOpcodeSet, Values: metadata.Values{"size": json.RawMessage(`"too big"`)}},
	}, tp.metadata(t))
}

func TestCreateMetadataSpanFromValues_SplitSpanSizeLimit(t *testing.T) {
	// 2 scores that are each under the per span limit, but over it together.
	half := consts.MaxMetadataSpanSize/2 + 100
	pad := strings.Repeat("x", half)
	values := metadata.Values{
		"a": json.RawMessage(`{"value":"` + pad + `"}`),
		"b": json.RawMessage(`{"value":"` + pad + `"}`),
	}
	require.Greater(t, values.Size(), consts.MaxMetadataSpanSize)

	tp := newCapturingTracerProvider()
	_, err := CreateMetadataSpanFromValues(
		context.Background(), tp, &meta.SpanReference{},
		"test.location", "test", nil,
		metadata.KindInngestScore, enums.MetadataOpcodeMerge, values, enums.MetadataScopeRun,
	)
	require.NoError(t, err)
	require.Len(t, tp.metadata(t), 2)

	// A single split span over the limit fails the whole write.
	values["c"] = json.RawMessage(`{"value":"` + strings.Repeat("x", consts.MaxMetadataSpanSize) + `"}`)
	tp = newCapturingTracerProvider()
	_, err = CreateMetadataSpanFromValues(
		context.Background(), tp, &meta.SpanReference{},
		"test.location", "test", nil,
		metadata.KindInngestScore, enums.MetadataOpcodeMerge, values, enums.MetadataScopeRun,
	)
	require.ErrorIs(t, err, metadata.ErrMetadataSpanTooLarge)
	require.Empty(t, tp.attrs)
}

func TestCreateMetadataSpanFromValues_SplitCumulativeLimit(t *testing.T) {
	values := metadata.Values{
		"a": json.RawMessage(`{"value":1}`),
		"b": json.RawMessage(`{"value":2}`),
	}
	// Each split span is {"value": n}, IE 6 bytes. Leave room for only one.
	initialSize := consts.MaxRunMetadataSize - 6
	stateMd := &statev2.Metadata{
		Metrics: statev2.RunMetrics{
			MetadataSize:       initialSize,
			MetadataSizeLoaded: initialSize,
		},
	}

	tp := newCapturingTracerProvider()
	_, err := CreateMetadataSpanFromValues(
		context.Background(), tp, &meta.SpanReference{},
		"test.location", "test", stateMd,
		metadata.KindInngestScore, enums.MetadataOpcodeMerge, values, enums.MetadataScopeRun,
	)
	require.ErrorIs(t, err, metadata.ErrRunMetadataSizeExceeded)
	require.Empty(t, tp.attrs)
	require.Equal(t, initialSize, stateMd.Metrics.MetadataSize)

	delete(values, "b")
	_, err = CreateMetadataSpanFromValues(
		context.Background(), tp, &meta.SpanReference{},
		"test.location", "test", stateMd,
		metadata.KindInngestScore, enums.MetadataOpcodeMerge, values, enums.MetadataScopeRun,
	)
	require.NoError(t, err)
	require.Equal(t, consts.MaxRunMetadataSize, stateMd.Metrics.MetadataSize)
}

func TestCreateMetadataSpanFromValues_SplitRollsBackUnwrittenSpans(t *testing.T) {
	tp := &failAfterTracerProvider{TracerProvider: NewNoopTracerProvider(), ok: 1}
	stateMd := &statev2.Metadata{}

	values := metadata.Values{
		"a": json.RawMessage(`{"value":1}`),
		"b": json.RawMessage(`{"value":2}`),
	}
	_, err := CreateMetadataSpanFromValues(
		context.Background(), tp, &meta.SpanReference{},
		"test.location", "test", stateMd,
		metadata.KindInngestScore, enums.MetadataOpcodeMerge, values, enums.MetadataScopeRun,
	)
	require.Error(t, err)
	// Only the first span was created, so only its size is kept.
	require.Equal(t, 6, stateMd.Metrics.MetadataSize)
}

func TestCreateMetadataSpanFromValues_EmptyLegacyWriteIsNoop(t *testing.T) {
	tp := newCapturingTracerProvider()
	ref, err := CreateMetadataSpanFromValues(
		context.Background(), tp, &meta.SpanReference{},
		"test.location", "test", nil,
		metadata.KindInngestScore, enums.MetadataOpcodeMerge, metadata.Values{}, enums.MetadataScopeRun,
	)
	require.NoError(t, err)
	require.Nil(t, ref)
	require.Empty(t, tp.attrs)
}

// failAfterTracerProvider creates ok spans, then fails every CreateSpan call.
type failAfterTracerProvider struct {
	TracerProvider
	ok int
}

func (f *failAfterTracerProvider) CreateSpan(ctx context.Context, name string, opts *CreateSpanOptions) (*meta.SpanReference, error) {
	if f.ok == 0 {
		return nil, errors.New("tracer backend unavailable")
	}
	f.ok--
	return f.TracerProvider.CreateSpan(ctx, name, opts)
}
