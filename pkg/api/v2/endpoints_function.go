package apiv2

import (
	"context"
	"crypto/rand"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/util"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/oklog/ulid/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// InvokeFunction invokes a function either synchronously or asynchronously.
func (s *Service) InvokeFunction(ctx context.Context, req *apiv2.InvokeFunctionRequest) (*apiv2.InvokeFunctionResponse, error) {
	if err := validateInvokeRequest(ctx, req); err != nil {
		return nil, err
	}

	if result := s.rateLimiter.CheckRateLimit(ctx, apiv2.V2_InvokeFunction_FullMethodName); result.Limited {
		return nil, s.base.NewError(http.StatusTooManyRequests, apiv2base.ErrorRateLimited,
			"API rate limit exceeded. The request was rejected and no function was invoked.")
	}

	if s.functions == nil || s.executor == nil || s.eventPublisher == nil {
		return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Invoke is not yet implemented")
	}

	// Build the composite function slug from app_id and function_id.
	// If the function_id already has the app_id prefix, use it as-is.
	functionSlug := req.FunctionId
	if len(req.AppId) > 0 && !strings.HasPrefix(req.FunctionId, req.AppId+"-") {
		functionSlug = req.AppId + "-" + req.FunctionId
	}

	f, err := s.functions.GetFunction(ctx, functionSlug)
	if err != nil {
		return nil, s.base.NewError(404, apiv2base.ErrorNotFound, "function not found")
	}

	var idempotencyHash string
	if req.IdempotencyKey != nil {
		idempotencyHash = util.XXHash(*req.IdempotencyKey)
	}

	// Create an invoke event using the data from the above request.
	// Pass this into an event publisher, and use this event within the schedule request.
	eventID := ulid.MustNew(ulid.Now(), rand.Reader)
	data := req.GetData().AsMap()
	data[consts.InngestEventDataPrefix] = event.InngestMetadata{
		InvokeType:           "api",
		InvokeFnID:           f.ID.String(),
		InvokeIdempotencyKey: idempotencyHash,
	}
	event := event.BaseTrackedEvent{
		ID:          eventID,
		AccountID:   f.AccountID,
		WorkspaceID: f.EnvironmentID,
		Event: event.Event{
			ID:        eventID.String(),
			Name:      consts.FnInvokeName,
			Data:      data,
			Timestamp: time.Now().UnixMilli(),
		},
	}
	if err := s.eventPublisher.Publish(context.WithoutCancel(ctx), event); err != nil {
		return nil, s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, "Unable to publish invoke event")
	}

	// Schedule the function directly, instead of waiting for pubsub.  This improves latency
	// in the fast path, and is necessary for us to return the run ID.
	sr := execution.NewScheduleRequest(f)
	sr.IdempotencyKey = &idempotencyHash
	sr.Events = append(sr.Events, event)
	runID, _, err := s.executor.Schedule(ctx, sr)
	scheduleStatus := executor.ScheduleStatus(err)

	go func() {
		// Record metrics for function invocation
		metrics.RecordCounterMetric(ctx, 1, metrics.CounterOpt{
			PkgName:     "apiv2",
			MetricName:  "function_invoke_total",
			Description: "Total number of function invocation attempts via API",
			Tags: map[string]any{
				"status": scheduleStatus,
			},
		})
	}()

	switch scheduleStatus {
	case "success":
		// XXX: We should eventually allow sync invokes.  If this is a sync invoke, we want
		// to await the run and ensure that it's finished.  We only do this for up to N seconds,
		// then respond with a status code and the run information.
		//
		// In order to enable this, we very much need to consider rate limits and requests
		// to the backing data store (eg. Clickhouse).  Instead of polling this, we probably
		// want a pub/sub system with a polling check every N secondss (ie. 30) so that we
		// don't hammer the DB with hundreds of thousands of these sync invokes concurrently
		// running.
		return &apiv2.InvokeFunctionResponse{
			Data: &apiv2.InvokeFunctionData{
				RunId: runID.String(),
			},
			Metadata: &apiv2.ResponseMetadata{
				FetchedAt: timestamppb.Now(),
			},
		}, nil
	case "rate_limited":
		return nil, s.base.NewError(
			http.StatusUnprocessableEntity,
			apiv2base.ErrorRateLimited,
			"Function execution rate limit exceeded. Please try again later.",
		)
	case "debounced":
		return nil, s.base.NewError(
			http.StatusUnprocessableEntity,
			apiv2base.ErrorFunctionDebounced,
			"Function invocation was debounced.",
		)
	case "skipped":
		return nil, s.base.NewError(
			http.StatusUnprocessableEntity,
			apiv2base.ErrorFunctionSkipped,
			"Function invocation was skipped because the function is paused or draining.",
		)
	case "idempotency":
		_ = grpc.SetHeader(ctx, metadata.Pairs("x-http-code", "409"))
		return &apiv2.InvokeFunctionResponse{
			Data: &apiv2.InvokeFunctionData{
				RunId: runID.String(),
			},
			Metadata: &apiv2.ResponseMetadata{
				FetchedAt: timestamppb.Now(),
			},
		}, nil
	}

	// Check for idempotency errors that ScheduleStatus didn't classify.
	// This can happen when sentinel errors lose their identity crossing
	// gRPC boundaries (e.g. the cloud state proxy).
	if isIdempotencyError(err) {
		_ = grpc.SetHeader(ctx, metadata.Pairs("x-http-code", "409"))
		return &apiv2.InvokeFunctionResponse{
			Data: &apiv2.InvokeFunctionData{
				RunId: runID.String(),
			},
			Metadata: &apiv2.ResponseMetadata{
				FetchedAt: timestamppb.Now(),
			},
		}, nil
	}

	logger.From(ctx).Error("error invoking function via api", "error", err)
	return nil, s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, "There was an error invoking your function")
}

func (s *Service) InvokeFunctionBySlug(ctx context.Context, req *apiv2.InvokeFunctionBySlugRequest) (*apiv2.InvokeFunctionResponse, error) {
	if req.FunctionSlug == "" {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorMissingField, "Function slug is required")
	}

	return s.InvokeFunction(ctx, &apiv2.InvokeFunctionRequest{
		FunctionId:     req.FunctionSlug,
		Data:           req.Data,
		IdempotencyKey: req.IdempotencyKey,
	})
}

// isIdempotencyError checks whether the given error represents an idempotency
// conflict. It first checks via errors.Is for the standard sentinel errors,
// then falls back to string matching for cases where the sentinel identity is
// lost crossing gRPC boundaries (e.g. the cloud state proxy).
func isIdempotencyError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, queue.ErrQueueItemExists) ||
		errors.Is(err, executor.ErrFunctionSkippedIdempotency) ||
		errors.Is(err, state.ErrIdentifierExists) {
		return true
	}
	// Fallback: check the error string for known idempotency error messages
	// that may have been serialized across gRPC boundaries.
	msg := err.Error()
	return strings.Contains(msg, state.ErrIdentifierExists.Error()) ||
		strings.Contains(msg, queue.ErrQueueItemExists.Error()) ||
		strings.Contains(msg, executor.ErrFunctionSkippedIdempotency.Error())
}
