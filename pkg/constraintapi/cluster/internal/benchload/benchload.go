// Package benchload runs a paced load from many callers for a fixed time and
// reports the request rate and the latency percentiles.  the benchmarks of
// the cluster package and of the tcp package use it.
package benchload

import (
	"context"
	"math/bits"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const (
	// warmFor is how long the load runs before the measured time starts.
	warmFor = 3 * time.Second
	// defaultSeconds is the measured time of one run.  the primary runs its
	// housekeeping every 5 seconds, so one run has three passes in it.
	defaultSeconds = 15
	// SecondsEnv names the environment variable that sets the measured time
	// of one run in seconds.
	SecondsEnv = "CLUSTER_BENCH_SECONDS"
	// window is the length of one reported part of a long run.
	window = time.Minute
	// requestsPerOp is the number of requests one Op sends.
	requestsPerOp = 2
)

// Config is one run.
type Config struct {
	// Rate is the target in requests per second.  one Op is two requests.
	// zero means every caller sends its next Op at once.
	Rate int
	// Callers is the number of goroutines that send.
	Callers int
	// Op sends acquire number k and its release.  it returns the latency of
	// the acquire call.
	Op func(ctx context.Context, k int64) (time.Duration, error)
}

// histBuckets is enough for latencies up to 2^40 nanoseconds, about 18
// minutes.
const (
	histSub     = 64
	histBuckets = histSub * 36
)

// hist counts durations in buckets.  a bucket is at most 1.6% wide, and a
// percentile is the upper edge of its bucket.
type hist struct {
	buckets [histBuckets]atomic.Int64
}

func (h *hist) add(d time.Duration) {
	v := uint64(max(d, 0))
	i := int(v)
	if v >= histSub {
		e := bits.Len64(v) - 7
		i = min(histSub*(e+1)+int(v>>e)-histSub, histBuckets-1)
	}
	h.buckets[i].Add(1)
}

// upper returns the largest duration bucket i counts.
func upper(i int) time.Duration {
	if i < histSub {
		return time.Duration(i)
	}
	e := i/histSub - 1
	m := uint64(histSub + i%histSub)
	return time.Duration((m+1)<<e - 1)
}

// summary is the merged count of some hists.
type summary struct {
	buckets [histBuckets]int64
	total   int64
}

func summarize(hs ...*hist) *summary {
	s := &summary{}
	for _, h := range hs {
		for i := range h.buckets {
			n := h.buckets[i].Load()
			s.buckets[i] += n
			s.total += n
		}
	}
	return s
}

// at returns the duration at quantile q in microseconds.
func (s *summary) at(q float64) float64 {
	if s.total == 0 {
		return 0
	}
	rank := int64(float64(s.total-1) * q)
	var seen int64
	for i, n := range s.buckets {
		seen += n
		if seen > rank {
			return float64(upper(i).Nanoseconds()) / 1e3
		}
	}
	return 0
}

// part is the load one window of the measured time sent.
type part struct {
	// ops counts the Ops sent in this window that got an answer.
	ops atomic.Int64
	// service is the latency of the acquire call.
	service hist
	// sched is the time from when the schedule wanted the acquire sent until
	// its answer.  it is the service latency plus the time the caller was
	// behind its schedule because its last Op was slow, so it also counts
	// the requests a stall delayed.
	sched hist
}

// Run sends the load of cfg for the warm up and the measured time, and
// reports req/s and the percentiles of the measured time.  the benchmark
// must run with -benchtime=1x.  a run longer than one minute also logs one
// line per minute.
func Run(b *testing.B, cfg Config) {
	b.Helper()
	measure := defaultSeconds * time.Second
	if v := os.Getenv(SecondsEnv); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			b.Fatalf("%s is %q, want a number of seconds", SecondsEnv, v)
		}
		measure = time.Duration(n) * time.Second
	}
	var period time.Duration
	if cfg.Rate > 0 {
		period = time.Duration(float64(time.Second) * requestsPerOp * float64(cfg.Callers) / float64(cfg.Rate))
	}
	parts := make([]part, (measure+window-1)/window)

	var (
		k      atomic.Int64
		failed atomic.Bool
		wg     sync.WaitGroup
	)
	ctx := context.Background()
	b.ResetTimer()
	begin := time.Now()
	from := begin.Add(warmFor)
	until := from.Add(measure)
	for i := range cfg.Callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// the callers start spread over one period.
			next := begin.Add(period * time.Duration(i) / time.Duration(cfg.Callers))
			for {
				// a caller that sleeps is on time.  what the sleep adds is not
				// counted, because it comes from this process and not from the
				// cluster.
				onTime := period == 0
				if d := time.Until(next); d > 0 {
					time.Sleep(d)
					onTime = true
				}
				now := time.Now()
				if !now.Before(until) {
					return
				}
				var late time.Duration
				if !onTime {
					late = now.Sub(next)
				}
				next = next.Add(period)
				took, err := cfg.Op(ctx, k.Add(1))
				if err != nil {
					// one message is enough.  every caller fails the same way.
					if failed.CompareAndSwap(false, true) {
						b.Error(err)
					}
					continue
				}
				if now.Before(from) {
					continue
				}
				p := &parts[now.Sub(from)/window]
				p.ops.Add(1)
				p.service.add(took)
				p.sched.add(late + took)
			}
		}()
	}
	wg.Wait()
	b.StopTimer()

	var ops int64
	var service, sched []*hist
	for i := range parts {
		p := &parts[i]
		ops += p.ops.Load()
		service = append(service, &p.service)
		sched = append(sched, &p.sched)
		if len(parts) == 1 {
			break
		}
		length := min(window, measure-time.Duration(i)*window)
		sv, sc := summarize(&p.service), summarize(&p.sched)
		b.Logf("minute %d: %.0f req/s, p50 %.0f µs, p99 %.0f µs, p99.9 %.0f µs, max %.0f µs, from the schedule p99 %.0f µs, p99.9 %.0f µs",
			i+1, float64(requestsPerOp*p.ops.Load())/length.Seconds(), sv.at(0.5), sv.at(0.99), sv.at(0.999), sv.at(1), sc.at(0.99), sc.at(0.999))
	}
	sv := summarize(service...)
	b.ReportMetric(0, "ns/op")
	b.ReportMetric(float64(requestsPerOp*ops)/measure.Seconds(), "req/s")
	b.ReportMetric(sv.at(0.5), "p50-µs")
	b.ReportMetric(sv.at(0.99), "p99-µs")
	b.ReportMetric(sv.at(0.999), "p999-µs")
	b.ReportMetric(sv.at(1), "max-µs")
	if period > 0 {
		sc := summarize(sched...)
		b.ReportMetric(sc.at(0.99), "sched-p99-µs")
		b.ReportMetric(sc.at(0.999), "sched-p999-µs")
	}
}
