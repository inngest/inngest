package memory

import (
	"context"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
)

// stats keeps metrics off the hot path.  every metrics call builds a tag map
// and an attribute set, which costs more than the constraint work itself.
// counters are added with atomics and flushed once a second as one metrics
// call per series.  histogram observations are appended to one of a few
// buffers and recorded by one goroutine every statsDrainEvery through a
// histogram prepared once per tag set, so each value still reaches the
// histogram, at most a second late, without building tags.  a producer never
// wakes the goroutine.  when a buffer is full the observation is dropped and
// counted in dropped.
type stats struct {
	shard string

	shards  [statsShards]eventShard
	dropped atomic.Int64

	// requested and granted are the untagged series every acquire touches.
	requested atomic.Int64
	granted   atomic.Int64

	// issued is indexed by the lease source enums, which are small.  a
	// source outside the table goes through counters.
	issued [issuedServices][issuedLocations][issuedModes]atomic.Int64

	// counters holds every other tagged series.
	mu       sync.RWMutex
	counters map[counterKey]*atomic.Int64

	// prepared histograms, used by the drain goroutine only
	latency     map[int]*metrics.PreparedHistogram
	retryAfter  map[constraintapi.LeaseSource]*metrics.PreparedHistogram
	semDuration map[string]*metrics.PreparedHistogram
	leaseAge    *metrics.PreparedHistogram
	prepareErr  bool
}

// eventShard is one observation buffer.  buf receives observations and alt is
// the buffer the drain hands back, so neither side allocates in steady state.
type eventShard struct {
	mu  sync.Mutex
	buf []metricEvent
	alt []metricEvent
	_   [8]byte
}

const (
	statsShards     = 64
	statsShardCap   = 4096
	statsDrainEvery = 100 * time.Millisecond
	statsFlushEvery = time.Second

	issuedServices  = 8
	issuedLocations = 8
	issuedModes     = 2
)

type histKind uint8

const (
	histRequestLatency histKind = iota
	histRetryAfter
	histSemaphoreDuration
	histLeaseAge
)

type metricEvent struct {
	kind    histKind
	value   time.Duration
	source  constraintapi.LeaseSource
	attempt int
	op      string
}

type counterName uint8

const (
	ctrRequested counterName = iota
	ctrGranted
	ctrLimiting
	ctrExhausted
	ctrIssued
	ctrSemaphoreOp
)

// counterKey identifies one counter series.  fn is set only when high
// cardinality instrumentation tags the series with the function ID.
type counterKey struct {
	name   counterName
	fn     uuid.UUID
	tag    string
	source constraintapi.LeaseSource
}

func newStats(shard string) *stats {
	return &stats{
		shard:       shard,
		counters:    map[counterKey]*atomic.Int64{},
		latency:     map[int]*metrics.PreparedHistogram{},
		retryAfter:  map[constraintapi.LeaseSource]*metrics.PreparedHistogram{},
		semDuration: map[string]*metrics.PreparedHistogram{},
	}
}

// record queues one histogram observation.  the shard is picked at random so
// concurrent callers rarely share a lock.
func (s *stats) record(ev metricEvent) {
	sh := &s.shards[rand.Uint32()%statsShards]
	sh.mu.Lock()
	if len(sh.buf) >= statsShardCap {
		sh.mu.Unlock()
		s.dropped.Add(1)
		return
	}
	sh.buf = append(sh.buf, ev)
	sh.mu.Unlock()
}

// counter returns the series for k, creating it on first use.
func (s *stats) counter(k counterKey) *atomic.Int64 {
	s.mu.RLock()
	c := s.counters[k]
	s.mu.RUnlock()
	if c != nil {
		return c
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if c = s.counters[k]; c == nil {
		c = &atomic.Int64{}
		s.counters[k] = c
	}
	return c
}

func inIssuedTable(src constraintapi.LeaseSource) bool {
	return src.Service >= 0 && int(src.Service) < issuedServices &&
		src.Location >= 0 && int(src.Location) < issuedLocations &&
		src.RunProcessingMode >= 0 && int(src.RunProcessingMode) < issuedModes
}

// acquired records the counters one Acquire produces.
func (s *stats) acquired(fn uuid.UUID, res *acquireResult, source constraintapi.LeaseSource) {
	if fn == uuid.Nil {
		s.requested.Add(int64(res.requested))
		s.granted.Add(int64(res.granted))
	} else {
		s.counter(counterKey{name: ctrRequested, fn: fn}).Add(int64(res.requested))
		s.counter(counterKey{name: ctrGranted, fn: fn}).Add(int64(res.granted))
	}
	for _, c := range res.limitingConstraints {
		s.counter(counterKey{name: ctrLimiting, fn: fn, tag: c.MetricsIdentifier()}).Add(1)
	}
	for _, c := range res.exhaustedConstraints {
		s.counter(counterKey{name: ctrExhausted, fn: fn, tag: c.MetricsIdentifier()}).Add(1)
	}
	if res.status == 3 {
		if inIssuedTable(source) {
			s.issued[source.Service][source.Location][source.RunProcessingMode].Add(int64(len(res.leases)))
		} else {
			s.counter(counterKey{name: ctrIssued, source: source}).Add(int64(len(res.leases)))
		}
	}
}

// semaphoreOp records one SemaphoreManager call.
func (s *stats) semaphoreOp(op string, start time.Time) {
	s.counter(counterKey{name: ctrSemaphoreOp, tag: op}).Add(1)
	s.record(metricEvent{kind: histSemaphoreDuration, op: op, value: time.Since(start)})
}

// run drains observations and flushes counters until stop closes.
func (s *stats) run(stop <-chan struct{}) {
	ctx := context.Background()
	ticker := time.NewTicker(statsDrainEvery)
	defer ticker.Stop()
	drains := 0
	for {
		select {
		case <-ticker.C:
			s.drain(ctx)
			drains++
			if drains%int(statsFlushEvery/statsDrainEvery) == 0 {
				s.flush(ctx)
			}
		case <-stop:
			s.drain(ctx)
			s.flush(ctx)
			return
		}
	}
}

// drain records every queued observation.
func (s *stats) drain(ctx context.Context) {
	for i := range s.shards {
		sh := &s.shards[i]
		sh.mu.Lock()
		events := sh.buf
		sh.buf = sh.alt[:0]
		sh.alt = events
		sh.mu.Unlock()
		for _, ev := range events {
			s.emit(ctx, ev)
		}
	}
	if n := s.dropped.Swap(0); n > 0 {
		logger.StdlibLogger(ctx).Warn("dropped constraint metric observations", "count", n, "shard", s.shard)
	}
}

func (s *stats) sourceTags(source constraintapi.LeaseSource) map[string]any {
	return map[string]any{
		"location":            source.Location.String(),
		"service":             source.Service.String(),
		"run_processing_mode": source.RunProcessingMode.String(),
		"shard":               s.shard,
	}
}

// prepared returns the histogram for one tag set, building it on first use.
// a build failure is logged once and the observation is dropped.
func (s *stats) prepared(ctx context.Context, cached *metrics.PreparedHistogram, build func() (*metrics.PreparedHistogram, error)) *metrics.PreparedHistogram {
	if cached != nil {
		return cached
	}
	p, err := build()
	if err != nil {
		if !s.prepareErr {
			s.prepareErr = true
			logger.StdlibLogger(ctx).Error("could not prepare constraint metric", "err", err, "shard", s.shard)
		}
		return nil
	}
	return p
}

func (s *stats) emit(ctx context.Context, ev metricEvent) {
	var p *metrics.PreparedHistogram
	switch ev.kind {
	case histRequestLatency:
		p = s.prepared(ctx, s.latency[ev.attempt], func() (*metrics.PreparedHistogram, error) {
			return metrics.PrepareConstraintAPIRequestLatency(ctx, metrics.HistogramOpt{
				PkgName: pkgName,
				Tags:    map[string]any{"operation": "acquire", "attempt": ev.attempt, "shard": s.shard},
			})
		})
		s.latency[ev.attempt] = p
	case histRetryAfter:
		p = s.prepared(ctx, s.retryAfter[ev.source], func() (*metrics.PreparedHistogram, error) {
			return metrics.PrepareConstraintAPIRetryAfterDuration(ctx, metrics.HistogramOpt{PkgName: pkgName, Tags: s.sourceTags(ev.source)})
		})
		s.retryAfter[ev.source] = p
	case histSemaphoreDuration:
		p = s.prepared(ctx, s.semDuration[ev.op], func() (*metrics.PreparedHistogram, error) {
			return metrics.PrepareConstraintAPISemaphoreDuration(ctx, metrics.HistogramOpt{
				PkgName: pkgName,
				Tags:    map[string]any{"operation": ev.op, "status": "success"},
			})
		})
		s.semDuration[ev.op] = p
	case histLeaseAge:
		p = s.prepared(ctx, s.leaseAge, func() (*metrics.PreparedHistogram, error) {
			return metrics.PrepareConstraintAPIScavengerLeaseAge(ctx, metrics.HistogramOpt{PkgName: pkgName, Tags: map[string]any{"shard": s.shard}})
		})
		s.leaseAge = p
	}
	if p != nil {
		p.RecordDuration(ctx, ev.value)
	}
}

// flush emits every non zero counter and resets it.
func (s *stats) flush(ctx context.Context) {
	if n := s.requested.Swap(0); n != 0 {
		metrics.IncrConstraintAPILeasesRequestedCounter(ctx, n, metrics.CounterOpt{PkgName: "constraintapi", Tags: map[string]any{"shard": s.shard}})
	}
	if n := s.granted.Swap(0); n != 0 {
		metrics.IncrConstraintAPILeasesGrantedCounter(ctx, n, metrics.CounterOpt{PkgName: "constraintapi", Tags: map[string]any{"shard": s.shard}})
	}

	for svc := range s.issued {
		for loc := range s.issued[svc] {
			for mode := range s.issued[svc][loc] {
				n := s.issued[svc][loc][mode].Swap(0)
				if n == 0 {
					continue
				}
				source := constraintapi.LeaseSource{
					Service:           constraintapi.LeaseService(svc),
					Location:          constraintapi.CallerLocation(loc),
					RunProcessingMode: constraintapi.RunProcessingMode(mode),
				}
				metrics.IncrConstraintAPIIssuedLeaseCounter(ctx, n, metrics.CounterOpt{PkgName: pkgName, Tags: s.sourceTags(source)})
			}
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	for key, c := range s.counters {
		n := c.Swap(0)
		if n == 0 {
			continue
		}
		tags := map[string]any{"shard": s.shard}
		if key.fn != uuid.Nil {
			tags["function_id"] = key.fn
		}
		switch key.name {
		case ctrRequested:
			metrics.IncrConstraintAPILeasesRequestedCounter(ctx, n, metrics.CounterOpt{PkgName: "constraintapi", Tags: tags})
		case ctrGranted:
			metrics.IncrConstraintAPILeasesGrantedCounter(ctx, n, metrics.CounterOpt{PkgName: "constraintapi", Tags: tags})
		case ctrLimiting:
			// the same series IncrConstraintAPILimitingConstraintsCounter
			// writes, added n at once
			tags["limiting_constraint"] = key.tag
			metrics.RecordCounterMetric(ctx, n, metrics.CounterOpt{
				PkgName:     "constraintapi",
				MetricName:  "constraintapi_limiting_constraints_total",
				Description: "Total number of times constraints limited capacity acquisition",
				Tags:        tags,
			})
		case ctrExhausted:
			tags["constraint"] = key.tag
			metrics.RecordCounterMetric(ctx, n, metrics.CounterOpt{
				PkgName:     "constraintapi",
				MetricName:  "constraintapi_exhausted_constraints_total",
				Description: "Total number of times constraints exhausted capacity acquisition",
				Tags:        tags,
			})
		case ctrIssued:
			metrics.IncrConstraintAPIIssuedLeaseCounter(ctx, n, metrics.CounterOpt{PkgName: pkgName, Tags: s.sourceTags(key.source)})
		case ctrSemaphoreOp:
			metrics.RecordCounterMetric(ctx, n, metrics.CounterOpt{
				PkgName:     pkgName,
				MetricName:  "constraintapi_semaphore_total",
				Description: "Total semaphore manager operations",
				Tags:        map[string]any{"operation": key.tag, "status": "success"},
			})
		}
	}
}
