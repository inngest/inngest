package loader

import (
	"context"
	"fmt"
	"net/http"

	"github.com/graph-gophers/dataloader"
	"github.com/inngest/inngest/pkg/cqrs"
)

type ctxKey string

const (
	loadersKey = ctxKey("dataloaders")
)

type LoaderParams struct {
	DB cqrs.Manager
}

// Middleware attaches a new DataLoader to each request context. DataLoader's
// cache is designed to be short-lived (e.g. the life of an HTTP request). It
// doesn't seem to have any cache invalidation.
func Middleware(params LoaderParams) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			loaders := newLoaders(params)
			ctx := toCtx(r.Context(), loaders)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// FromCtx returns the DataLoader from the context.
func FromCtx(ctx context.Context) *loaders {
	return ctx.Value(loadersKey).(*loaders)
}

// toCtx sets the DataLoader on the context.
func toCtx(ctx context.Context, loaders *loaders) context.Context {
	return context.WithValue(ctx, loadersKey, loaders)
}

// LoadOne loads a single item from the given loader.
func LoadOne[T interface{}](
	ctx context.Context,
	loader *dataloader.Loader,
	key dataloader.Key,
) (*T, error) {
	thunk := loader.Load(ctx, key)
	result, err := thunk()
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, nil
	}

	directOutput, ok := result.(T)
	if ok {
		return &directOutput, nil
	}

	ptrOutput, ok := result.(*T)
	if ok {
		return ptrOutput, nil
	}

	return nil, fmt.Errorf("unexpected type %T", result)
}

// LoadMany loads many items from the given loader.
func LoadMany[T interface{}](
	ctx context.Context,
	loader *dataloader.Loader,
	keys dataloader.Keys,
) ([]T, error) {
	thunkMany := loader.LoadMany(ctx, keys)
	results, errs := thunkMany()
	if len(errs) > 0 {
		return []T{}, errs[1]
	}

	output := make([]T, len(keys))
	for i, result := range results {
		typedResult, ok := result.(T)
		if !ok {
			typedResultP, ok := result.(*T)
			if !ok {
				return []T{}, fmt.Errorf("unexpected type %T", result)
			}

			typedResult = *typedResultP
		}
		output[i] = typedResult
	}

	return output, nil
}

// LoadOneWithString loads a single item from the given loader with a string
// key.
func LoadOneWithString[T interface{}](
	ctx context.Context,
	loader *dataloader.Loader,
	key string,
) (*T, error) {
	return LoadOne[T](ctx, loader, dataloader.StringKey(key))
}

// LoadManyWithString loads many items from the given loader using string keys.
func LoadManyWithString[T interface{}](
	ctx context.Context,
	loader *dataloader.Loader,
	keys []string,
) ([]T, error) {
	return LoadMany[T](ctx, loader, dataloader.NewKeysFromStrings(keys))
}

type loaders struct {
	RunTraceLoader       *dataloader.Loader
	LegacyRunTraceLoader *dataloader.Loader
	RunSpanLoader        *dataloader.Loader
	EventLoader          *dataloader.Loader
	DebugRunLoader       *dataloader.Loader
	DebugSessionLoader   *dataloader.Loader
}

func newLoaders(params LoaderParams) *loaders {
	loaders := &loaders{}
	tr := &traceReader{loaders: loaders, reader: params.DB}
	er := &eventReader{loaders: loaders, reader: params.DB}

	loaders.RunTraceLoader = dataloader.NewBatchedLoader(tr.GetRunTrace)
	loaders.LegacyRunTraceLoader = dataloader.NewBatchedLoader(tr.GetLegacyRunTrace)
	loaders.RunSpanLoader = dataloader.NewBatchedLoader(tr.GetLegacySpanRun)
	loaders.EventLoader = dataloader.NewBatchedLoader(er.GetEvents)
	loaders.DebugRunLoader = dataloader.NewBatchedLoader(tr.GetDebugRunTrace)
	loaders.DebugSessionLoader = dataloader.NewBatchedLoader(tr.GetDebugSessionTrace)

	return loaders
}
