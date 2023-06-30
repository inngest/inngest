package cqrs

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Function struct {
	ID        uuid.UUID
	AppID     uuid.UUID
	Config    string
	Name      string
	Slug      string
	CreatedAt time.Time
}

type FunctionManager interface {
	FunctionReader
	FunctionWriter
}

type FunctionReader interface {
	GetAppFunctions(ctx context.Context, appID uuid.UUID) ([]*Function, error)
	GetFunctions(ctx context.Context) ([]*Function, error)
}

type FunctionWriter interface {
	InsertFunction(ctx context.Context, params InsertFunctionParams) (*Function, error)
}

type InsertFunctionParams struct {
	ID        uuid.UUID
	AppID     uuid.UUID
	Name      string
	Slug      string
	Config    string
	CreatedAt time.Time
}
