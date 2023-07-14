package cqrs

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/inngest"
)

type Function struct {
	ID        uuid.UUID
	AppID     uuid.UUID
	Config    string
	Name      string
	Slug      string
	CreatedAt time.Time
}

func (f Function) InngestFunction() (*inngest.Function, error) {
	fn := inngest.Function{}
	err := json.Unmarshal([]byte(f.Config), &fn)
	if err != nil {
		return nil, err
	}
	return &fn, nil
}

type FunctionManager interface {
	FunctionReader
	FunctionWriter
}

type FunctionReader interface {
	GetAppFunctions(ctx context.Context, appID uuid.UUID) ([]*Function, error)
	GetFunctions(ctx context.Context) ([]*Function, error)
	GetFunctionByID(ctx context.Context, id uuid.UUID) (*Function, error)
}

type FunctionWriter interface {
	InsertFunction(ctx context.Context, params InsertFunctionParams) (*Function, error)
	UpdateFunctionConfig(ctx context.Context, arg UpdateFunctionConfigParams) (*Function, error)
	// DeleteFunctionsByAppID deletes all functions for a specific app.
	DeleteFunctionsByAppID(ctx context.Context, appID uuid.UUID) error
	// DeleteFunctionsByIDs deletes all functions with the given IDs
	DeleteFunctionsByIDs(ctx context.Context, ids []uuid.UUID) error
}

type InsertFunctionParams struct {
	ID        uuid.UUID
	AppID     uuid.UUID
	Name      string
	Slug      string
	Config    string
	CreatedAt time.Time
}

type UpdateFunctionConfigParams struct {
	Config string
	ID     uuid.UUID
}
