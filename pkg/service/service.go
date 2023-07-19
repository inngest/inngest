package service

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/inngest/inngest/pkg/logger"
	"golang.org/x/sync/errgroup"
)

var (
	defaultTimeout = 30 * time.Second

	ErrPreTimeout = fmt.Errorf("service did not pre-up within the given timeout")

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
	case err = <-preCh:
		close(preCh)
		if err != nil {
			return err
		}
	}

	// Listen for signals straight after running pre.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	runCtx, cleanup := context.WithCancel(ctx)
	defer func() {
		if r := recover(); r != nil {
			l.Error().Interface("recover", r).Msg("service panicked")
			cleanup()
		}
	}()

	// Create a new parent waitgroup which can be used to prevent stopping
	// until the WG reaches 0.  This can be used for ephemeral goroutines.
	wg := &sync.WaitGroup{}
	runCtx = context.WithValue(runCtx, wgctxVal, wg)

	runErr := make(chan error)
	l.Info().Msg("service starting")
	go func() {
		err := s.Run(runCtx)
		// Communicate this error to the outer select.
		runErr <- err
		// Call cleanup, triggering Stop below.  In this case
		// we don't need to wait for a signal to terminate.
		cleanup()
	}()

	select {
	case sig := <-sigs:
		// Terminating via a signal
		l.Info().Interface("signal", sig).Msg("received signal")
		cleanup()
	case err = <-runErr:
		// Run terminated.  Fetch the error from the goroutine.
		if err != nil {
			l.Error().Err(err).Msg("service errored")
		} else {
			l.Warn().Msg("service run stopped")
		}
	case <-runCtx.Done():
		l.Warn().Msg("service run stopped")
	}

	stopCh := make(chan error)
	go func() {
		l.Info().Msg("service cleaning up")
		// Create a new context that's not cancelled.
		if err := s.Stop(context.Background()); err != nil && err != context.Canceled {
			stopCh <- err
			return
		}

		// Wait for everything in the run waitgroup
		wg.Wait()

		stopCh <- nil
	}()
	select {
	case <-time.After(stopTimeout(s)):
		l.Error().Msg("service did not clean up within timeout")
		return err
	case stopErr := <-stopCh:
		if stopErr != nil {
			err = multierror.Append(err, stopErr)
		}
	}

	if err == context.Canceled {
		// Ignore plz.
		return nil
	}

	return err
}
