package executor

import (
	"context"
	"time"
)

// StopTimeout returns the overall stop budget for the executor.
// The Helm chart sets terminationGracePeriodSeconds=120 with a 15s
// preStop hook, leaving ~105s.  We use 90s as the total budget;
// the service framework gives Stop() 80% of this (72s) for
// in-flight queue items and reserves the rest for the global
// waitgroup drain.
func (s *svc) StopTimeout() time.Duration { return 90 * time.Second }

func (s *svc) Stop(ctx context.Context) error {
	s.exec.CloseLifecycleListeners(ctx)

	// Wait for all in-flight queue runs to finish, but respect the
	// context deadline so that the service framework's stop timeout
	// is honoured instead of blocking indefinitely.
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
