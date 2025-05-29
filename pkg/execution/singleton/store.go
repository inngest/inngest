package singleton

import (
	"context"
)

type SingletonStore interface {
	Exists(ctx context.Context, key string) (bool, error)
}
