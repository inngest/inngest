package connectworker

import (
	"context"

	connectproto "github.com/inngest/inngest/proto/gen/connect/v1"
)

// RequestHandler serves the executor requests that arrive over the gateway
// connection.
type RequestHandler interface {
	// Admit decides if this worker takes the request.  the ACK tells the
	// gateway that this worker owns the request and must reply to it, so the
	// decision has to happen before the ACK and not inside Serve.  an error
	// drops the request with no ACK and no call to Serve or Done, which leaves
	// it free to run on another worker.  Admit runs on the connection's read
	// loop and must not block, because the same loop reads gateway heartbeats.
	Admit(req *connectproto.GatewayExecutorRequestData) error

	// Serve runs on a worker pool goroutine after the ACK.  the lease is
	// extended until Serve returns.  an error sends no reply, and the executor
	// retries the request after the lease expires.
	Serve(ctx context.Context, req *connectproto.GatewayExecutorRequestData) (Response, error)

	// Done runs once for each admitted request, after Connect writes or buffers the
	// reply, or after the request is dropped before the ACK and never reaches
	// Serve.
	//
	// this allows the handler to manage internal accounting for each ongoing request.
	//
	// note:  Done does not write or change the reply.  it runs on the worker pool
	// goroutine before that goroutine takes the next request, so it must not block.
	Done(req *connectproto.GatewayExecutorRequestData)
}

// Response is the reply to one executor request.  Connect copies the request
// IDs and trace contexts from the request into the SDKResponse.
type Response struct {
	Status connectproto.SDKResponseStatus
	Body   []byte

	// NoRetry and RetryAfter carry the X-Inngest-No-Retry and Retry-After
	// values that an HTTP serve endpoint sends as headers.
	NoRetry    bool
	RetryAfter *string

	// SDKVersion and RequestVersion carry the X-Inngest-SDK and
	// x-inngest-req-version values.  the executor uses them to pick retry
	// behavior and the execution protocol version.
	SDKVersion     string
	RequestVersion uint32
}

func newSDKResponse(req *connectproto.GatewayExecutorRequestData, resp Response) *connectproto.SDKResponse {
	return &connectproto.SDKResponse{
		RequestId:      req.RequestId,
		AccountId:      req.AccountId,
		EnvId:          req.EnvId,
		AppId:          req.AppId,
		Status:         resp.Status,
		Body:           resp.Body,
		NoRetry:        resp.NoRetry,
		RetryAfter:     resp.RetryAfter,
		SdkVersion:     resp.SDKVersion,
		RequestVersion: resp.RequestVersion,
		SystemTraceCtx: req.SystemTraceCtx,
		UserTraceCtx:   req.UserTraceCtx,
		RunId:          req.RunId,
	}
}
