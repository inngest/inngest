package checkpoint

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestProcessMetadataForwardsStepType(t *testing.T) {
	ctx := context.Background()
	l := logger.StdlibLogger(ctx)
	rec := &recordingMetadataListener{}

	c := checkpointer{
		Opts: Opts{
			TracerProvider:         tracing.NewNoopTracerProvider(),
			SyncLifecycleListeners: []execution.SyncLifecycleListener{rec},
		},
	}

	accountID := uuid.New()
	envID := uuid.New()
	appID := uuid.New()
	fnID := uuid.New()
	runID := ulid.MustNew(ulid.Now(), nil)

	md := &sv2.Metadata{
		ID: sv2.ID{
			RunID:      runID,
			FunctionID: fnID,
			Tenant: sv2.Tenant{
				AccountID: accountID,
				EnvID:     envID,
				AppID:     appID,
			},
		},
		Config: *sv2.InitConfig(&sv2.Config{}),
	}
	md.Config.SetAppName("my-app")
	md.Config.SetFunctionSlug("my-app-fn")

	op := state.GeneratorOpcode{
		Op:   enums.OpcodeStepRun,
		ID:   "hashed-step-id",
		Name: "my-step",
		Metadata: []metadata.ScopedUpdate{
			{
				Update: metadata.Update{
					RawUpdate: metadata.RawUpdate{
						Kind:   metadata.Kind("userland.custom"),
						Op:     enums.MetadataOpcodeSet,
						Values: metadata.Values{"k": json.RawMessage(`"v"`)},
					},
				},
				Scope: enums.MetadataScopeStep,
			},
		},
	}

	c.processMetadata(ctx, l, accountID, md, nil, op, "checkpoint.test")

	require.Len(t, rec.entries, 1)
	entry := rec.entries[0]
	require.Equal(t, accountID, entry.AccountID)
	require.Equal(t, envID, entry.EnvID)
	require.Equal(t, appID, entry.AppID)
	require.Equal(t, fnID, entry.FunctionID)
	require.Equal(t, "my-app", entry.AppName)
	require.Equal(t, "my-app-fn", entry.FunctionSlug)
	require.Equal(t, "hashed-step-id", entry.StepHashedID)
	require.Equal(t, enums.StepTypeRun.String(), entry.StepType)
}
