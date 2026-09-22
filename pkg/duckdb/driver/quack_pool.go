package driver

import (
	"context"
	"errors"

	"github.com/inngest/inngest/pkg/logger"
)

// errQuackUnavailable is returned when no quack listener is published and the
// process isn't disabled — in practice, after Connector.Close.
var errQuackUnavailable = errors.New("duckdb: quack listener is not running")

// quackEndpoint is one bootstrapped quack listener. gen identifies which
// subprocess spawn it belongs to (process.quackGen at bootstrap time).
type quackEndpoint struct {
	listenURL string
	token     string
	gen       uint64
}

// pooledQuackConn is the sqlExecer behind every connection beyond the first
// when Options.QuackConns > 1. It owns at most one quackSession, tagged with
// the generation of the listener it was opened against, and re-handshakes
// whenever that generation is no longer the live one — so a subprocess
// restart never leaves it talking to a dead listener with stale
// credentials.
//
// database/sql never uses one driver.Conn from two goroutines at once, so
// its fields need no locking of their own.
type pooledQuackConn struct {
	p    *process
	sess *quackSession
	gen  uint64
}

func (c *pooledQuackConn) exec(ctx context.Context, sqlText string) (cols []string, rows []row, err error) {
	err = c.run(ctx, func(s *quackSession) error {
		var e error
		cols, rows, e = s.exec(ctx, sqlText)
		return e
	})
	return cols, rows, err
}

func (c *pooledQuackConn) query(ctx context.Context, sqlText string) (cols []string, types []string, rows []row, err error) {
	err = c.run(ctx, func(s *quackSession) error {
		var e error
		cols, types, rows, e = s.query(ctx, sqlText)
		return e
	})
	return cols, types, rows, err
}

// run executes fn on a current session, applying the same failure policy as
// process.runWithRestartLocked: statement errors and caller cancellation are
// surfaced as-is; any other failure restarts the subprocess (at most once
// per generation, across every connection) and retries fn once.
func (c *pooledQuackConn) run(ctx context.Context, fn func(*quackSession) error) error {
	gen, err := c.attempt(ctx, fn)
	if err == nil || !c.shouldRecover(ctx, err) {
		return err
	}

	logger.StdlibLogger(ctx).Warn("duckdb: pooled quack connection failed; recovering",
		"transport", "quack", "generation", gen, "error", err)
	c.sess = nil
	if rerr := c.p.recoverQuack(ctx, gen, err); rerr != nil {
		return rerr
	}
	if ctx.Err() != nil {
		return err
	}
	_, err = c.attempt(ctx, fn)
	return err
}

// attempt runs fn once, first re-handshaking if the live listener's
// generation differs from the session's. It returns the generation fn ran
// against (or tried to handshake with) for recoverQuack.
func (c *pooledQuackConn) attempt(ctx context.Context, fn func(*quackSession) error) (uint64, error) {
	ep, err := c.p.currentQuackEndpoint()
	if err != nil {
		return 0, err
	}
	if c.sess == nil || c.gen != ep.gen {
		sess, err := newQuackSession(ctx, ep.listenURL, ep.token)
		if err != nil {
			c.sess = nil
			return ep.gen, err
		}
		if c.sess != nil {
			logger.StdlibLogger(ctx).Debug("duckdb: pooled quack session re-handshaked after subprocess restart",
				"transport", "quack", "stale_generation", c.gen, "generation", ep.gen, "connection_id", sess.connectionID)
		}
		c.sess, c.gen = sess, ep.gen
	}
	return c.gen, fn(c.sess)
}

// currentSession returns a session against the live listener,
// re-handshaking if needed, for callers (the quack appenders) that drive
// protocol messages sqlExecer doesn't cover.
func (c *pooledQuackConn) currentSession(ctx context.Context) (*quackSession, error) {
	if _, err := c.attempt(ctx, func(*quackSession) error { return nil }); err != nil {
		return nil, err
	}
	return c.sess, nil
}

func (c *pooledQuackConn) shouldRecover(ctx context.Context, err error) bool {
	switch {
	case errors.Is(err, errStatementFailed),
		errors.Is(err, ErrDisabled),
		errors.Is(err, errQuackUnavailable):
		return false
	}
	// Same reasoning as runWithRestartLocked: a quack request aborted by the
	// caller's own ctx leaves nothing behind that a restart would fix.
	return ctx.Err() == nil
}
