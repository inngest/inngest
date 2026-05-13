package apiv1

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

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

type metadataFallbackTraceReader struct {
	cqrs.TraceReader
	span *cqrs.OtelSpan
}

func (r metadataFallbackTraceReader) GetRunSpanByRunID(context.Context, ulid.ULID, uuid.UUID, uuid.UUID) (*cqrs.OtelSpan, error) {
	return r.span, nil
}

func (r metadataFallbackTraceReader) GetLatestExecutionSpanByStepID(context.Context, ulid.ULID, string, uuid.UUID, uuid.UUID) (*cqrs.OtelSpan, error) {
	return r.span, nil
}

type metadataFallbackTracerProvider struct {
	t      *testing.T
	wantID statev2.ID
	called bool
}

type metadataStateLoader struct {
	statev2.RunService
	loadMetadata func(context.Context, statev2.ID) (statev2.Metadata, error)
}

func (s metadataStateLoader) LoadMetadata(ctx context.Context, id statev2.ID) (statev2.Metadata, error) {
	if s.loadMetadata != nil {
		return s.loadMetadata(ctx, id)
	}

	return statev2.Metadata{}, errors.New("unexpected LoadMetadata call")
}

func (p *metadataFallbackTracerProvider) CreateSpan(_ context.Context, _ string, opts *tracing.CreateSpanOptions) (*meta.SpanReference, error) {
	p.called = true

	require.NotNil(p.t, opts.Metadata)
	require.Equal(p.t, p.wantID, opts.Metadata.ID)
	require.NotPanics(p.t, func() {
		_ = opts.Metadata.Config.DebugRunID()
		_ = opts.Metadata.Config.DebugSessionID()
	})

	return &meta.SpanReference{}, nil
}

func (p *metadataFallbackTracerProvider) CreateDroppableSpan(context.Context, string, *tracing.CreateSpanOptions) (*tracing.DroppableSpan, error) {
	return nil, errors.New("unexpected CreateDroppableSpan call")
}

func (p *metadataFallbackTracerProvider) UpdateSpan(context.Context, *tracing.UpdateSpanOptions) error {
	return errors.New("unexpected UpdateSpan call")
}

func TestAddRunMetadataFallbackInitializesStateMetadata(t *testing.T) {
	ctx := t.Context()
	auth, err := apiv1auth.NilAuthFinder(ctx)
	require.NoError(t, err)

	runID := ulid.Make()
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

	tp := &metadataFallbackTracerProvider{t: t, wantID: wantID}
	r := router{API: &API{opts: Opts{
		TraceReader: metadataFallbackTraceReader{span: &cqrs.OtelSpan{
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

func TestAddRunMetadataFallbackInitializesMissingStateMetadata(t *testing.T) {
	ctx := t.Context()
	auth, err := apiv1auth.NilAuthFinder(ctx)
	require.NoError(t, err)

	runID := ulid.Make()
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

	tp := &metadataFallbackTracerProvider{t: t, wantID: wantID}
	r := router{API: &API{opts: Opts{
		TraceReader: metadataFallbackTraceReader{span: &cqrs.OtelSpan{
			RawOtelSpan: cqrs.RawOtelSpan{
				TraceID: "00000000000000000000000000000001",
				SpanID:  "0000000000000001",
			},
			RunID:      runID,
			FunctionID: functionID,
			AppID:      appID,
		}},
		TracerProvider: tp,
		State: metadataStateLoader{loadMetadata: func(context.Context, statev2.ID) (statev2.Metadata, error) {
			return statev2.Metadata{}, statev2.ErrMetadataNotFound
		}},
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

	runID := ulid.Make()
	loadErr := errors.New("temporary state load failure")
	tp := &metadataFallbackTracerProvider{t: t}
	r := router{API: &API{opts: Opts{
		TraceReader: metadataFallbackTraceReader{span: &cqrs.OtelSpan{
			RawOtelSpan: cqrs.RawOtelSpan{
				TraceID: "00000000000000000000000000000001",
				SpanID:  "0000000000000001",
			},
			RunID:      runID,
			FunctionID: uuid.New(),
			AppID:      uuid.New(),
		}},
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

	runID := ulid.Make()
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

	tp := &metadataFallbackTracerProvider{t: t, wantID: wantID}
	r := router{API: &API{opts: Opts{
		TraceReader: metadataFallbackTraceReader{span: &cqrs.OtelSpan{
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
		Target: RunMetadataTarget{StepID: &stepID},
		Metadata: []metadata.Update{{RawUpdate: metadata.RawUpdate{
			Kind:   metadata.KindInngestScore,
			Op:     enums.MetadataOpcodeMerge,
			Values: metadata.Values{"accuracy": json.RawMessage(`1`)},
		}}},
	})
	require.NoError(t, err)
	require.True(t, tp.called)
}

func TestAddRunMetadataRejectsInvalidScoreMetadata(t *testing.T) {
	ctx := t.Context()
	auth, err := apiv1auth.NilAuthFinder(ctx)
	require.NoError(t, err)

	runID := ulid.Make()
	r := router{API: &API{opts: Opts{
		TraceReader: metadataFallbackTraceReader{span: &cqrs.OtelSpan{
			RawOtelSpan: cqrs.RawOtelSpan{
				TraceID: "00000000000000000000000000000001",
				SpanID:  "0000000000000001",
			},
			RunID:      runID,
			FunctionID: uuid.New(),
			AppID:      uuid.New(),
		}},
		State: metadataStateLoader{loadMetadata: func(context.Context, statev2.ID) (statev2.Metadata, error) {
			require.Fail(t, "LoadMetadata should not be called for invalid metadata")
			return statev2.Metadata{}, nil
		}},
	}}}

	err = r.AddRunMetadata(ctx, auth, runID, &AddRunMetadataRequest{
		Metadata: []metadata.Update{{RawUpdate: metadata.RawUpdate{
			Kind:   metadata.KindInngestScore,
			Op:     enums.MetadataOpcodeMerge,
			Values: metadata.Values{"score": json.RawMessage(`{"value":1}`)},
		}}},
	})
	require.ErrorIs(t, err, metadata.ErrScoreValueInvalid)
}

func TestAddRunMetadataAllowsRunScopedScoreMetadata(t *testing.T) {
	ctx := t.Context()
	auth, err := apiv1auth.NilAuthFinder(ctx)
	require.NoError(t, err)

	runID := ulid.Make()
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

	tp := &metadataFallbackTracerProvider{t: t, wantID: wantID}
	r := router{API: &API{opts: Opts{
		TraceReader: metadataFallbackTraceReader{span: &cqrs.OtelSpan{
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
			Kind:   metadata.KindInngestScore,
			Op:     enums.MetadataOpcodeMerge,
			Values: metadata.Values{"accuracy": json.RawMessage(`1`)},
		}}},
	})
	require.NoError(t, err)
	require.True(t, tp.called)
}

func TestAddRunMetadataRejectsDisallowedInngestKind(t *testing.T) {
	ctx := t.Context()
	auth, err := apiv1auth.NilAuthFinder(ctx)
	require.NoError(t, err)

	runID := ulid.Make()
	r := router{API: &API{opts: Opts{
		TraceReader: metadataFallbackTraceReader{span: &cqrs.OtelSpan{
			RawOtelSpan: cqrs.RawOtelSpan{
				TraceID: "00000000000000000000000000000001",
				SpanID:  "0000000000000001",
			},
			RunID:      runID,
			FunctionID: uuid.New(),
			AppID:      uuid.New(),
		}},
	}}}

	err = r.AddRunMetadata(ctx, auth, runID, &AddRunMetadataRequest{
		Metadata: []metadata.Update{{RawUpdate: metadata.RawUpdate{
			Kind:   "inngest.internal",
			Op:     enums.MetadataOpcodeMerge,
			Values: metadata.Values{"score": json.RawMessage(`1`)},
		}}},
	})
	require.ErrorIs(t, err, metadata.ErrKindNotAllowed)
}
