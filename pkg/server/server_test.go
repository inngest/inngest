package server

import (
	"context"
	"fmt"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type mockserver struct {
	name string
	pre  func(ctx context.Context) error
	run  func(ctx context.Context) error
	stop func(ctx context.Context) error

	startTimeout time.Duration
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

func TestStart(t *testing.T) {
	m := mockserver{
		pre:  func(ctx context.Context) error { return nil },
		run:  func(ctx context.Context) error { <-time.After(50 * time.Millisecond); return nil },
		stop: func(ctx context.Context) error { return nil },
	}
	now := time.Now()
	err := Start(context.Background(), m)
	require.NoError(t, err)
	require.WithinDuration(t, time.Now(), now.Add(50*time.Millisecond), 5*time.Millisecond)
}

func TestSigint(t *testing.T) {
	m := mockserver{
		pre:  func(ctx context.Context) error { return nil },
		run:  func(ctx context.Context) error { <-time.After(time.Hour); return nil },
		stop: func(ctx context.Context) error { return nil },
	}

	wg := sync.WaitGroup{}
	wg.Add(1)

	now := time.Now()
	go func() {
		err := Start(context.Background(), m)
		require.NoError(t, err)
		wg.Done()
	}()

	<-time.After(50 * time.Millisecond)
	err := syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	require.NoError(t, err)

	wg.Wait()
	require.WithinDuration(t, time.Now(), now.Add(50*time.Millisecond), 5*time.Millisecond)
}

func TestSigterm(t *testing.T) {
	m := mockserver{
		pre:  func(ctx context.Context) error { return nil },
		run:  func(ctx context.Context) error { <-time.After(time.Hour); return nil },
		stop: func(ctx context.Context) error { return nil },
	}

	wg := sync.WaitGroup{}
	wg.Add(1)

	now := time.Now()
	go func() {
		err := Start(context.Background(), m)
		require.NoError(t, err)
		wg.Done()
	}()

	<-time.After(50 * time.Millisecond)
	err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	require.NoError(t, err)

	wg.Wait()
	require.WithinDuration(t, time.Now(), now.Add(50*time.Millisecond), 5*time.Millisecond)
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
