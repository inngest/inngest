package apiv2

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"time"

	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/util"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/oklog/ulid/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// InvokeFunction invokes a function either synchronously or asynchronously.
func (s *Service) InvokeFunction(ctx context.Context, req *apiv2.InvokeFunctionRequest) (*apiv2.InvokeFunctionResponse, error) {
	if s.functions == nil || s.executor == nil || s.eventPublisher == nil {
		return nil, s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, "Unable to invoke functions")
	}

	if err := ValidateInvokeRequest(ctx, req); err != nil {
		return nil, err
	}

	f, err := s.functions.GetFunction(ctx, req.FunctionId)
	if err != nil {
		return nil, s.base.NewError(404, apiv2base.ErrorNotFound, "function not found")
	}

	var idempotencyHash string
	if req.IdempotencyKey != nil {
		idempotencyHash = util.XXHash(req.IdempotencyKey)
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
	md, err := s.executor.Schedule(ctx, sr)
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
				RunId:    md.ID.RunID.String(),
				Status:   apiv2.RunStatus_RUN_STATUS_QUEUED,
				QueuedAt: timestamppb.Now(),
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
		// TODO: Return the run ID here!!  Without this, people won't be able to find the
		// original run ID.
		return nil, s.base.NewError(
			http.StatusConflict,
			apiv2base.ErrorIdempotencyConflict,
			fmt.Sprintf("A function execution with this idempotency key already exists. Idempotency key: %s", *req.IdempotencyKey),
		)
	}

	logger.From(ctx).Error("error invoking function via api", "error", err)
	return nil, s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, "There was an error invoking your function")
}
