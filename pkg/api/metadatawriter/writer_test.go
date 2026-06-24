package metadatawriter

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestWriterPersistsMetadataSizeDeltaForLiveState(t *testing.T) {
	runID := ulid.Make()
	state := &writerState{
		md: statev2.Metadata{
			ID: statev2.ID{
				RunID:      runID,
				FunctionID: uuid.New(),
				Tenant: statev2.Tenant{
					AccountID: uuid.New(),
					EnvID:     uuid.New(),
					AppID:     uuid.New(),
				},
			},
		},
	}

	err := Writer{
		State:          state,
		TracerProvider: writerTracer{},
		PkgName:        "metadatawriter.test",
	}.Write(context.Background(), WriteRequest{
		ID: state.md.ID,
		Items: NewItems([]metadata.Update{testMetadataUpdate(t)}, func(md *statev2.Metadata) (*meta.SpanReference, metadata.Scope) {
			return &meta.SpanReference{DynamicSpanID: "parent"}, enums.MetadataScopeRun
		}),
		Location: "TestWriter",
	})

	require.NoError(t, err)
	require.Positive(t, state.incremented)
}

func TestWriterUsesFallbackWithoutPersistingMetadataSizeDelta(t *testing.T) {
	runID := ulid.Make()
	accountID := uuid.New()
	envID := uuid.New()
	loaderCalled := false
	state := &writerState{err: statev2.ErrMetadataNotFound}

	err := Writer{
		State:          state,
		TracerProvider: writerTracer{},
		MissingStateLoader: func(ctx context.Context, id statev2.ID) (*statev2.Metadata, error) {
			loaderCalled = true
			require.Equal(t, runID, id.RunID)
			return &statev2.Metadata{
				ID: statev2.ID{
					RunID:      id.RunID,
					FunctionID: uuid.New(),
					Tenant: statev2.Tenant{
						AccountID: id.Tenant.AccountID,
						EnvID:     id.Tenant.EnvID,
						AppID:     uuid.New(),
					},
				},
			}, nil
		},
		PkgName: "metadatawriter.test",
	}.Write(context.Background(), WriteRequest{
		ID: statev2.ID{
			RunID: runID,
			Tenant: statev2.Tenant{
				AccountID: accountID,
				EnvID:     envID,
			},
		},
		Items: NewItems([]metadata.Update{testMetadataUpdate(t)}, func(md *statev2.Metadata) (*meta.SpanReference, metadata.Scope) {
			return &meta.SpanReference{DynamicSpanID: "parent"}, enums.MetadataScopeRun
		}),
		Location: "TestWriter",
	})

	require.NoError(t, err)
	require.True(t, loaderCalled)
	require.Zero(t, state.incremented)
}

func testMetadataUpdate(t *testing.T) metadata.Update {
	t.Helper()

	return metadata.Update{
		RawUpdate: metadata.RawUpdate{
			Kind:   metadata.KindInngestScore,
			Op:     enums.MetadataOpcodeMerge,
			Values: metadata.Values{"accuracy": json.RawMessage(`{"value":1}`)},
		},
	}
}

type writerState struct {
	statev2.RunService
	md          statev2.Metadata
	err         error
	incremented int
}

func (s *writerState) LoadMetadata(ctx context.Context, id statev2.ID, _ ...statev2.LoadMetadataOption) (statev2.Metadata, error) {
	return s.md, s.err
}

func (s *writerState) IncrementMetadataSize(ctx context.Context, id statev2.ID, delta int) error {
	s.incremented += delta
	return nil
}

type writerTracer struct{}

func (writerTracer) CreateSpan(ctx context.Context, name string, opts *tracing.CreateSpanOptions) (*meta.SpanReference, error) {
	return &meta.SpanReference{DynamicSpanID: opts.DynamicSpanIDOverride}, nil
}

func (writerTracer) CreateDroppableSpan(context.Context, string, *tracing.CreateSpanOptions) (*tracing.DroppableSpan, error) {
	return nil, nil
}

func (writerTracer) UpdateSpan(context.Context, *tracing.UpdateSpanOptions) error {
	return nil
}
