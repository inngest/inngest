package singleton

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/util"
)

var (
	ErrEvaluatingSingletonExpression = fmt.Errorf("singleton expression evaluation failed")
	ErrNotASingleton                 = fmt.Errorf("singleton expression resolved to false")
)

type Singleton interface {
	Singleton(ctx context.Context, key string, c inngest.Singleton) (bool, error)
}

// SingletonKey returns the singleton key given a function ID, singleton config,
// and incoming event data.
func SingletonKey(ctx context.Context, id uuid.UUID, c inngest.Singleton, evt map[string]any) (string, error) {
	if c.Key == nil {
		return id.String(), nil
	}
	eval, err := expressions.NewExpressionEvaluator(ctx, *c.Key)
	if err != nil {
		return "", ErrEvaluatingSingletonExpression
	}
	res, _, err := eval.Evaluate(ctx, expressions.NewData(map[string]any{"event": evt}))
	if err != nil {
		return "", ErrEvaluatingSingletonExpression
	}
	if v, ok := res.(bool); ok && !v {
		return "", ErrNotASingleton
	}

	return hash(res, id), nil
}

func hash(res any, id uuid.UUID) string {
	sum := util.XXHash(res)
	return fmt.Sprintf("%s-%s", id, sum)
}

func singleton(ctx context.Context, store SingletonStore, key string, s inngest.Singleton) (bool, error) {
	result, err := store.Exists(ctx, key)
	if err != nil {
		log.Fatal(err)
	}
	return result, err
}
