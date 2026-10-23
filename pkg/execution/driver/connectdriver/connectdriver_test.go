package connectdriver

import (
	"context"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	connectgrpc "github.com/inngest/inngest/pkg/connect/grpc"
	"github.com/inngest/inngest/pkg/execution/driver"
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

func TestProxyRequestUsesStableJobID(t *testing.T) {
	for _, tt := range []struct {
		name         string
		requestJobID string
		itemJobID    string
		wantJobID    string
	}{
		{name: "request job ID", requestJobID: "job-123", itemJobID: "item-job", wantJobID: "job-123"},
		{name: "queue item fallback", itemJobID: "item-job", wantJobID: "item-job"},
		{name: "different parallel job", requestJobID: "parallel-job", wantJobID: "parallel-job"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			u, err := url.Parse("https://example.com/inngest?fnId=test-fn")
			require.NoError(t, err)
			forwarder := &captureForwarder{}
			id := sv2.ID{RunID: ulid.Make(), FunctionID: uuid.New(), Tenant: sv2.Tenant{AccountID: uuid.New(), EnvID: uuid.New(), AppID: uuid.New()}}

			// Recovery changes dispatch time and generation, but must find the
			// original Connect request lease and buffered response.
			for generation := 1; generation <= 2; generation++ {
				ctx := queue.WithGenerationID(context.Background(), generation)
				dispatchID := driver.DispatchRequestID(time.Unix(int64(generation), 0), tt.wantJobID, generation).String()
				input := []byte(fmt.Sprintf(`{"ctx":{"request_id":%q,"generation_id":%d}}`, dispatchID, generation))
				_, err = ProxyRequest(ctx, context.Background(), forwarder, id,
					queue.Item{JobID: &tt.itemJobID}, httpdriver.Request{
						RequestID: dispatchID, JobID: tt.requestJobID, URL: *u,
						Step: inngest.Step{ID: "step"}, Edge: inngest.Edge{Incoming: "step"}, Input: input,
					})
				require.NoError(t, err)
				require.Equal(t, tt.wantJobID, forwarder.opts.Data.RequestId)
				require.Equal(t, tt.wantJobID, forwarder.opts.Data.JobId)
				// Preserve the dispatch identity and generation in the SDK payload.
				require.Equal(t, input, forwarder.opts.Data.RequestPayload)
			}
		})
	}
}

func TestProxyRequestFallsBackToDispatchIDWithoutJobID(t *testing.T) {
	requestID := ulid.Make().String()
	u, err := url.Parse("https://example.com/inngest?fnId=test-fn")
	require.NoError(t, err)
	forwarder := &captureForwarder{}
	_, err = ProxyRequest(context.Background(), context.Background(), forwarder, sv2.ID{}, queue.Item{},
		httpdriver.Request{RequestID: requestID, URL: *u, Input: []byte(`{"ctx":{}}`)})
	require.NoError(t, err)
	require.Equal(t, requestID, forwarder.opts.Data.RequestId)
}
