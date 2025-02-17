package connectdriver

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/connect/pubsub"
	"github.com/inngest/inngest/pkg/execution/driver/httpdriver"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	itrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/inngest/inngest/proto/gen/connect/v1"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"net/http"
	"net/url"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/syscode"
)

const (
	pkgName = "connect.execution.driver"
)

func NewDriver(ctx context.Context, psf pubsub.RequestForwarder) driver.Driver {
	return &executor{
		forwarder: psf,
	}
}

type executor struct {
	forwarder pubsub.RequestForwarder
}

// RuntimeType fulfills the inngest.Runtime interface.
func (e executor) RuntimeType() string {
	return "connect"
}

func (e executor) Execute(ctx context.Context, sl sv2.StateLoader, s sv2.Metadata, item queue.Item, edge inngest.Edge, step inngest.Step, idx, attempt int) (*state.DriverResponse, error) {
	traceCtx := context.Background()

	traceCtx, span := itrace.ConnectTracer().Start(traceCtx, "Execute")
	span.SetAttributes(attribute.Bool("inngest.system", true))
	defer span.End()

	span.SetAttributes(
		attribute.String("account_id", s.ID.Tenant.AccountID.String()),
		attribute.String("env_id", s.ID.Tenant.EnvID.String()),
		attribute.String("app_id", s.ID.Tenant.AppID.String()),
		attribute.String("run_id", s.ID.RunID.String()),
		attribute.String("function_id", s.ID.FunctionID.String()),
	)

	start := time.Now()
	defer func() {
		metrics.HistogramConnectExecutorEndToEndDuration(ctx, time.Since(start).Milliseconds(), metrics.HistogramOpt{
			PkgName: pkgName,
		})
	}()

	input, err := driver.MarshalV1(ctx, sl, s, step, idx, "", attempt)
	if err != nil {
		return nil, err
	}

	uri, err := url.Parse(step.URI)
	if err != nil {
		return nil, err
	}

	if e.forwarder == nil {
		return nil, fmt.Errorf("missing connect request forwarder")
	}

	return ProxyRequest(ctx, traceCtx, e.forwarder, s.ID.Tenant, httpdriver.Request{
		WorkflowID: s.ID.FunctionID,
		RunID:      s.ID.RunID,
		URL:        *uri,
		Input:      input,
		Edge:       edge,
		Step:       step,
	})
}

// ProxyRequest proxies the request to the SDK over a long-lived connection with the given input.
func ProxyRequest(ctx, traceCtx context.Context, forwarder pubsub.RequestForwarder, tenant sv2.Tenant, r httpdriver.Request) (*state.DriverResponse, error) {
	requestToForward := connect.GatewayExecutorRequestData{
		// TODO Find out if we can supply this in a better way. We still use the URL concept a lot,
		// even though this has no meaning in connect.
		FunctionSlug:   r.URL.Query().Get("fnId"),
		RequestPayload: r.Input,
		AppId:          tenant.AppID.String(),
		EnvId:          tenant.EnvID.String(),
	}
	// If we have a generator step name, ensure we add the step ID parameter
	if r.Edge.IncomingGeneratorStep != "" {
		requestToForward.StepId = &r.Edge.IncomingGeneratorStep
	} else {
		requestToForward.StepId = &r.Edge.Incoming
	}

	span := trace.SpanFromContext(traceCtx)
	span.SetAttributes(
		attribute.String("step_id", requestToForward.GetStepId()),
	)

	resp, err := do(ctx, traceCtx, forwarder, tenant.AppID, &requestToForward)
	if err != nil {
		return nil, err
	}

	return httpdriver.HandleHttpResponse(ctx, r, resp)
}

func do(ctx, traceCtx context.Context, forwarder pubsub.RequestForwarder, appId uuid.UUID, data *connect.GatewayExecutorRequestData) (*httpdriver.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, consts.MaxFunctionTimeout)
	defer cancel()

	pre := time.Now()
	resp, err := forwarder.Proxy(ctx, traceCtx, appId, data)
	dur := time.Since(pre)

	span := trace.SpanFromContext(traceCtx)
	if err != nil {
		span.RecordError(err)
	}

	// TODO Check if we need some of the request error handling logic from httpdriver.do()
	if err != nil && resp == nil {
		return nil, err
	}

	// Return gateway-handled errors like  syscode.CodeOutputTooLarge
	var sysErr *syscode.Error
	{
		syscodeError := &syscode.Error{}
		if errors.As(err, &syscodeError) {
			sysErr = syscodeError
		}
	}

	// TODO Should be handled above, verify this
	//// Read 1 extra byte above the max so that we can check if the response is
	//// too large
	//byt, err := io.ReadAll(io.LimitReader(resp.Body, consts.MaxBodySize+1))
	//if err != nil {
	//	return nil, fmt.Errorf("error reading response body: %w", err)
	//}
	//var sysErr *syscode.Error
	//if len(byt) > consts.MaxBodySize {
	//	sysErr = &syscode.Error{Code: syscode.CodeOutputTooLarge}
	//
	//	// Override the output so the user sees the syserrV in the UI rather
	//	// than a JSON parsing error
	//	byt, _ = json.Marshal(sysErr.Code)
	//}

	noRetryStr := ""
	if resp.NoRetry {
		noRetryStr = "true"
	}

	// Check the retry status from the headers and versions.
	noRetry := !httpdriver.ShouldRetry(int(resp.Status), noRetryStr, resp.SdkVersion)

	// Extract the retry at header if it hasn't been set explicitly in streaming.
	var retryAtStr *string
	if after := resp.RetryAfter; retryAtStr == nil && after != nil {
		retryAtStr = after
	}
	var retryAt *time.Time
	if retryAtStr != nil {
		if at, err := httpdriver.ParseRetry(*retryAtStr); err == nil {
			retryAt = &at
			span.SetAttributes(attribute.String("retry_at", at.String()))
		}
	}

	// TODO connect is only implemented by SDKs, but we can include a flag in the proxied resp as well...
	isSDK := true

	statusCode := 0
	switch resp.Status {
	case connect.SDKResponseStatus_DONE:
		statusCode = http.StatusOK
	case connect.SDKResponseStatus_ERROR:
		statusCode = http.StatusInternalServerError
	case connect.SDKResponseStatus_NOT_COMPLETED:
		statusCode = http.StatusPartialContent
	}

	span.SetAttributes(
		attribute.Int("status_code", statusCode),
	)

	return &httpdriver.Response{
		Body:           resp.Body,
		StatusCode:     statusCode,
		Duration:       dur,
		RetryAt:        retryAt,
		NoRetry:        noRetry,
		RequestVersion: int(resp.RequestVersion),
		IsSDK:          isSDK,
		Sdk:            resp.SdkVersion,
		Header:         http.Header{}, // not supported by connect
		SysErr:         sysErr,
	}, err

}
