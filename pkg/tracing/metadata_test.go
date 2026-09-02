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
// populated independently in buildSyncMetadataEntry.
func TestCreateMetadataSpanFromValues_NotifiesSyncListenersWithStepIdentity(t *testing.T) {
	tp := NewNoopTracerProvider()
	rec := &recordingMetadataListener{}

	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	stateMd := &statev2.Metadata{ID: statev2.ID{RunID: runID}}

	stepID := "step-hash-1"
	stepAttempt := 2
	withStepIdentity := func(cfg *MetadataSpanConfig) {
		meta.AddAttr(cfg.Attrs, meta.Attrs.StepID, &stepID)
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
	require.Equal(t, stepID, got.StepID)
	require.NotNil(t, got.StepAttempt)
	require.Equal(t, stepAttempt, *got.StepAttempt)
	require.Nil(t, got.StepIndex)
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
