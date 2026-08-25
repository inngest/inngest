package connectdriver

import (
	"context"
	"net/url"
	"testing"

	"github.com/google/uuid"
	connectgrpc "github.com/inngest/inngest/pkg/connect/grpc"
	"github.com/inngest/inngest/pkg/execution/driver/httpdriver"
	"github.com/inngest/inngest/pkg/execution/queue"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

type captureForwarder struct {
	opts connectgrpc.ProxyOpts
}

func (c *captureForwarder) Proxy(ctx, traceCtx context.Context, opts connectgrpc.ProxyOpts) (*connectpb.SDKResponse, error) {
	c.opts = opts
	return &connectpb.SDKResponse{
		RequestId:      opts.Data.RequestId,
		Status:         connectpb.SDKResponseStatus_DONE,
		Body:           []byte(`{"ok":true}`),
		SdkVersion:     "test-sdk",
		RequestVersion: 1,
	}, nil
}

func TestProxyRequestUsesRequestIDAndJobID(t *testing.T) {
	requestID := "01ARZ3NDEKTSV4RRFFQ69G5FAV"
	jobID := "job-123"
	u, err := url.Parse("https://example.com/inngest?fnId=test-fn")
	require.NoError(t, err)

	forwarder := &captureForwarder{}
	_, err = ProxyRequest(
		context.Background(),
		context.Background(),
		forwarder,
		sv2.ID{
			RunID:      ulid.Make(),
			FunctionID: uuid.New(),
			Tenant: sv2.Tenant{
				AccountID: uuid.New(),
				EnvID:     uuid.New(),
				AppID:     uuid.New(),
			},
		},
		queue.Item{JobID: &jobID},
		httpdriver.Request{
			RequestID: requestID,
			JobID:     jobID,
			URL:       *u,
			Step:      inngest.Step{ID: "step"},
			Edge:      inngest.Edge{Incoming: "step"},
			Input:     []byte(`{"ctx":{}}`),
		},
	)
	require.NoError(t, err)
	require.NotNil(t, forwarder.opts.Data)
	require.Equal(t, requestID, forwarder.opts.Data.RequestId)
	require.Equal(t, jobID, forwarder.opts.Data.JobId)
}
