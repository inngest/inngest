package driver

import (
	"context"
	"errors"
	"time"

	"github.com/inngest/inngest/pkg/duckdb/driver/internal/result"
	"github.com/inngest/inngest/pkg/duckdb/worker"
	"github.com/inngest/inngest/pkg/logger"
)

// errQuackUnavailable is returned when no worker endpoint is published and the
// process isn't disabled — in practice, after Connector.Close.
var errQuackUnavailable = errors.New("duckdb: worker is not running")

// workerEndpoint is one bootstrapped worker client. gen identifies which
// worker spawn it belongs to (process.quackGen at bootstrap time).
type workerEndpoint struct {
	client *worker.Client
	gen    uint64
}

// pooledQuackConn is the sqlExecer behind every concurrent worker session
// beyond the primary one (Options.QuackConns > 1 and Connector.OpenShared).
// It owns at most one *worker.Session, tagged with the generation of the
// worker it was opened against, and transparently re-opens whenever that
// generation is no longer the live one.
type pooledQuackConn struct {
	p     *process
	sess  *worker.Session
	exec  *workerSessionExecer
	gen   uint64
	gated bool
}

func (c *pooledQuackConn) Exec(ctx context.Context, sqlText string) (cols []string, rows []result.Row, err error) {
	err = c.run(ctx, func(s *workerSessionExecer) error {
		var e error
		cols, rows, e = s.Exec(ctx, sqlText)
		return e
	})
	return cols, rows, err
}

func (c *pooledQuackConn) Query(ctx context.Context, sqlText string) (cols []string, types []string, rows []result.Row, err error) {
	err = c.run(ctx, func(s *workerSessionExecer) error {
		var e error
		cols, types, rows, e = s.Query(ctx, sqlText)
		return e
	})
	return cols, types, rows, err
}

func (c *pooledQuackConn) QueryWithStreams(ctx context.Context, sqlText string, streams []worker.SendStream) (cols []string, types []string, rows []result.Row, err error) {
	err = c.run(ctx, onceOnly(func(s *workerSessionExecer) error {
		var e error
		cols, types, rows, e = s.QueryWithStreams(ctx, sqlText, streams)
		return e
	}))
	return cols, types, rows, err
}

func (c *pooledQuackConn) run(ctx context.Context, fn func(*workerSessionExecer) error) error {
	gen, err := c.attempt(ctx, fn)
	if err == nil || !c.shouldRecover(ctx, err) {
		return err
	}

	logger.StdlibLogger(ctx).Warn("duckdb: pooled worker connection failed; recovering",
		"transport", "worker", "generation", gen, "error", err)
	c.sess = nil
	c.exec = nil
	if rerr := c.p.recoverQuack(ctx, gen, err); rerr != nil {
		return rerr
	}
	if ctx.Err() != nil {
		return err
	}
	_, err = c.attempt(ctx, fn)
	return err
}

func (c *pooledQuackConn) attempt(ctx context.Context, fn func(*workerSessionExecer) error) (uint64, error) {
	ep, err := c.p.currentWorkerEndpoint(ctx)
	if err != nil {
		return 0, err
	}
	if c.sess == nil || c.gen != ep.gen || c.sess.Abandoned() {
		if c.gated && (c.sess == nil || c.sess.Abandoned()) && c.p.tooManyAbandoned() {
			return ep.gen, ErrTooManyAbandoned
		}
		sess, err := ep.client.OpenSession(ctx, worker.WithAbandonedCounter(&c.p.abandoned))
		if err != nil {
			c.sess = nil
			c.exec = nil
			return ep.gen, err
		}
		switch {
		case c.sess == nil:
		case c.sess.Abandoned():
			logger.StdlibLogger(ctx).Debug("duckdb: pooled worker session reopened after an abandoned statement",
				"transport", "worker", "abandoned_session_id", c.sess.ID(), "session_id", sess.ID())
		default:
			logger.StdlibLogger(ctx).Debug("duckdb: pooled worker session reopened after subprocess restart",
				"transport", "worker", "stale_generation", c.gen, "generation", ep.gen, "session_id", sess.ID())
		}
		c.sess = sess
		c.exec = &workerSessionExecer{sess: sess}
		c.gen = ep.gen
	}
	return c.gen, fn(c.exec)
}

func (c *pooledQuackConn) Close() error {
	if c.sess == nil {
		return nil
	}
	sess := c.sess
	c.sess = nil
	c.exec = nil
	if ep := c.p.workerEP.Load(); ep == nil || ep.gen != c.gen {
		return nil
	}
	closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sess.Close(closeCtx); err != nil {
		logger.StdlibLogger(context.Background()).Debug("duckdb: failed to disconnect a closed pooled worker connection",
			"transport", "worker", "session_id", sess.ID(), "error", err)
	}
	return nil
}

func (c *pooledQuackConn) currentSession(ctx context.Context) (*worker.Session, error) {
	var sess *worker.Session
	if err := c.run(ctx, func(s *workerSessionExecer) error {
		sess = s.sess
		return nil
	}); err != nil {
		return nil, err
	}
	return sess, nil
}

func (c *pooledQuackConn) shouldRecover(ctx context.Context, err error) bool {
	switch {
	case errors.Is(err, result.ErrStatementFailed),
		errors.Is(err, worker.ErrStreamSource),
		errors.Is(err, ErrDisabled),
		errors.Is(err, errQuackUnavailable),
		errors.Is(err, ErrTooManyAbandoned):
		return false
	}
	return ctx.Err() == nil
}

func (p *process) tooManyAbandoned() bool {
	limit := p.maxAbandoned
	if limit <= 0 {
		limit = DefaultMaxAbandonedStatements
	}
	return p.abandoned.Load() >= int64(limit)
}
