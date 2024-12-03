package expr

import "github.com/sourcegraph/conc/pool"

type errPoolOpts struct {
	concurrency int64
	firstErr    bool
}

func newErrPool(opts errPoolOpts) *pool.ErrorPool {
	p := pool.New().WithErrors().WithMaxGoroutines(int(opts.concurrency))

	if opts.firstErr {
		p = p.WithFirstError()
	}

	return p
}
