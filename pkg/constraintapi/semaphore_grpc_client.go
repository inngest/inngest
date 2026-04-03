package constraintapi

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	pb "github.com/inngest/inngest/proto/gen/constraintapi/v1"
	"github.com/inngest/inngest/proto/gen/constraintapi/v1/constraintapiconnect"
)

// grpcSemaphoreManager implements SemaphoreManager by proxying to the constraint API gRPC service.
type grpcSemaphoreManager struct {
	client constraintapiconnect.ConstraintAPIClient
}

// NewGRPCSemaphoreManager returns a SemaphoreManager that delegates to the constraint API gRPC service.
func NewGRPCSemaphoreManager(client constraintapiconnect.ConstraintAPIClient) SemaphoreManager {
	return &grpcSemaphoreManager{client: client}
}

func (m *grpcSemaphoreManager) SetCapacity(ctx context.Context, accountID uuid.UUID, name, idempotencyKey string, capacity int64) error {
	_, err := m.client.SetSemaphoreCapacity(ctx, connect.NewRequest(&pb.SemaphoreSetCapacityRequest{
		AccountId:      accountID.String(),
		Name:           name,
		IdempotencyKey: idempotencyKey,
		Capacity:       capacity,
	}))
	if err != nil {
		return fmt.Errorf("grpc SetSemaphoreCapacity: %w", err)
	}
	return nil
}

func (m *grpcSemaphoreManager) AdjustCapacity(ctx context.Context, accountID uuid.UUID, name, idempotencyKey string, delta int64) error {
	_, err := m.client.AdjustSemaphoreCapacity(ctx, connect.NewRequest(&pb.SemaphoreAdjustCapacityRequest{
		AccountId:      accountID.String(),
		Name:           name,
		IdempotencyKey: idempotencyKey,
		Delta:          delta,
	}))
	if err != nil {
		return fmt.Errorf("grpc AdjustSemaphoreCapacity: %w", err)
	}
	return nil
}

func (m *grpcSemaphoreManager) GetCapacity(ctx context.Context, accountID uuid.UUID, name, usageValue string) (int64, int64, error) {
	resp, err := m.client.GetSemaphoreCapacity(ctx, connect.NewRequest(&pb.SemaphoreGetCapacityRequest{
		AccountId:  accountID.String(),
		Name:       name,
		UsageValue: usageValue,
	}))
	if err != nil {
		return 0, 0, fmt.Errorf("grpc GetSemaphoreCapacity: %w", err)
	}
	return resp.Msg.Capacity, resp.Msg.Usage, nil
}

func (m *grpcSemaphoreManager) ReleaseSemaphore(ctx context.Context, accountID uuid.UUID, name, usageValue, idempotencyKey string, weight int64) error {
	_, err := m.client.ReleaseSemaphore(ctx, connect.NewRequest(&pb.SemaphoreReleaseRequest{
		AccountId:      accountID.String(),
		Name:           name,
		UsageValue:     usageValue,
		IdempotencyKey: idempotencyKey,
		Weight:         weight,
	}))
	if err != nil {
		return fmt.Errorf("grpc ReleaseSemaphore: %w", err)
	}
	return nil
}
