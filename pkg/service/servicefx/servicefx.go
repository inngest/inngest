// Package servicefx runs the long-running parts of an Fx app: the tasks that
// ProvideTasks adds, and the service.Service values that Provide adds.
// Module starts them when the app starts and stops them when the app stops.
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

// defaultTimeout is the time Pre and Stop of a service.Service get when the
// service does not set one.  it is equal to the default of package service.
const defaultTimeout = 30 * time.Second

// Task is a part of an app that runs from app start to app stop, such as a
// queue consumer or a server.  every function is optional.
type Task struct {
	Name string

	// Start runs when the app starts.  an error stops the start of the app.
	Start func(ctx context.Context) error

	// Run runs in a new goroutine after every Start returns.  stopping the
	// app cancels ctx.  if Run returns before the app stops, the app stops.
	Run func(ctx context.Context) error

	// Stop runs when the app stops, after Run returns.
	Stop func(ctx context.Context) error
}

// ProvideTasks adds the Task that each constructor returns to the tasks that
// Module runs.
func ProvideTasks(constructors ...any) fx.Option {
	annotated := make([]any, 0, len(constructors))
	for _, constructor := range constructors {
		annotated = append(annotated, fx.Annotate(constructor, fx.ResultTags(`group:"tasks"`)))
	}
	return fx.Provide(annotated...)
}

// Provide adds the service.Service that each constructor returns to the
// services that Module runs, with the lifecycle of service.StartAll.  the app
// fails to build when a result does not implement service.Service.  the
// result is not provided as its own type, so a service that other
// constructors use needs its own provider.
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

// Module runs every task and service.  it also provides the context.Context
// of the app.  stopping the app stops the tasks and services, then cancels
// the context and waits for the goroutines started with service.Go, then
// closes the dependencies.  without this order, goroutines that use the
// context of the app see closed clients.
//
// Module adds an invoke.  Fx runs invokes in the order of the options, so an
// invoke that must run before any task is built goes before Module.
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

// Runner starts and stops the tasks of the app.
type Runner struct {
	tasks      []Task
	shutdowner fx.Shutdowner
	// cancelApp cancels the context that Module provides.
	cancelApp context.CancelFunc
	// ctx is the context of Start and Run.  stop cancels it first.
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
	Tasks      []Task            `group:"tasks"`
	Services   []service.Service `group:"services"`
}

func newRunner(params runnerParams, cancelApp context.CancelFunc) *Runner {
	ctx, cancel := context.WithCancel(params.Context)
	tasks := append([]Task(nil), params.Tasks...)
	for _, svc := range params.Services {
		tasks = append(tasks, serviceTask(svc))
	}

	r := &Runner{
		tasks:      tasks,
		shutdowner: params.Shutdowner,
		cancelApp:  cancelApp,
		ctx:        ctx,
		cancel:     cancel,
	}
	// the dependencies of the tasks are built before this hook is added.  Fx
	// stops hooks in reverse order, so the tasks stop before the dependencies
	// close.
	params.Lifecycle.Append(fx.Hook{
		OnStart: r.start,
		OnStop:  r.stop,
	})
	return r
}

// Err returns the error of the first task whose Run returned an error before
// the app stopped.
func (r *Runner) Err() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.err
}

// start calls Start on every task at the same time, then calls Run on every
// task in a new goroutine.  Start and Run get a context that lives until the
// app stops, because a service.Service can start goroutines in Pre.
func (r *Runner) start(context.Context) error {
	errs := make([]error, len(r.tasks))
	var wg sync.WaitGroup
	for i, task := range r.tasks {
		if task.Start == nil {
			continue
		}
		wg.Go(func() {
			if err := task.Start(r.taskContext(task)); err != nil {
				errs[i] = fmt.Errorf("service %s: %w", task.Name, err)
			}
		})
	}
	wg.Wait()
	if err := errors.Join(errs...); err != nil {
		r.cancel()
		r.cancelApp()
		return err
	}

	for _, task := range r.tasks {
		if task.Run == nil {
			continue
		}
		ctx := r.taskContext(task)
		r.runs.Go(func() {
			log := logger.StdlibLogger(ctx)
			log.Info("service starting", "service", task.Name)
			err := task.Run(ctx)
			if r.ctx.Err() != nil {
				if err != nil && !errors.Is(err, context.Canceled) {
					log.Error("service run errored", "error", err)
				}
				log.Info("service run finished", "service", task.Name)
				return
			}
			// a task that returns stops the app, as in service.StartAll.
			exitCode := 0
			if err != nil && !errors.Is(err, context.Canceled) {
				r.setErr(fmt.Errorf("service %s errored: %w", task.Name, err))
				exitCode = 1
			}
			if err := r.shutdowner.Shutdown(fx.ExitCode(exitCode)); err != nil {
				log.Error("could not stop the app", "error", err)
			}
		})
	}
	return nil
}

// stop cancels the context of every Run and waits for every Run to return.
// then it calls Stop on every task at the same time.  last, it cancels the
// context of the app and waits for the goroutines started with service.Go.
func (r *Runner) stop(stopCtx context.Context) error {
	r.cancel()
	if err := wait(stopCtx, r.runs.Wait); err != nil {
		r.cancelApp()
		return fmt.Errorf("services did not finish running: %w", err)
	}

	errs := make([]error, len(r.tasks))
	var wg sync.WaitGroup
	for i, task := range r.tasks {
		if task.Stop == nil {
			continue
		}
		wg.Go(func() {
			ctx := logger.WithStdlib(stopCtx, logger.StdlibLogger(r.taskContext(task)))
			logger.StdlibLogger(ctx).Info("service cleaning up", "service", task.Name)
			if err := task.Stop(ctx); err != nil && !errors.Is(err, context.Canceled) {
				errs[i] = fmt.Errorf("service %s: %w", task.Name, err)
			}
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

// taskContext adds the caller field to the log lines of one task.
func (r *Runner) taskContext(task Task) context.Context {
	return logger.WithStdlib(r.ctx, logger.StdlibLogger(r.ctx).With("caller", task.Name))
}

// serviceTask runs a service.Service as a Task.  Pre and Stop keep the
// timeouts of package service.
func serviceTask(svc service.Service) Task {
	return Task{
		Name: svc.Name(),
		Start: func(ctx context.Context) error {
			return pre(ctx, svc)
		},
		Run: svc.Run,
		Stop: func(ctx context.Context) error {
			return stop(ctx, svc)
		},
	}
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
		return err
	case <-time.After(timeout):
		return service.ErrPreTimeout
	}
}

// stop calls Stop with a context that is not cancelled, and waits for the
// stop timeout of the service.
func stop(ctx context.Context, svc service.Service) error {
	timeout := defaultTimeout
	if t, ok := svc.(service.StopTimeouter); ok {
		timeout = t.StopTimeout()
	}

	done := make(chan error, 1)
	go func() {
		done <- svc.Stop(context.WithoutCancel(ctx))
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("did not clean up within %s", timeout)
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
