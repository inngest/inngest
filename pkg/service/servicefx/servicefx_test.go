package servicefx

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

// events records the order of lifecycle calls across services and hooks.
type events struct {
	mu   sync.Mutex
	list []string
}

func (e *events) add(event string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.list = append(e.list, event)
}

func (e *events) get() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]string(nil), e.list...)
}

type testService struct {
	events *events
	preErr error
	// run is called by Run.  nil blocks until the context is cancelled.
	run func(context.Context) error
}

func (s *testService) Name() string {
	return "test"
}

func (s *testService) Pre(context.Context) error {
	s.events.add("pre")
	return s.preErr
}

func (s *testService) Run(ctx context.Context) error {
	s.events.add("run")
	if s.run != nil {
		return s.run(ctx)
	}
	<-ctx.Done()
	return nil
}

func (s *testService) Stop(context.Context) error {
	s.events.add("stop")
	return nil
}

type dependency struct{}

func newTestApp(t *testing.T, svc *testService) (*fx.App, *Runner) {
	var runner *Runner
	app := fx.New(
		fx.NopLogger,
		fx.Provide(func(ctx context.Context, lifecycle fx.Lifecycle) *dependency {
			lifecycle.Append(fx.StopHook(func() {
				if ctx.Err() != nil {
					svc.events.add("app context cancelled")
				}
				svc.events.add("dependency closed")
			}))
			return &dependency{}
		}),
		Provide(func(*dependency) *testService {
			return svc
		}),
		Module(t.Context()),
		fx.Populate(&runner),
	)
	require.NoError(t, app.Err())
	return app, runner
}

func TestRunnerStopOrder(t *testing.T) {
	svc := &testService{
		events: &events{},
	}
	app, runner := newTestApp(t, svc)

	require.NoError(t, app.Start(t.Context()))
	require.Eventually(t, func() bool {
		return len(svc.events.get()) == 2
	}, time.Second, time.Millisecond)
	require.NoError(t, app.Stop(t.Context()))

	require.Equal(t, []string{
		"pre",
		"run",
		"stop",
		"app context cancelled",
		"dependency closed",
	}, svc.events.get())
	require.NoError(t, runner.Err())
}

func TestRunnerRunReturns(t *testing.T) {
	tests := []struct {
		name         string
		runErr       error
		wantExitCode int
		wantErr      string
	}{
		{
			name:         "error",
			runErr:       errors.New("subscription closed"),
			wantExitCode: 1,
			wantErr:      "service test errored: subscription closed",
		},
		{
			name:         "nil",
			wantExitCode: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &testService{
				events: &events{},
				run: func(context.Context) error {
					return tt.runErr
				},
			}
			app, runner := newTestApp(t, svc)
			shutdown := app.Wait()
			require.NoError(t, app.Start(t.Context()))

			select {
			case signal := <-shutdown:
				require.Equal(t, tt.wantExitCode, signal.ExitCode)
			case <-time.After(5 * time.Second):
				t.Fatal("the service did not stop the app")
			}
			require.NoError(t, app.Stop(t.Context()))

			if tt.wantErr == "" {
				require.NoError(t, runner.Err())
				return
			}
			require.EqualError(t, runner.Err(), tt.wantErr)
		})
	}
}

func TestRunnerPreError(t *testing.T) {
	svc := &testService{
		events: &events{},
		preErr: errors.New("port in use"),
	}
	app, _ := newTestApp(t, svc)

	require.ErrorContains(t, app.Start(t.Context()), "service test: port in use")
	// Run is not called, and Fx closes the dependencies that are built.
	require.Equal(t, []string{
		"pre",
		"app context cancelled",
		"dependency closed",
	}, svc.events.get())
}

func TestRunnerTask(t *testing.T) {
	tests := []struct {
		name string
		// run is the Run of the task.  nil blocks until the context is
		// cancelled.
		run          func(context.Context) error
		wantShutdown bool
		wantErr      string
		wantEvents   []string
	}{
		{
			name: "runs until the app stops",
			wantEvents: []string{
				"start",
				"run",
				"stop",
				"app context cancelled",
				"dependency closed",
			},
		},
		{
			name: "error before the app stops",
			run: func(context.Context) error {
				return errors.New("subscription closed")
			},
			wantShutdown: true,
			wantErr:      "service task errored: subscription closed",
			wantEvents: []string{
				"start",
				"run",
				"stop",
				"app context cancelled",
				"dependency closed",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev := &events{}
			var runner *Runner
			app := fx.New(
				fx.NopLogger,
				fx.Provide(func(ctx context.Context, lifecycle fx.Lifecycle) *dependency {
					lifecycle.Append(fx.StopHook(func() {
						if ctx.Err() != nil {
							ev.add("app context cancelled")
						}
						ev.add("dependency closed")
					}))
					return &dependency{}
				}),
				ProvideTasks(func(*dependency) Task {
					return Task{
						Name: "task",
						Start: func(context.Context) error {
							ev.add("start")
							return nil
						},
						Run: func(ctx context.Context) error {
							ev.add("run")
							if tt.run != nil {
								return tt.run(ctx)
							}
							<-ctx.Done()
							return nil
						},
						Stop: func(context.Context) error {
							ev.add("stop")
							return nil
						},
					}
				}),
				Module(t.Context()),
				fx.Populate(&runner),
			)
			require.NoError(t, app.Err())
			shutdown := app.Wait()
			require.NoError(t, app.Start(t.Context()))

			if tt.wantShutdown {
				select {
				case <-shutdown:
				case <-time.After(5 * time.Second):
					t.Fatal("the task did not stop the app")
				}
			} else {
				require.Eventually(t, func() bool {
					return len(ev.get()) == 2
				}, time.Second, time.Millisecond)
			}
			require.NoError(t, app.Stop(t.Context()))

			require.Equal(t, tt.wantEvents, ev.get())
			if tt.wantErr == "" {
				require.NoError(t, runner.Err())
				return
			}
			require.EqualError(t, runner.Err(), tt.wantErr)
		})
	}
}
