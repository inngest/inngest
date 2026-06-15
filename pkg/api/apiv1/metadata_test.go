package apiv1

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/api/apiv1/apiv1auth"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// metadataTraceReader stubs the trace reader for legacy-path tests.
type metadataTraceReader struct {
	cqrs.TraceReader
	span *cqrs.OtelSpan
}

func (r metadataTraceReader) GetRunSpanByRunID(context.Context, ulid.ULID, uuid.UUID, uuid.UUID) (*cqrs.OtelSpan, error) {
	return r.span, nil
}

func (r metadataTraceReader) GetLatestExecutionSpanByStepID(context.Context, ulid.ULID, string, uuid.UUID, uuid.UUID) (*cqrs.OtelSpan, error) {
	return r.span, nil
}

type metadataTracerProvider struct {
	t      *testing.T
	wantID statev2.ID
	called bool
}

type metadataStateLoader struct {
	statev2.RunService
	loadMetadata func(context.Context, statev2.ID) (statev2.Metadata, error)
}

func (s metadataStateLoader) LoadMetadata(ctx context.Context, id statev2.ID, _ ...statev2.LoadMetadataOption) (statev2.Metadata, error) {
	if s.loadMetadata != nil {
		return s.loadMetadata(ctx, id)
	}
	return statev2.Metadata{}, errors.New("unexpected LoadMetadata call")
}

func (p *metadataTracerProvider) CreateSpan(_ context.Context, _ string, opts *tracing.CreateSpanOptions) (*meta.SpanReference, error) {
	p.called = true

	require.NotNil(p.t, opts.Metadata)
	require.Equal(p.t, p.wantID, opts.Metadata.ID)
	require.NotPanics(p.t, func() {
		_ = opts.Metadata.Config.DebugRunID()
		_ = opts.Metadata.Config.DebugSessionID()
	})

	return &meta.SpanReference{}, nil
}

func (p *metadataTracerProvider) CreateDroppableSpan(context.Context, string, *tracing.CreateSpanOptions) (*tracing.DroppableSpan, error) {
	return nil, errors.New("unexpected CreateDroppableSpan call")
}

func (p *metadataTracerProvider) UpdateSpan(context.Context, *tracing.UpdateSpanOptions) error {
	return errors.New("unexpected UpdateSpan call")
}

// newRunID returns a ULID with the current time (new path).
func newRunID() ulid.ULID {
	return ulid.Make()
}

// preDeterministicSpanIDRunID returns a ULID timestamped before deterministicSpanIDCutoff (legacy path).
func preDeterministicSpanIDRunID() ulid.ULID {
	ts := ulid.Timestamp(deterministicSpanIDCutoff.Add(-time.Hour))
	return ulid.MustNew(ts, rand.New(rand.NewSource(0)))
}

// TestAddRunMetadataNewPathUsesStateForTenantIDs verifies that on the new path
// the FunctionID and AppID attached to the metadata span come from the state
// store, not from a ClickHouse span query.
func TestAddRunMetadataNewPathUsesStateForTenantIDs(t *testing.T) {
	ctx := t.Context()
	auth, err := apiv1auth.NilAuthFinder(ctx)
	require.NoError(t, err)

	runID := newRunID()
	functionID := uuid.New()
	appID := uuid.New()
	wantID := statev2.ID{
		RunID:      runID,
		FunctionID: functionID,
		Tenant: statev2.Tenant{
			AppID:     appID,
			EnvID:     auth.WorkspaceID(),
			AccountID: auth.AccountID(),
		},
	}

	tp := &metadataTracerProvider{t: t, wantID: wantID}
	r := router{API: &API{opts: Opts{
		State: metadataStateLoader{loadMetadata: func(_ context.Context, _ statev2.ID) (statev2.Metadata, error) {
			return statev2.Metadata{ID: wantID}, nil
		}},
		TracerProvider: tp,
	}}}

	err = r.AddRunMetadata(ctx, auth, runID, &AddRunMetadataRequest{
		Metadata: []metadata.Update{{RawUpdate: metadata.RawUpdate{
			Kind:   "userland.scores",
			Op:     enums.MetadataOpcodeMerge,
			Values: metadata.Values{"score": json.RawMessage(`{"value":1}`)},
		}}},
	})
	require.NoError(t, err)
	require.True(t, tp.called)
}

// TestAddRunMetadataNewPathFallbackToPartialIDWhenStateMissing verifies that
// when the state store has no record for the run, the metadata span is still
// created using the partial ID (zero FunctionID/AppID).
func TestAddRunMetadataNewPathFallbackToPartialIDWhenStateMissing(t *testing.T) {
	ctx := t.Context()
	auth, err := apiv1auth.NilAuthFinder(ctx)
	require.NoError(t, err)

	runID := newRunID()
	wantID := statev2.ID{
		RunID: runID,
		Tenant: statev2.Tenant{
			EnvID:     auth.WorkspaceID(),
			AccountID: auth.AccountID(),
		},
	}

	tp := &metadataTracerProvider{t: t, wantID: wantID}
	r := router{API: &API{opts: Opts{
		State: metadataStateLoader{loadMetadata: func(context.Context, statev2.ID) (statev2.Metadata, error) {
			return statev2.Metadata{}, statev2.ErrMetadataNotFound
		}},
		TracerProvider: tp,
	}}}

	err = r.AddRunMetadata(ctx, auth, runID, &AddRunMetadataRequest{
		Metadata: []metadata.Update{{RawUpdate: metadata.RawUpdate{
			Kind:   "userland.scores",
			Op:     enums.MetadataOpcodeMerge,
			Values: metadata.Values{"score": json.RawMessage(`{"value":1}`)},
		}}},
	})
	require.NoError(t, err)
	require.True(t, tp.called)
}

func TestAddRunMetadataReturnsTransientStateLoadError(t *testing.T) {
	ctx := t.Context()
	auth, err := apiv1auth.NilAuthFinder(ctx)
	require.NoError(t, err)

	runID := newRunID()
	loadErr := errors.New("temporary state load failure")
	tp := &metadataTracerProvider{t: t}
	r := router{API: &API{opts: Opts{
		TracerProvider: tp,
		State: metadataStateLoader{loadMetadata: func(context.Context, statev2.ID) (statev2.Metadata, error) {
			return statev2.Metadata{}, loadErr
		}},
	}}}

	err = r.AddRunMetadata(ctx, auth, runID, &AddRunMetadataRequest{
		Metadata: []metadata.Update{{RawUpdate: metadata.RawUpdate{
			Kind:   "userland.scores",
			Op:     enums.MetadataOpcodeMerge,
			Values: metadata.Values{"score": json.RawMessage(`{"value":1}`)},
		}}},
	})
	require.ErrorIs(t, err, loadErr)
	var publicErr publicerr.Error
	require.ErrorAs(t, err, &publicErr)
	require.Equal(t, 500, publicErr.Status)
	require.Equal(t, "Unable to load run metadata", publicErr.Message)
	require.False(t, tp.called)
}

func TestAddRunMetadataAllowsScoreMetadata(t *testing.T) {
	ctx := t.Context()
	auth, err := apiv1auth.NilAuthFinder(ctx)
	require.NoError(t, err)

	runID := newRunID()
	functionID := uuid.New()
	appID := uuid.New()
	stepID := "score-step"
	wantID := statev2.ID{
		RunID:      runID,
		FunctionID: functionID,
		Tenant: statev2.Tenant{
			AppID:     appID,
			EnvID:     auth.WorkspaceID(),
			AccountID: auth.AccountID(),
		},
	}

	tp := &metadataTracerProvider{t: t, wantID: wantID}
	r := router{API: &API{opts: Opts{
		State: metadataStateLoader{loadMetadata: func(_ context.Context, _ statev2.ID) (statev2.Metadata, error) {
			return statev2.Metadata{ID: wantID}, nil
		}},
		TracerProvider: tp,
	}}}

	err = r.AddRunMetadata(ctx, auth, runID, &AddRunMetadataRequest{
		Target: RunMetadataTarget{StepID: &stepID},
		Metadata: []metadata.Update{{RawUpdate: metadata.RawUpdate{
			Kind:   metadata.KindInngestScore + ".passed",
			Op:     enums.MetadataOpcodeMerge,
			Values: metadata.Values{"value": json.RawMessage(`true`)},
		}}},
	})
	require.NoError(t, err)
	require.True(t, tp.called)
}

func TestAddRunMetadataRejectsInvalidScoreMetadata(t *testing.T) {
	ctx := t.Context()
	auth, err := apiv1auth.NilAuthFinder(ctx)
	require.NoError(t, err)

	runID := newRunID()
	r := router{API: &API{opts: Opts{
		State: metadataStateLoader{loadMetadata: func(context.Context, statev2.ID) (statev2.Metadata, error) {
			require.Fail(t, "LoadMetadata should not be called for invalid metadata")
			return statev2.Metadata{}, nil
		}},
	}}}

	err = r.AddRunMetadata(ctx, auth, runID, &AddRunMetadataRequest{
		Metadata: []metadata.Update{{RawUpdate: metadata.RawUpdate{
			Kind:   metadata.KindInngestScore + ".score",
			Op:     enums.MetadataOpcodeMerge,
			Values: metadata.Values{"value": json.RawMessage(`{"nested":1}`)},
		}}},
	})
	require.ErrorIs(t, err, metadata.ErrScoreValueInvalid)
}

func TestAddRunMetadataAllowsRunScopedScoreMetadata(t *testing.T) {
	ctx := t.Context()
	auth, err := apiv1auth.NilAuthFinder(ctx)
	require.NoError(t, err)

	runID := newRunID()
	functionID := uuid.New()
	appID := uuid.New()
	wantID := statev2.ID{
		RunID:      runID,
		FunctionID: functionID,
		Tenant: statev2.Tenant{
			AppID:     appID,
			EnvID:     auth.WorkspaceID(),
			AccountID: auth.AccountID(),
		},
	}

	tp := &metadataTracerProvider{t: t, wantID: wantID}
	r := router{API: &API{opts: Opts{
		State: metadataStateLoader{loadMetadata: func(_ context.Context, _ statev2.ID) (statev2.Metadata, error) {
			return statev2.Metadata{ID: wantID}, nil
		}},
		TracerProvider: tp,
	}}}

	err = r.AddRunMetadata(ctx, auth, runID, &AddRunMetadataRequest{
		Metadata: []metadata.Update{{RawUpdate: metadata.RawUpdate{
			Kind:   metadata.KindInngestScore + ".accuracy",
			Op:     enums.MetadataOpcodeMerge,
			Values: metadata.Values{"value": json.RawMessage(`1`)},
		}}},
	})
	require.NoError(t, err)
	require.True(t, tp.called)
}

func TestAddRunMetadataRejectsDisallowedInngestKind(t *testing.T) {
	ctx := t.Context()
	auth, err := apiv1auth.NilAuthFinder(ctx)
	require.NoError(t, err)

	runID := newRunID()
	r := router{API: &API{opts: Opts{}}}

	err = r.AddRunMetadata(ctx, auth, runID, &AddRunMetadataRequest{
		Metadata: []metadata.Update{{RawUpdate: metadata.RawUpdate{
			Kind:   "inngest.internal",
			Op:     enums.MetadataOpcodeMerge,
			Values: metadata.Values{"score": json.RawMessage(`1`)},
		}}},
	})
	require.ErrorIs(t, err, metadata.ErrKindNotAllowed)
}

// TestAddRunMetadataLegacyPathForOldRuns verifies that run IDs created before
// deterministicSpanIDCutoff are routed through the legacy ClickHouse query
// path (getParentSpan with retry).
func TestAddRunMetadataLegacyPathForOldRuns(t *testing.T) {
	ctx := t.Context()
	auth, err := apiv1auth.NilAuthFinder(ctx)
	require.NoError(t, err)

	runID := preDeterministicSpanIDRunID()
	functionID := uuid.New()
	appID := uuid.New()
	wantID := statev2.ID{
		RunID:      runID,
		FunctionID: functionID,
		Tenant: statev2.Tenant{
			AppID:     appID,
			EnvID:     auth.WorkspaceID(),
			AccountID: auth.AccountID(),
		},
	}

	tp := &metadataTracerProvider{t: t, wantID: wantID}
	r := router{API: &API{opts: Opts{
		TraceReader: metadataTraceReader{span: &cqrs.OtelSpan{
			RawOtelSpan: cqrs.RawOtelSpan{
				TraceID: "00000000000000000000000000000001",
				SpanID:  "0000000000000001",
			},
			RunID:      runID,
			FunctionID: functionID,
			AppID:      appID,
		}},
		TracerProvider: tp,
	}}}

	err = r.AddRunMetadata(ctx, auth, runID, &AddRunMetadataRequest{
		Metadata: []metadata.Update{{RawUpdate: metadata.RawUpdate{
			Kind:   "userland.scores",
			Op:     enums.MetadataOpcodeMerge,
			Values: metadata.Values{"score": json.RawMessage(`{"value":1}`)},
		}}},
	})
	require.NoError(t, err)
	require.True(t, tp.called)
}
