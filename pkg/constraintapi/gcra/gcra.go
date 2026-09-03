// Package gcra is the Go port of lua/helper/gcra.lua.  it has no dependency
// on constraintapi so the Lua test tables there can run against both.
//
// all arithmetic is float64, like Lua numbers.  a stored TAT is a float64 as
// well, so a limit that does not divide the period keeps its fraction.
package gcra

import "math"

// Result mirrors the table returned by the Lua rateLimit and throttle helpers.
// times are in the unit of the call, nanoseconds for RateLimit and
// milliseconds for Throttle.
type Result struct {
	Limited bool

	// Limit is burst + 1, the number of requests admitted at once from an
	// empty state.
	Limit int

	// Remaining is how many requests fit right now.  0 on the limited path.
	Remaining int

	// Usage is the used token count without burst.  negative when the stored
	// TAT is in the past.
	Usage int64

	ResetAfter       float64
	RetryAfter       float64
	RetryAt          float64
	EmissionInterval float64
	DVT              float64
	TAT              float64
	NewTAT           float64
	Increment        float64
	AllowAt          float64
	Diff             float64
	Next             float64
}

// Throttle evaluates one GCRA step in milliseconds.  quantity 0 simulates a
// request of 1 and never stores.  store is true when the caller must write
// newTAT with ttlSeconds, which happens only when quantity > 0 and the request
// was admitted.
func Throttle(tat float64, present bool, nowMS, periodMS float64, limit, burst, quantity int) (res Result, newTAT float64, ttlSeconds int64, store bool) {
	return step(tat, present, nowMS, periodMS, limit, burst, quantity, 1_000)
}

// RateLimit evaluates one GCRA step in nanoseconds.  see Throttle.
func RateLimit(tat float64, present bool, nowNS, periodNS float64, limit, burst, quantity int) (res Result, newTAT float64, ttlSeconds int64, store bool) {
	return step(tat, present, nowNS, periodNS, limit, burst, quantity, 1_000_000_000)
}

// step is the Lua helper line for line.  perSecond converts the call's time
// unit to the seconds of the stored TTL.
func step(tat float64, present bool, now, period float64, limit, burst, quantity int, perSecond float64) (res Result, newTAT float64, ttlSeconds int64, store bool) {
	if limit < 1 {
		limit = 1
	}

	res.Limit = burst + 1

	emission := period / float64(limit)
	res.EmissionInterval = emission

	// retry_at assumes all remaining capacity is consumed
	res.RetryAt = now + emission

	dvt := emission * float64(burst+1)
	res.DVT = dvt

	if !present {
		tat = now
	}
	res.TAT = tat

	// quantity 0 simulates a request of 1 to report retry and remaining
	origQuantity := quantity
	if quantity == 0 {
		quantity = 1
	}

	increment := float64(quantity) * emission
	res.Increment = increment

	newTAT = tat + increment
	if now > tat {
		newTAT = now + increment
	}
	res.NewTAT = newTAT

	ttl := tat - now
	res.ResetAfter = ttl

	res.Usage = int64(math.Min(math.Ceil(ttl/emission), float64(limit)))

	allowAt := newTAT - dvt
	res.AllowAt = allowAt

	diff := now - allowAt
	res.Diff = diff

	if diff < 0 {
		if increment <= dvt {
			res.RetryAfter = -diff
			res.RetryAt = now - diff
		}
		if origQuantity > 0 {
			res.Next = dvt - ttl
			res.Remaining = 0
			res.Limited = true
			return res, newTAT, 0, false
		}
	}

	if origQuantity > 0 {
		ttl = newTAT - now
		res.ResetAfter = ttl
		res.Usage = int64(math.Min(math.Ceil(ttl/emission), float64(limit)))
		// string.format("%d", ...) in Lua 5.1 truncates toward zero
		ttlSeconds = int64(math.Max(ttl/perSecond, 1))
		store = true
	}

	next := dvt - ttl
	res.Next = next

	if next > -emission {
		res.Remaining = int(math.Floor(next / emission))
	}

	return res, newTAT, ttlSeconds, store
}
