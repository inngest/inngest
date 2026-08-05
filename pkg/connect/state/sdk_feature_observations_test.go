package state

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	cqsync "github.com/inngest/inngest/pkg/cqrs/sync"
	"github.com/inngest/inngest/pkg/sdk"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	sdkfeatureobs "github.com/inngest/inngest/proto/gen/sdk_feature_observations/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestWorkerGroupSyncIncludesFeatureObservations(t *testing.T) {
	ctx := context.Background()
	appID := uuid.New()
	syncID := uuid.New()
	existingAppID := uuid.New()
	existingSyncID := uuid.New()

	var registerReq sdk.RegisterRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/fn/register", r.URL.Path)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&registerReq))

		require.NoError(t, json.NewEncoder(w).Encode(cqsync.Reply{
			OK:     true,
			AppID:  &appID,
			SyncID: &syncID,
		}))
	}))
	defer server.Close()

	group := &WorkerGroup{
		AccountID:  uuid.New(),
		EnvID:      uuid.New(),
		AppName:    "connect-app",
		SDKLang:    "js",
		SDKVersion: "v4.13.0",
		Hash:       "worker-group-hash",
		SyncData: SyncData{
			Functions: []sdk.SDKFunction{
				{
					Name: "Test function",
					Slug: "test-function",
				},
			},
			FeatureObservations: sdk.FeatureObservationsFromProto(testFeatureObservations()),
		},
	}
	groupManager := &testWorkerGroupManager{
		existing: &WorkerGroup{
			AppID:  &existingAppID,
			SyncID: &existingSyncID,
		},
	}

	err := group.Sync(ctx, groupManager, server.URL, &connectpb.WorkerConnectRequestData{
		AuthData:     &connectpb.AuthData{SyncToken: "sync-token"},
		Capabilities: []byte(`{"connect":"v1"}`),
		SdkLanguage:  "js",
		SdkVersion:   "v4.13.0",
	}, true)
	require.NoError(t, err)

	require.True(t, proto.Equal(group.SyncData.FeatureObservations.Proto(), registerReq.FeatureObservations.Proto()))
	require.Equal(t, sdk.DeployTypeConnect, registerReq.DeployType)
	require.Equal(t, "connect-app", registerReq.AppName)
	require.Equal(t, "worker-group-hash", registerReq.IdempotencyKey)
	require.Equal(t, appID, *group.AppID)
	require.Equal(t, syncID, *group.SyncID)
	require.Equal(t, group, groupManager.updated)
}

func TestWorkerGroupSyncSkipsWhenFeatureObservationsAreUnchanged(t *testing.T) {
	ctx := context.Background()
	existingAppID := uuid.New()
	existingSyncID := uuid.New()
	observations := sdk.FeatureObservationsFromProto(testFeatureObservations())
	observationsHash, err := sdkFeatureObservationsHash(observations)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("expected unchanged feature observations to skip sync")
	}))
	defer server.Close()

	group := &WorkerGroup{
		AccountID:  uuid.New(),
		EnvID:      uuid.New(),
		AppName:    "connect-app",
		SDKLang:    "js",
		SDKVersion: "v4.13.0",
		Hash:       "worker-group-hash",
		SyncData: SyncData{
			Functions: []sdk.SDKFunction{
				{
					Name: "Test function",
					Slug: "test-function",
				},
			},
			FeatureObservations: observations,
		},
	}
	groupManager := &testWorkerGroupManager{
		existing: &WorkerGroup{
			AppID:                   &existingAppID,
			SyncID:                  &existingSyncID,
			FeatureObservationsHash: observationsHash,
		},
	}

	err = group.Sync(ctx, groupManager, server.URL, &connectpb.WorkerConnectRequestData{
		AuthData:     &connectpb.AuthData{SyncToken: "sync-token"},
		Capabilities: []byte(`{"connect":"v1"}`),
		SdkLanguage:  "js",
		SdkVersion:   "v4.13.0",
	}, true)
	require.NoError(t, err)
	require.Equal(t, existingAppID, *group.AppID)
	require.Equal(t, existingSyncID, *group.SyncID)
	require.Nil(t, groupManager.updated)
}

func testFeatureObservations() *sdkfeatureobs.FeatureObservations {
	return &sdkfeatureobs.FeatureObservations{
		AiMetadataExtraction: &sdkfeatureobs.AIMetadataExtraction{
			ReadinessReason: sdkfeatureobs.AIMetadataExtractionReadinessReason_AI_METADATA_EXTRACTION_READINESS_REASON_OTEL_PROVIDER_MISSING,
			OtelSetup: &sdkfeatureobs.OTelSetup{
				ProviderFound:             true,
				ProviderSource:            sdkfeatureobs.OTelProviderSource_OTEL_PROVIDER_SOURCE_FIRST_PARTY,
				AddSpanProcessorAttempted: true,
				SpanProcessorAdded:        false,
				Failure:                   sdkfeatureobs.OTelSetupFailure_OTEL_SETUP_FAILURE_NO_PROVIDER,
			},
		},
		ExtendedTraces: &sdkfeatureobs.ExtendedTraces{
			ReadinessReason: sdkfeatureobs.ExtendedTracesReadinessReason_EXTENDED_TRACES_READINESS_REASON_NOT_ENABLED_BY_USER,
			Config: &sdkfeatureobs.ExtendedTracesConfig{
				Behavior: sdkfeatureobs.ExtendedTracesBehavior_EXTENDED_TRACES_BEHAVIOR_AUTO,
			},
			OtelSetup: &sdkfeatureobs.OTelSetup{
				Path:                      sdkfeatureobs.OTelSetupPath_OTEL_SETUP_PATH_EXTEND_EXISTING_PROVIDER,
				ProviderSource:            sdkfeatureobs.OTelProviderSource_OTEL_PROVIDER_SOURCE_USER_PROVIDED,
				AddSpanProcessorAttempted: true,
				SpanProcessorAdded:        true,
			},
		},
		SendEvents: &sdkfeatureobs.SendEvents{
			ReadinessReason: sdkfeatureobs.SendEventsReadinessReason_SEND_EVENTS_READINESS_REASON_READY,
			Config: &sdkfeatureobs.SendEventsConfig{
				EventKeyConfigured:               true,
				EventApiOriginOverrideConfigured: true,
			},
		},
	}
}

type testWorkerGroupManager struct {
	existing *WorkerGroup
	updated  *WorkerGroup
}

func (m *testWorkerGroupManager) GetWorkerGroupByHash(_ context.Context, _ uuid.UUID, _ string) (*WorkerGroup, error) {
	return m.existing, nil
}

func (m *testWorkerGroupManager) UpdateWorkerGroup(_ context.Context, _ uuid.UUID, group *WorkerGroup) error {
	m.updated = group
	return nil
}
