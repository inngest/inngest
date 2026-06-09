package apiv2

import (
	"context"
	"crypto/rand"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/util"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/oklog/ulid/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Service) GetFunction(ctx context.Context, req *apiv2.GetFunctionRequest) (*apiv2.GetFunctionResponse, error) {
	if req.AppId == "" || req.FunctionId == "" {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorMissingField, "App ID and function ID are required")
	}

	if result := s.rateLimiter.CheckRateLimit(ctx, apiv2.V2_GetFunction_FullMethodName); result.Limited {
		return nil, s.base.NewError(http.StatusTooManyRequests, apiv2base.ErrorRateLimited,
			"API rate limit exceeded. The request was rejected and no function was fetched.")
	}

	if s.functions == nil {
		return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Get function is not yet implemented")
	}

	appID := decodePathParam(req.AppId)
	functionID := decodePathParam(req.FunctionId)

	fn, err := s.functions.GetFunctionByApp(ctx, appID, functionID)
	if err != nil {
		return nil, s.getFunctionError(err)
	}
	fn.AppName = appID
	fn.Function.Slug = publicFunctionID(appID, functionID, fn)

	return s.getFunctionResponse(ctx, fn), nil
}

func (s *Service) getFunctionError(err error) error {
	if errors.Is(err, ErrFunctionNotFound) {
		return s.base.NewError(http.StatusNotFound, apiv2base.ErrorNotFound, "Function not found")
	}
	return s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, "Unable to fetch function")
}

func (s *Service) getFunctionResponse(ctx context.Context, fn inngest.DeployedFunction) *apiv2.GetFunctionResponse {
	return &apiv2.GetFunctionResponse{
		Data:     toFunction(fn, s.planConcurrencyLimit(ctx, fn)),
		Metadata: &apiv2.ResponseMetadata{FetchedAt: timestamppb.Now()},
	}
}

// InvokeFunction invokes a function either synchronously or asynchronously.
func (s *Service) InvokeFunction(ctx context.Context, req *apiv2.InvokeFunctionRequest) (*apiv2.InvokeFunctionResponse, error) {
	if err := validateInvokeRequest(ctx, req); err != nil {
		return nil, err
	}

	// URI-decode path parameters to handle encoded characters (e.g. %2F for slashes)
	req.FunctionId = decodePathParam(req.FunctionId)
	req.AppId = decodePathParam(req.AppId)

	if result := s.rateLimiter.CheckRateLimit(ctx, apiv2.V2_InvokeFunction_FullMethodName); result.Limited {
		return nil, s.base.NewError(http.StatusTooManyRequests, apiv2base.ErrorRateLimited,
			"API rate limit exceeded. The request was rejected and no function was invoked.")
	}

	if s.functions == nil || s.executor == nil || s.eventPublisher == nil {
		return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Invoke is not yet implemented")
	}

	f, err := s.functions.GetFunctionByApp(ctx, req.AppId, req.FunctionId)
	if err != nil {
		return nil, s.getFunctionError(err)
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

func decodePathParam(value string) string {
	if decoded, err := url.PathUnescape(value); err == nil {
		return decoded
	}
	return value
}

func publicFunctionID(appID string, requestedFunctionID string, fn inngest.DeployedFunction) string {
	if fn.Function.Slug != "" && fn.Function.Slug != fn.Slug {
		return fn.Function.Slug
	}
	if requestedFunctionID != "" && requestedFunctionID != fn.Slug {
		return requestedFunctionID
	}

	functionID := fn.Function.Slug
	if functionID == "" {
		functionID = fn.Slug
	}
	if trimmed := strings.TrimPrefix(functionID, appID+"-"); trimmed != functionID {
		return trimmed
	}
	return functionID
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

func (s *Service) planConcurrencyLimit(ctx context.Context, fn inngest.DeployedFunction) int {
	if s.functionConfig == nil {
		return models.UnknownPlanConcurrencyLimit
	}
	return s.functionConfig.PlanConcurrencyLimit(ctx, fn)
}

func toFunction(fn inngest.DeployedFunction, planConcurrencyLimit int) *apiv2.Function {
	return &apiv2.Function{
		Id:         functionRefID(fn),
		Name:       fn.Function.Name,
		Slug:       functionRefID(fn),
		IsPaused:   !fn.PausedAt.IsZero() && fn.PausedAt.Before(time.Now()),
		IsArchived: !fn.ArchivedAt.IsZero() && fn.ArchivedAt.Before(time.Now()),
		App:        toFunctionApp(fn),
		Triggers:   toFunctionTriggers(fn.Function.Triggers),
		Configuration: toFunctionConfiguration(
			fn.Function,
			planConcurrencyLimit,
		),
	}
}

func toFunctionApp(fn inngest.DeployedFunction) *apiv2.FunctionApp {
	return &apiv2.FunctionApp{
		Id: appRefID(fn),
	}
}

func toFunctionTriggers(triggers inngest.MultipleTriggers) []*apiv2.FunctionTrigger {
	result := make([]*apiv2.FunctionTrigger, 0, len(triggers))
	for _, trigger := range triggers {
		if trigger.EventTrigger != nil {
			result = append(result, &apiv2.FunctionTrigger{
				Type:      apiv2.FunctionTriggerType_FUNCTION_TRIGGER_TYPE_EVENT,
				Value:     trigger.Event,
				Condition: trigger.Expression,
			})
		}
		if trigger.CronTrigger != nil {
			result = append(result, &apiv2.FunctionTrigger{
				Type:  apiv2.FunctionTriggerType_FUNCTION_TRIGGER_TYPE_CRON,
				Value: trigger.Cron,
			})
		}
	}
	return result
}

func toFunctionConfiguration(fn inngest.Function, planConcurrencyLimit int) *apiv2.FunctionConfiguration {
	if len(fn.Steps) == 0 {
		fn.Steps = []inngest.Step{{}}
	}

	config := models.ToFunctionConfiguration(&fn, planConcurrencyLimit)
	if config == nil {
		return nil
	}

	return &apiv2.FunctionConfiguration{
		Cancellations: toFunctionCancellationConfigurations(config.Cancellations),
		Retries:       toFunctionRetryConfiguration(config.Retries),
		Priority:      config.Priority,
		EventsBatch:   toFunctionEventsBatchConfiguration(config.EventsBatch),
		Concurrency:   toFunctionConcurrencyConfigurations(config.Concurrency),
		RateLimit:     toFunctionRateLimitConfiguration(config.RateLimit),
		Debounce:      toFunctionDebounceConfiguration(config.Debounce),
		Throttle:      toFunctionThrottleConfiguration(config.Throttle),
		Singleton:     toFunctionSingletonConfiguration(config.Singleton),
	}
}

func toFunctionCancellationConfigurations(configs []*models.CancellationConfiguration) []*apiv2.FunctionCancellationConfiguration {
	result := make([]*apiv2.FunctionCancellationConfiguration, 0, len(configs))
	for _, config := range configs {
		result = append(result, &apiv2.FunctionCancellationConfiguration{
			Event:     config.Event,
			Timeout:   config.Timeout,
			Condition: config.Condition,
		})
	}
	return result
}

func toFunctionRetryConfiguration(config *models.RetryConfiguration) *apiv2.FunctionRetryConfiguration {
	if config == nil {
		return nil
	}
	return &apiv2.FunctionRetryConfiguration{
		Value:     int32(config.Value),
		IsDefault: config.IsDefault,
	}
}

func toFunctionEventsBatchConfiguration(config *models.EventsBatchConfiguration) *apiv2.FunctionEventsBatchConfiguration {
	if config == nil {
		return nil
	}
	return &apiv2.FunctionEventsBatchConfiguration{
		MaxSize: int32(config.MaxSize),
		Timeout: config.Timeout,
		Key:     config.Key,
	}
}

func toFunctionConcurrencyConfigurations(configs []*models.ConcurrencyConfiguration) []*apiv2.FunctionConcurrencyConfiguration {
	result := make([]*apiv2.FunctionConcurrencyConfiguration, 0, len(configs))
	for _, config := range configs {
		item := &apiv2.FunctionConcurrencyConfiguration{
			Scope: toFunctionConcurrencyScope(config.Scope),
			Key:   config.Key,
		}
		if config.Limit != nil {
			item.Limit = &apiv2.FunctionConcurrencyLimitConfiguration{
				Value:       int32(config.Limit.Value),
				IsPlanLimit: config.Limit.IsPlanLimit,
			}
		}
		result = append(result, item)
	}
	return result
}

func toFunctionConcurrencyScope(scope models.ConcurrencyScope) apiv2.FunctionConcurrencyScope {
	switch scope {
	case models.ConcurrencyScopeAccount:
		return apiv2.FunctionConcurrencyScope_FUNCTION_CONCURRENCY_SCOPE_ACCOUNT
	case models.ConcurrencyScopeEnvironment:
		return apiv2.FunctionConcurrencyScope_FUNCTION_CONCURRENCY_SCOPE_ENVIRONMENT
	default:
		return apiv2.FunctionConcurrencyScope_FUNCTION_CONCURRENCY_SCOPE_FUNCTION
	}
}

func toFunctionRateLimitConfiguration(config *models.RateLimitConfiguration) *apiv2.FunctionRateLimitConfiguration {
	if config == nil {
		return nil
	}
	return &apiv2.FunctionRateLimitConfiguration{
		Limit:  int32(config.Limit),
		Period: config.Period,
		Key:    config.Key,
	}
}

func toFunctionDebounceConfiguration(config *models.DebounceConfiguration) *apiv2.FunctionDebounceConfiguration {
	if config == nil {
		return nil
	}
	return &apiv2.FunctionDebounceConfiguration{
		Period: config.Period,
		Key:    config.Key,
	}
}

func toFunctionThrottleConfiguration(config *models.ThrottleConfiguration) *apiv2.FunctionThrottleConfiguration {
	if config == nil {
		return nil
	}
	return &apiv2.FunctionThrottleConfiguration{
		Burst:  int32(config.Burst),
		Key:    config.Key,
		Limit:  int32(config.Limit),
		Period: config.Period,
	}
}

func toFunctionSingletonConfiguration(config *models.SingletonConfiguration) *apiv2.FunctionSingletonConfiguration {
	if config == nil {
		return nil
	}
	return &apiv2.FunctionSingletonConfiguration{
		Mode: toFunctionSingletonMode(config.Mode),
		Key:  config.Key,
	}
}

func toFunctionSingletonMode(mode models.SingletonMode) apiv2.FunctionSingletonMode {
	switch mode {
	case models.SingletonModeCancel:
		return apiv2.FunctionSingletonMode_FUNCTION_SINGLETON_MODE_CANCEL
	default:
		return apiv2.FunctionSingletonMode_FUNCTION_SINGLETON_MODE_SKIP
	}
}
