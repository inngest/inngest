package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/shirou/gopsutil/process"
	"github.com/stretchr/testify/require"
)

type mockserver struct {
	name string
	pre  func(ctx context.Context) error
	run  func(ctx context.Context) error
	stop func(ctx context.Context) error

	startTimeout time.Duration
	runTimeout   time.Duration
}

func (m mockserver) Name() string {
	if m.name != "" {
		return m.name
	}
	return "mock"
}

func (m mockserver) Pre(ctx context.Context) error {
	return m.pre(ctx)
}

func (m mockserver) Run(ctx context.Context) error {
	return m.run(ctx)
}

func (m mockserver) Stop(ctx context.Context) error {
	return m.stop(ctx)
}

func (m mockserver) StartTimeout() time.Duration {
	if m.startTimeout != 0 {
		return m.startTimeout
	}
	return defaultTimeout
}

func (m mockserver) RunTimeout() time.Duration {
	if m.runTimeout != 0 {
		return m.runTimeout
	}
	return defaultTimeout
}

func TestStart(t *testing.T) {
	m := mockserver{
		pre:  func(ctx context.Context) error { return nil },
		run:  func(ctx context.Context) error { <-time.After(500 * time.Millisecond); return nil },
		stop: func(ctx context.Context) error { return nil },
	}
	now := time.Now()
	err := Start(context.Background(), m)
	require.NoError(t, err)
	require.WithinDuration(t, time.Now(), now.Add(500*time.Millisecond), 25*time.Millisecond)
}

func TestSignals(t *testing.T) {
	signals := map[string]syscall.Signal{
		"SIGTERM": syscall.SIGTERM,
		"SIGINT":  syscall.SIGINT,
	}

	for name, sig := range signals {

		t.Run(fmt.Sprintf("%s: It waits the full run timeout to end", name), func(t *testing.T) {
			timeout := 5 * time.Second
			m := mockserver{
				pre: func(ctx context.Context) error { return nil },
				run: func(ctx context.Context) error {
					start := time.Now()
					i := 0
					// Run for longer than the timeout, ensuring that the service ends with ErrRunTimeout.
					for time.Now().Before(start.Add(timeout * 2)) {
						if i > 2 {
							require.NotNil(t, ctx.Err(), "expected ctx to be cancelled with signal")
						}
						i++
						<-time.After(time.Second)
					}
					return nil
				},
				runTimeout: timeout,
				stop:       func(ctx context.Context) error { return nil },
			}

			// Track when the fn is done.
			wg := sync.WaitGroup{}
			wg.Add(1)

			go func() {
				// Block until the service finishes.
				err := Start(context.Background(), m)
				// The server above does not finish before the timeout.
				require.Equal(t, true, errors.Is(err, ErrRunTimeout))
				wg.Done()
			}()

			<-time.After(50 * time.Millisecond)

			start := time.Now()

			// XXX: We use gopsutil for support on linux and windows.
			p, err := process.NewProcess(int32(syscall.Getpid()))
			require.NoError(t, err)
			err = p.SendSignal(sig)
			require.NoError(t, err)

			wg.Wait()

			// The timeout should end after runTimeout.
			require.WithinDuration(t, time.Now(), start.Add(m.runTimeout), 50*time.Millisecond)
		})

		t.Run(fmt.Sprintf("%s: Run context is cancelled and we end the fn", name), func(t *testing.T) {
			timeout := 5 * time.Second
			m := mockserver{
				pre: func(ctx context.Context) error { return nil },
				run: func(ctx context.Context) error {
					select {
					case <-ctx.Done():
						<-time.After(50 * time.Millisecond)
						return nil
					case <-time.After(time.Hour):
					}
					return nil
				},
				runTimeout: timeout,
				stop:       func(ctx context.Context) error { return nil },
			}

			// Track when the fn is done.
			wg := sync.WaitGroup{}
			wg.Add(1)

			go func() {
				// Block until the service finishes.
				err := Start(context.Background(), m)
				// The server above does not finish before the timeout.
				require.NoError(t, err)
				wg.Done()
			}()

			<-time.After(50 * time.Millisecond)

			start := time.Now()
			p, err := process.NewProcess(int32(syscall.Getpid()))
			require.NoError(t, err)
			err = p.SendSignal(sig)
			require.NoError(t, err)

			wg.Wait()

			// The timeout should end almost immediately.
			require.WithinDuration(t, time.Now(), start, 100*time.Millisecond)
		})
	}
}

func TestPreError(t *testing.T) {
	m := mockserver{
		pre:  func(ctx context.Context) error { return fmt.Errorf("pre error") },
		run:  func(ctx context.Context) error { return nil },
		stop: func(ctx context.Context) error { return nil },
	}

	err := Start(context.Background(), m)
	require.Error(t, err)
	require.ErrorContains(t, err, "pre error")
}

func TestPreTimeout(t *testing.T) {
	m := mockserver{
		pre:          func(ctx context.Context) error { <-time.After(time.Second); return nil },
		run:          func(ctx context.Context) error { return nil },
		stop:         func(ctx context.Context) error { return nil },
		startTimeout: time.Millisecond,
	}
	err := Start(context.Background(), m)
	require.Error(t, err)
	require.ErrorContains(t, err, ErrPreTimeout.Error())
}

func TestStartAll(t *testing.T) {
	var invocations int32
	m := mockserver{
		pre: func(ctx context.Context) error { return nil },
		run: func(ctx context.Context) error {
			atomic.AddInt32(&invocations, 1)
			<-time.After(500 * time.Millisecond)
			return nil
		},
		stop: func(ctx context.Context) error { return nil },
	}
	now := time.Now()
	err := StartAll(context.Background(), m, m, m)
	require.NoError(t, err)
	require.WithinDuration(t, time.Now(), now.Add(500*time.Millisecond), 10*time.Millisecond)
	require.Equal(t, int32(3), atomic.LoadInt32(&invocations))
}

// TestSingleSvcError ensures that all services shut down if one service errors.
func TestSingleSvcError(t *testing.T) {
	var invocations int32
	var stops int32
	m := mockserver{
		pre: func(ctx context.Context) error { return nil },
		run: func(ctx context.Context) error {
			atomic.AddInt32(&invocations, 1)
			if atomic.LoadInt32(&invocations) == 1 {
				// The first service should error.
				<-time.After(500 * time.Millisecond)
				return fmt.Errorf("boo")
			}
			//The others should run.
			select {
			case <-time.After(time.Minute):
			case <-ctx.Done():
			}
			return nil
		},
		stop: func(ctx context.Context) error {
			atomic.AddInt32(&stops, 1)
			return nil
		},
	}
	now := time.Now()
	err := StartAll(context.Background(), m, m, m)
	require.Error(t, err, "expected service to return an error")
	require.ErrorContains(t, err, "boo")
	require.WithinDuration(t, time.Now(), now.Add(500*time.Millisecond), 10*time.Millisecond)
	require.Equal(t, int32(3), atomic.LoadInt32(&invocations))
	require.Equal(t, int32(3), atomic.LoadInt32(&stops))
}
