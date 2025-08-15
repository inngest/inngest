package cqrs

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Account struct {
	ID         uuid.UUID `json:"internal_id"`
	Name       string    `json:"name"`
	QueueShard string    `json:"queue_shard"`
	CreatedAt  time.Time `json:"created_at"`
}

type AccountReader interface {
	GetAccountByUUID(ctx context.Context, id uuid.UUID) (*Account, error)
}
