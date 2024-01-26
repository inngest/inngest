package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/inngest/inngest/pkg/logger"
	"golang.org/x/sync/errgroup"
)

var (
	defaultTimeout = 30 * time.Second

	ErrPreTimeout = fmt.Errorf("service.Pre did not end within the given timeout")
	ErrRunTimeout = fmt.Errorf("service.Run did not end within the given timeout")

	wgctxVal = wgctx{}
)

type wgctx struct{}

// GetWaitgroup returns a waitgroup from the top-level service context
func GetWaitgroup(ctx context.Context) *sync.WaitGroup {
	wg, _ := ctx.Value(wgctxVal).(*sync.WaitGroup)
	if wg == nil {
		wg = &sync.WaitGroup{}
	}
	return wg
}

// Service represents a basic interface for a long-running service.  By invoking
// the Start function with a service, we automatically call Pre to initialize
// the service prior to starting (with a timeout), run the service via Run and listen
// for termination signals.  These term signals are caught and then the service is
// gracefully shut down via Stop.
type Service interface {
	// Name returns the service name
	Name() string
	// Pre initializes the service, returning an error if the service is not
	// capable of running.
	Pre(ctx context.Context) error
	// Run runs the service as a blocking operation, until the given context
	// is cancelled.
	Run(ctx context.Context) error
	// Stop is called to gracefully shut down the service.
	Stop(ctx context.Context) error
}

// StartTimeouter lets a Service define the timeout period when running Pre
type StartTimeouter interface {
	Service
	StartTimeout() time.Duration
}

// startTimeout returns the timeout duration used when starting the service.
// We attempt to typecast the service into a StartTimouter, returning the duration
// provided by this function or the defaultTimeout.
func startTimeout(s Service) time.Duration {
	if t, ok := s.(StartTimeouter); ok {
		return t.StartTimeout()
	}
	return defaultTimeout
}

// RunTimeouter lets a Service define how long the Run method can block for prior
// to starting cleanup.
type RunTimeouter interface {
	Service
	RunTimeout() time.Duration
}

// runTimeout returns the timeout duration used when an interrupt is received.
func runTimeout(s Service) time.Duration {
	if t, ok := s.(RunTimeouter); ok {
		return t.RunTimeout()
	}
	return defaultTimeout
}

// StopTimeouter lets a Service define the timeout period when running Pre
type StopTimeouter interface {
	Service
	StopTimeout() time.Duration
}

// stopTimeout returns the timeout duration used when starting the service.
// We attempt to typecast the service into a StopTimouter, returning the duration
// provided by this function or the defaultTimeout.
func stopTimeout(s Service) time.Duration {
	if t, ok := s.(StopTimeouter); ok {
		return t.StopTimeout()
	}
	return defaultTimeout
}

// StartAll starts all of the specified services, stopping all services when
// any of the group errors.
func StartAll(ctx context.Context, all ...Service) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	eg := &errgroup.Group{}
	for _, s := range all {
		svc := s
		eg.Go(func() error {
			err := Start(ctx, svc)
			// Close all other services.
			cancel()
			if err != nil && err != context.Canceled {
				return fmt.Errorf("service %s errored: %w", svc.Name(), err)
			}
			return nil
		})
	}
	return eg.Wait()
}

// Start runs a Service, invoking Pre() to bootstrap the Service, then Run()
// to run the Service.
//
// It blocks until an interrupt/kill signal, or the Run() command errors. We
// automatically call Stop() when terminating the Service.
func Start(ctx context.Context, s Service) (err error) {
	l := logger.From(ctx).With().Str("caller", s.Name()).Logger()
	ctx = logger.With(ctx, l)

	ctx, cleanup := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	defer cleanup()

	// Create a new parent waitgroup which can be used to prevent stopping
	// until the WG reaches 0.  This can be used for ephemeral goroutines.
	wg := &sync.WaitGroup{}
	ctx = context.WithValue(ctx, wgctxVal, wg)

	defer func() {
		if r := recover(); r != nil {
			l.Error().Interface("recover", r).Msg("service panicked")
		}
	}()

	if preErr := pre(ctx, s); preErr != nil {
		return preErr
	}

	if runErr := run(ctx, cleanup, s); runErr != nil && runErr != context.Canceled {
		logger.From(ctx).Error().Err(runErr).Msg("service run errored")
		err = errors.Join(err, runErr)
	}
	if stopErr := stop(ctx, s); stopErr != nil {
		logger.From(ctx).Error().Err(stopErr).Msg("service cleanup errored")
		err = errors.Join(err, stopErr)
	}

	return err
}

func pre(ctx context.Context, s Service) error {
	// Start the pre-run function with the timeout provided.
	preCh := make(chan error)
	go func() {
		// Run pre, and signal when complete.
		err := s.Pre(ctx)
		preCh <- err
	}()
	select {
	case <-time.After(startTimeout(s)):
		return ErrPreTimeout
	case err := <-preCh:
		close(preCh)
		if err != nil {
			return err
		}
	}
	return nil
}

// run calls the service's Run method, blocking until run completes or the
// parent context is cancelled.  If the parent context is cancelled, we wait
// up to 30 seconds
func run(ctx context.Context, stop func(), s Service) error {
	runErr := make(chan error)
	go func() {
		logger.From(ctx).Info().Msg("service starting")
		err := s.Run(ctx)
		// Communicate this error to the outer select.
		runErr <- err
	}()

	select {
	case err := <-runErr:
		// Run terminated.  Fetch the error from the goroutine.
		if err != nil {
			return err
		}
		logger.From(ctx).Info().Interface("signal", ctx.Err()).Msg("service finished")
	case <-ctx.Done():
		// We received a cancellation signal.  Allow Run to continue for up
		// to RunTimoeut period before quitting and cleaning up.

		// Ensure that we prevent the paretn context from capturing signals again.
		stop()
		// And set up a new context to quit if we receive the same signal again.
		repeat, cleanup := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
		defer cleanup()

		timeout := runTimeout(s)
		logger.From(ctx).Info().
			Interface("signal", ctx.Err()).
			Float64("seconds", timeout.Seconds()).
			Msg("signal received, service stopping")

		select {
		case <-repeat.Done():
			return fmt.Errorf("repeated signal received")
		case <-time.After(timeout):
			return ErrRunTimeout
		case err := <-runErr:
			return err
		}

	}
	return nil
}

func stop(ctx context.Context, s Service) error {
	stopCh := make(chan error)
	go func() {
		logger.From(ctx).Info().Msg("service cleaning up")
		// Create a new context that's not cancelled.
		if err := s.Stop(context.Background()); err != nil && err != context.Canceled {
			stopCh <- err
			return
		}
		// Wait for everything in the run waitgroup
		GetWaitgroup(ctx).Wait()
		stopCh <- nil
	}()

	select {
	case <-time.After(stopTimeout(s)):
		return fmt.Errorf("service did not clean up within timeout")
	case stopErr := <-stopCh:
		if stopErr != nil {
			return stopErr
		}
	}
	return nil
}
