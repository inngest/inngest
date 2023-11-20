package backoff

import (
	"time"

	"math/rand"
)

var (
	BackoffTable = []time.Duration{
		15 * time.Second,
		30 * time.Second,
		time.Minute,
		2 * time.Minute,
		5 * time.Minute,
		10 * time.Minute,
		20 * time.Minute,
		40 * time.Minute,
		time.Hour,
		2 * time.Hour,
	}

	backoffLen = len(BackoffTable) - 1

	DefaultBackoff BackoffFunc = TableBackoff
)

type BackoffFunc func(attemptNum int) time.Time

func ExponentialJitterBackoff(attemptNum int) time.Time {
	backoff := float64(uint(1) << (uint(attemptNum) - 1))
	backoff += backoff * (0.15 * rand.Float64())
	// Increase by a factor of 10 to get 10 second breaks at minimum.
	backoff = backoff * 10

	dur := time.Second * time.Duration(backoff)

	// Max this out at 12 hours
	if dur >= time.Hour*12 {
		jitter := time.Duration(rand.Int31n(120)) * time.Second
		return time.Now().Add(12 * time.Hour).Add(jitter)
	}

	return time.Now().Add(dur)
}

// TableBackoff returns a fixed backoff maxing out at 2 hours, with up to
// 30 seconds of jitter.
func TableBackoff(attemptNum int) time.Time {
	if attemptNum > backoffLen {
		attemptNum = backoffLen
	}
	at := BackoffTable[attemptNum]
	// Add between 0 and 30 seconds randomly for jitter.
	jitter := time.Duration(rand.Int31n(30_000)) * time.Millisecond
	return time.Now().Add(at).Add(jitter)
}

// GetLinearBackoffFunc returns a backoff function that returns a fixed interval
// between attempts.
func GetLinearBackoffFunc(interval time.Duration) BackoffFunc {
	return func(attemptNum int) time.Time {
		return time.Now().Add(interval)
	}
}
