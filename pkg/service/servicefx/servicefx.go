// Package servicefx runs service.Service values in an Fx app, with the
// lifecycle of service.StartAll.  a constructor adds a service with Provide.
// Module starts every provided service when the app starts and stops them
// when the app stops.
package servicefx

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/service"
	"go.uber.org/fx"
)

// defaultTimeout is the time Pre and Stop get when a service does not set
// one.  it is equal to the default of package service.
const defaultTimeout = 30 * time.Second

// Provide adds the service.Service that each constructor returns to the
// services that Module runs.  the app fails to build when a result does not
// implement service.Service.  the result is not provided as its own type, so
// a service that other constructors use needs its own provider.
func Provide(constructors ...any) fx.Option {
	annotated := make([]any, 0, len(constructors))
	for _, constructor := range constructors {
		annotated = append(annotated, fx.Annotate(
			constructor,
			fx.As(new(service.Service)),
			fx.ResultTags(`group:"services"`),
		))
	}
	return fx.Provide(annotated...)
}

// Module runs every service that Provide adds.  it also provides the
// context.Context of the app.  stopping the app stops the services, then
// cancels the context and waits for the goroutines started with service.Go,
// then closes the dependencies.  without this order, goroutines that use the
// context of the app see closed clients.
//
// Module adds an invoke.  Fx runs invokes in the order of the options, so an
// invoke that must run before any service is built goes before Module.
func Module(parent context.Context) fx.Option {
	ctx, cancel := context.WithCancel(parent)
	return fx.Options(
		fx.Provide(
			func() context.Context {
				return ctx
			},
			func(params runnerParams) *Runner {
				return newRunner(params, cancel)
			},
		),
		fx.Invoke(func(*Runner) {}),
	)
}

// Runner starts and stops the services of the app.
type Runner struct {
	services   []service.Service
	shutdowner fx.Shutdowner
	// cancelApp cancels the context that Module provides.
	cancelApp context.CancelFunc
	// ctx is the context of Pre and Run.  stop cancels it first.
	ctx    context.Context
	cancel context.CancelFunc

	runs sync.WaitGroup

	mu  sync.Mutex
	err error
}

type runnerParams struct {
	fx.In

	Context    context.Context
	Lifecycle  fx.Lifecycle
	Shutdowner fx.Shutdowner
	Services   []service.Service `group:"services"`
}

func newRunner(params runnerParams, cancelApp context.CancelFunc) *Runner {
	ctx, cancel := context.WithCancel(params.Context)
	r := &Runner{
		services:   params.Services,
		shutdowner: params.Shutdowner,
		cancelApp:  cancelApp,
		ctx:        ctx,
		cancel:     cancel,
	}
	// the dependencies of the services are built before this hook is added.
	// Fx stops hooks in reverse order, so the services stop before the
	// dependencies close.
	params.Lifecycle.Append(fx.Hook{
		OnStart: r.start,
		OnStop:  r.stop,
	})
	return r
}

// Err returns the error of the first service whose Run returned an error
// before the app stopped.
func (r *Runner) Err() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.err
}

// start calls Pre on every service at the same time, then calls Run on every
// service in a new goroutine.  Pre and Run get a context that lives until the
// app stops, because some services start goroutines in Pre.
func (r *Runner) start(context.Context) error {
	errs := make([]error, len(r.services))
	var wg sync.WaitGroup
	for i, svc := range r.services {
		wg.Go(func() {
			errs[i] = pre(r.serviceContext(svc), svc)
		})
	}
	wg.Wait()
	if err := errors.Join(errs...); err != nil {
		r.cancel()
		r.cancelApp()
		return err
	}

	for _, svc := range r.services {
		ctx := r.serviceContext(svc)
		r.runs.Go(func() {
			logger.StdlibLogger(ctx).Info("service starting", "service", svc.Name())
			err := svc.Run(ctx)
			if r.ctx.Err() != nil {
				if err != nil && !errors.Is(err, context.Canceled) {
					logger.StdlibLogger(ctx).Error("service run errored", "error", err)
				}
				logger.StdlibLogger(ctx).Info("service run finished", "service", svc.Name())
				return
			}
			// a service that returns stops the app, as in service.StartAll.
			exitCode := 0
			if err != nil && !errors.Is(err, context.Canceled) {
				r.setErr(fmt.Errorf("service %s errored: %w", svc.Name(), err))
				exitCode = 1
			}
			if err := r.shutdowner.Shutdown(fx.ExitCode(exitCode)); err != nil {
				logger.StdlibLogger(ctx).Error("could not stop the app", "error", err)
			}
		})
	}
	return nil
}

// stop cancels the context of every Run and waits for every Run to return.
// then it calls Stop on every service at the same time.  last, it cancels the
// context of the app and waits for the goroutines started with service.Go.
func (r *Runner) stop(stopCtx context.Context) error {
	r.cancel()
	if err := wait(stopCtx, r.runs.Wait); err != nil {
		r.cancelApp()
		return fmt.Errorf("services did not finish running: %w", err)
	}

	errs := make([]error, len(r.services))
	var wg sync.WaitGroup
	for i, svc := range r.services {
		wg.Go(func() {
			errs[i] = stop(r.serviceContext(svc), svc)
		})
	}
	wg.Wait()

	r.cancelApp()
	if err := wait(stopCtx, service.Wait); err != nil {
		errs = append(errs, fmt.Errorf("service goroutines did not finish: %w", err))
	}
	return errors.Join(errs...)
}

func (r *Runner) setErr(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err == nil {
		r.err = err
	}
}

// serviceContext adds the caller field to the log lines of one service.
func (r *Runner) serviceContext(svc service.Service) context.Context {
	return logger.WithStdlib(r.ctx, logger.StdlibLogger(r.ctx).With("caller", svc.Name()))
}

// pre calls Pre and waits for the start timeout of the service.  Pre keeps
// running after the timeout, as in package service.
func pre(ctx context.Context, svc service.Service) error {
	timeout := defaultTimeout
	if t, ok := svc.(service.StartTimeouter); ok {
		timeout = t.StartTimeout()
	}

	done := make(chan error, 1)
	go func() {
		done <- svc.Pre(ctx)
	}()
	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("service %s: %w", svc.Name(), err)
		}
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("service %s: %w", svc.Name(), service.ErrPreTimeout)
	}
}

// stop calls Stop with a context that is not cancelled, and waits for the
// stop timeout of the service.
func stop(ctx context.Context, svc service.Service) error {
	timeout := defaultTimeout
	if t, ok := svc.(service.StopTimeouter); ok {
		timeout = t.StopTimeout()
	}

	logger.StdlibLogger(ctx).Info("service cleaning up", "service", svc.Name())
	done := make(chan error, 1)
	go func() {
		done <- svc.Stop(context.WithoutCancel(ctx))
	}()
	select {
	case err := <-done:
		if err != nil && !errors.Is(err, context.Canceled) {
			return fmt.Errorf("service %s: %w", svc.Name(), err)
		}
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("service %s did not clean up within %s", svc.Name(), timeout)
	}
}

// wait calls f and returns when f returns or when ctx is done.
func wait(ctx context.Context, f func()) error {
	done := make(chan struct{})
	go func() {
		defer close(done)
		f()
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
