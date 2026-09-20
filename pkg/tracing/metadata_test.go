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
