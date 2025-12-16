package connectdriver

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/inngest/inngest/pkg/connect/grpc"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/driver/httpdriver"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/syscode"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	itrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	pkgName = "connect.execution.driver"
)

func NewDriver(ctx context.Context, psf grpc.RequestForwarder, tracer itrace.ConditionalTracer) driver.DriverV1 {
	return &executor{
		forwarder: psf,
		tracer:    tracer,
	}
}

type executor struct {
	forwarder grpc.RequestForwarder
	tracer    itrace.ConditionalTracer
}

func (e executor) Name() string {
	return "connect"
}

func (e executor) Execute(ctx context.Context, sl sv2.StateLoader, s sv2.Metadata, item queue.Item, edge inngest.Edge, step inngest.Step, idx, attempt int) (*state.DriverResponse, error) {
	if e.forwarder == nil {
		return nil, fmt.Errorf("missing connect request forwarder")
	}

	if e.tracer == nil {
		return nil, fmt.Errorf("missing connect tracer")
	}

	traceCtx := context.Background()

	traceCtx, span := e.tracer.NewSpan(traceCtx, "Execute", s.ID.Tenant.AccountID, s.ID.Tenant.EnvID)
	defer span.End()

	span.SetAttributes(
		// Ensure OTel Collector ships this to Honeycomb
		attribute.Bool("inngest.system", true),
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
			Tags: map[string]any{
				"account_id": s.ID.Tenant.AccountID.String(),
			},
		})
	}()

	jID := queue.JobIDFromContext(ctx)

	input, err := driver.MarshalV1(ctx, sl, s, step, idx, "", attempt, item.GetMaxAttempts(), jID)
	if err != nil {
		return nil, err
	}

	uri, err := url.Parse(step.URI)
	if err != nil {
		return nil, err
	}

	return ProxyRequest(ctx, traceCtx, e.forwarder, s.ID, item, httpdriver.Request{
		WorkflowID: s.ID.FunctionID,
		RunID:      s.ID.RunID,
		URL:        *uri,
		Input:      input,
		Edge:       edge,
		Step:       step,
	})
}

// ProxyRequest proxies the request to the SDK over a long-lived connection with the given input.
func ProxyRequest(ctx, traceCtx context.Context, forwarder grpc.RequestForwarder, id sv2.ID, item queue.Item, r httpdriver.Request) (*state.DriverResponse, error) {
	l := logger.StdlibLogger(ctx)

	var requestID string
	if item.JobID != nil {
		// Use the stable queue item ID
		requestID = *item.JobID
	} else {
		// This should never happen, handle it gracefully
		l.Warn("queue item missing jobID", "item", item, "id", id)
		requestID = ulid.MustNew(ulid.Now(), rand.Reader).String()
	}

	requestToForward := connect.GatewayExecutorRequestData{
		// TODO Find out if we can supply this in a better way. We still use the URL concept a lot,
		// even though this has no meaning in connect.
		FunctionSlug:   r.URL.Query().Get("fnId"),
		FunctionId:     id.FunctionID.String(),
		RequestPayload: r.Input,
		AppId:          id.Tenant.AppID.String(),
		EnvId:          id.Tenant.EnvID.String(),
		AccountId:      id.Tenant.AccountID.String(),
		RunId:          id.RunID.String(),
		RequestId:      requestID,
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

	opts := grpc.ProxyOpts{
		AccountID: id.Tenant.AccountID,
		EnvID:     id.Tenant.EnvID,
		AppID:     id.Tenant.AppID,
		Data:      &requestToForward,
	}

	if spanID, err := item.SpanID(); err != nil {
		l.Error("error retrieving span ID",
			"error", err,
			"run_id", id.RunID.String(),
		)
	} else {
		opts.SpanID = spanID.String()
	}

	resp, err := do(ctx, traceCtx, forwarder, opts)
	if err != nil {
		return nil, err
	}

	return httpdriver.HandleHttpResponse(ctx, r, resp)
}

func do(ctx, traceCtx context.Context, forwarder grpc.RequestForwarder, opts grpc.ProxyOpts) (*httpdriver.Response, error) {
	span := trace.SpanFromContext(traceCtx)

	pre := time.Now()
	resp, err := forwarder.Proxy(ctx, traceCtx, opts)
	dur := time.Since(pre)

	var sysErr *syscode.Error

	if err != nil {
		span.RecordError(err)

		syscodeError := &syscode.Error{}
		if errors.As(err, &syscodeError) || errors.As(err, syscodeError) {
			sysErr = syscodeError
		}
	}

	if resp == nil && err != nil {
		return nil, err
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
