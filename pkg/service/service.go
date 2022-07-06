package service

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/inngest/inngest-cli/pkg/logger"
)

var (
	defaultTimeout = 30 * time.Second

	ErrPreTimeout = fmt.Errorf("service did not pre-up within the given timeout")
)

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
	// Run runs the service as a blocking operation
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

// Start runs a Service, invoking Pre() to bootstrap the Service, then Run()
// to run the Service.
//
// It blocks until an interrupt/kill signal, or the Run() command errors. We
// automatically call Stop() when terminating the Service.
func Start(ctx context.Context, s Service) (err error) {
	l := logger.From(ctx).With().Str("service", s.Name()).Logger()

	preCh := make(chan error)
	preCtx, done := context.WithTimeout(ctx, startTimeout(s))
	defer done()

	go func() {
		err := s.Pre(preCtx)
		preCh <- err
	}()

	select {
	case <-preCtx.Done():
		return ErrPreTimeout
	case err = <-preCh:
		close(preCh)
		if err != nil {
			return err
		}
	}

	runCtx, cleanup := context.WithCancel(ctx)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	defer func() {
		if r := recover(); r != nil {
			l.Error().Interface("recover", r).Msg("service panicked")
			cleanup()
		}
	}()

	l.Info().Msg("service starting")
	go func() {
		err = s.Run(runCtx)
		// Call cleanup, triggering Stop below.  In this case
		// we don't need to wait for a signal to terminate.
		cleanup()
	}()

	select {
	case sig := <-sigs:
		// Terminating via a signal
		l.Info().Interface("signal", sig).Msg("received signal")
		cleanup()
	case <-runCtx.Done():
		// Run terminated.
		if err != nil {
			l.Error().Err(err).Msg("service errored")
		} else {
			l.Warn().Msg("service run finished")
		}
	}

	l.Info().Msg("service stopping")
	if stopErr := s.Stop(ctx); stopErr != nil {
		err = multierror.Append(err, stopErr)
	}

	return err
}
