package telemetry

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace"
)

var (
	userTracer Tracer
	o          sync.Once
)

type Tracer interface {
	Provider() *trace.TracerProvider
	Propagator() propagation.TextMapPropagator
	Shutdown(ctx context.Context) func()
}

type TracerOpts struct {
	ServiceName string
	Type        TracerType
}

func NewUserTracer(ctx context.Context, opts TracerOpts) error {
	var err error
	o.Do(func() {
		userTracer, err = newTracer(ctx, opts)
	})
	return err
}

func UserTracer() Tracer {
	if userTracer == nil {
		if err := NewUserTracer(context.Background(), TracerOpts{
			ServiceName: "default",
			Type:        TracerTypeNoop,
		}); err != nil {
			panic("fail to setup default user tracer")
		}
	}
	return userTracer
}

func CloseUserTracer(ctx context.Context) error {
	if userTracer != nil {
		userTracer.Shutdown(ctx)
	}
	return nil
}
