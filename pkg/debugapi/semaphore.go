package debugapi

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/inngest"
	pb "github.com/inngest/inngest/proto/gen/debug/v1"
)

func (d *debugAPI) GetSemaphoreLevel(ctx context.Context, req *pb.SemaphoreLevelRequest) (*pb.SemaphoreLevelResponse, error) {
	if d.sm == nil {
		return nil, fmt.Errorf("semaphore manager not configured")
	}

	accountID, err := uuid.Parse(req.GetAccountId())
	if err != nil {
		return nil, fmt.Errorf("invalid account_id: %w", err)
	}
	if req.GetName() == "" {
		return nil, fmt.Errorf("name is required")
	}

	return d.getSemaphoreLevel(ctx, accountID, req.GetName(), req.GetKey())
}

func (d *debugAPI) GetAppSemaphoreLevel(ctx context.Context, req *pb.AppSemaphoreLevelRequest) (*pb.SemaphoreLevelResponse, error) {
	if d.sm == nil {
		return nil, fmt.Errorf("semaphore manager not configured")
	}

	accountID, err := uuid.Parse(req.GetAccountId())
	if err != nil {
		return nil, fmt.Errorf("invalid account_id: %w", err)
	}
	appID, err := uuid.Parse(req.GetAppId())
	if err != nil {
		return nil, fmt.Errorf("invalid app_id: %w", err)
	}

	return d.getSemaphoreLevel(ctx, accountID, constraintapi.SemaphoreIDApp(appID), req.GetKey())
}

func (d *debugAPI) GetFunctionSemaphoreLevel(ctx context.Context, req *pb.FunctionSemaphoreLevelRequest) (*pb.SemaphoreLevelResponse, error) {
	if d.sm == nil {
		return nil, fmt.Errorf("semaphore manager not configured")
	}

	accountID, err := uuid.Parse(req.GetAccountId())
	if err != nil {
		return nil, fmt.Errorf("invalid account_id: %w", err)
	}
	functionID, err := uuid.Parse(req.GetFunctionId())
	if err != nil {
		return nil, fmt.Errorf("invalid function_id: %w", err)
	}

	name, err := d.functionSemaphoreName(ctx, functionID)
	if err != nil {
		return nil, err
	}

	return d.getSemaphoreLevel(ctx, accountID, name, req.GetKey())
}

func (d *debugAPI) SetSemaphoreLevel(ctx context.Context, req *pb.SetSemaphoreLevelRequest) (*pb.SetSemaphoreLevelResponse, error) {
	if d.sm == nil {
		return nil, fmt.Errorf("semaphore manager not configured")
	}

	accountID, err := uuid.Parse(req.GetAccountId())
	if err != nil {
		return nil, fmt.Errorf("invalid account_id: %w", err)
	}
	if req.GetName() == "" {
		return nil, fmt.Errorf("name is required")
	}

	return d.setSemaphoreLevel(ctx, accountID, req.GetName(), req.GetKey(), req.GetCapacity(), req.GetIdempotencyKey())
}

func (d *debugAPI) SetAppSemaphoreLevel(ctx context.Context, req *pb.SetAppSemaphoreLevelRequest) (*pb.SetSemaphoreLevelResponse, error) {
	if d.sm == nil {
		return nil, fmt.Errorf("semaphore manager not configured")
	}

	accountID, err := uuid.Parse(req.GetAccountId())
	if err != nil {
		return nil, fmt.Errorf("invalid account_id: %w", err)
	}
	appID, err := uuid.Parse(req.GetAppId())
	if err != nil {
		return nil, fmt.Errorf("invalid app_id: %w", err)
	}

	return d.setSemaphoreLevel(ctx, accountID, constraintapi.SemaphoreIDApp(appID), req.GetKey(), req.GetCapacity(), req.GetIdempotencyKey())
}

func (d *debugAPI) SetFunctionSemaphoreLevel(ctx context.Context, req *pb.SetFunctionSemaphoreLevelRequest) (*pb.SetSemaphoreLevelResponse, error) {
	if d.sm == nil {
		return nil, fmt.Errorf("semaphore manager not configured")
	}

	accountID, err := uuid.Parse(req.GetAccountId())
	if err != nil {
		return nil, fmt.Errorf("invalid account_id: %w", err)
	}
	functionID, err := uuid.Parse(req.GetFunctionId())
	if err != nil {
		return nil, fmt.Errorf("invalid function_id: %w", err)
	}

	name, err := d.functionSemaphoreName(ctx, functionID)
	if err != nil {
		return nil, err
	}

	return d.setSemaphoreLevel(ctx, accountID, name, req.GetKey(), req.GetCapacity(), req.GetIdempotencyKey())
}

func (d *debugAPI) getSemaphoreLevel(ctx context.Context, accountID uuid.UUID, name, key string) (*pb.SemaphoreLevelResponse, error) {
	capacity, usage, err := d.sm.GetCapacity(ctx, accountID, name, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get semaphore level: %w", err)
	}

	return &pb.SemaphoreLevelResponse{
		Level: &pb.SemaphoreLevel{
			Name:      name,
			Key:       key,
			Capacity:  capacity,
			Usage:     usage,
			Remaining: capacity - usage,
		},
	}, nil
}

func (d *debugAPI) setSemaphoreLevel(ctx context.Context, accountID uuid.UUID, name, key string, capacity int64, idempotencyKey string) (*pb.SetSemaphoreLevelResponse, error) {
	if capacity < 0 {
		return nil, fmt.Errorf("capacity must be >= 0")
	}
	if idempotencyKey == "" {
		return nil, fmt.Errorf("idempotency_key is required")
	}

	result, err := d.sm.SetCapacity(ctx, accountID, name, idempotencyKey, capacity)
	if err != nil {
		return nil, fmt.Errorf("failed to set semaphore level: %w", err)
	}

	level, err := d.getSemaphoreLevel(ctx, accountID, name, key)
	if err != nil {
		return nil, err
	}

	return &pb.SetSemaphoreLevelResponse{
		Applied: result.Applied,
		Level:   level.Level,
	}, nil
}

func (d *debugAPI) functionSemaphoreName(ctx context.Context, functionID uuid.UUID) (string, error) {
	if d.db == nil {
		return constraintapi.SemaphoreIDFn(functionID), nil
	}

	fn, err := d.db.GetFunctionByInternalUUID(ctx, functionID)
	if err != nil {
		return "", fmt.Errorf("could not retrieve function: %w", err)
	}

	inngestFunction, err := fn.InngestFunction()
	if err != nil {
		return "", fmt.Errorf("could not parse function config: %w", err)
	}
	if inngestFunction.Concurrency == nil {
		return constraintapi.SemaphoreIDFn(functionID), nil
	}

	var fnConcurrency *inngest.FnConcurrency
	for i := range inngestFunction.Concurrency.Fn {
		fc := &inngestFunction.Concurrency.Fn[i]
		if fc.EffectiveScope() != inngest.FnConcurrencyScopeFn {
			continue
		}
		if fnConcurrency != nil {
			return "", fmt.Errorf("function has multiple function-scoped semaphore concurrency limits")
		}
		fnConcurrency = fc
	}
	if fnConcurrency == nil || fnConcurrency.Key == nil {
		return constraintapi.SemaphoreIDFn(functionID), nil
	}

	return constraintapi.SemaphoreIDFnKey(functionID, *fnConcurrency.Key), nil
}
