package server

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

	ErrPreTimeout = fmt.Errorf("server did not pre-up within the given timeout")
)

// Server represents a basic interface for a long-running server.  By invoking
// the Start function with a server, we automatically call Pre to initialize
// the server prior to starting (with a timeout), run the server via Run and listen
// for termination signals.  These term signals are caught and then the server is
// gracefully shut down via Stop.
type Server interface {
	// Name returns the server name
	Name() string

	// Pre initialize the server, returning an error if the server is not
	// capable of running.
	Pre(ctx context.Context) error
	// Run runs the server as a blocking operation
	Run(ctx context.Context) error
	// Stop is called to gracefully shut down the server.
	Stop(ctx context.Context) error
}

// StartTimeouter lets a Server define the timeout period when running Pre
type StartTimeouter interface {
	StartTimeout() time.Duration
}

// startTimeout returns the timeout duration used when starting the server.
// We attempt to typecast the server into a StartTimouter, returning the duration
// provided by this function or the defaultTimeout.
func startTimeout(s Server) time.Duration {
	if t, ok := s.(StartTimeouter); ok {
		return t.StartTimeout()
	}
	return defaultTimeout
}

// Start runs a server, invoking Pre() to bootstrap the server, then Run()
// to run the server.
//
// It blocks until an interrupt/kill signal, or the Run() command errors. We
// automatically call Stop() when terminating the server.
func Start(ctx context.Context, s Server) (err error) {
	l := logger.From(ctx).With().Str("server", s.Name()).Logger()

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

	ctx, cleanup := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	defer func() {
		if r := recover(); r != nil {
			l.Error().Interface("recover", r).Msg("server panicked")
			cleanup()
		}
	}()

	l.Info().Msg("server starting")
	go func() {
		err = s.Run(ctx)
		// Call cleanup, triggering Stop below.  In this case
		// we don't need to wait for a signal to terminate.
		cleanup()
	}()

	select {
	case sig := <-sigs:
		// Terminating via a signal
		l.Info().Interface("signal", sig).Msg("received signal")
		cleanup()
	case <-ctx.Done():
		// Run terminated.
		if err != nil {
			l.Error().Err(err).Msg("server errored")
		} else {
			l.Warn().Msg("server run finished")
		}
	}

	l.Info().Msg("server stopping")
	if stopErr := s.Stop(ctx); stopErr != nil {
		err = multierror.Append(err, stopErr)
	}

	return err
}
