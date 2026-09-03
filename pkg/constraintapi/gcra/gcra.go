// Package gcra is the Go port of lua/helper/gcra.lua.  it has no dependency
// on constraintapi so the Lua test tables there can run against both.
//
// all arithmetic is float64, like Lua numbers.  a stored TAT is a float64 as
// well, so a limit that does not divide the period keeps its fraction.
package gcra

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
	return Result{}, 0, 0, false
}

// RateLimit evaluates one GCRA step in nanoseconds.  see Throttle.
func RateLimit(tat float64, present bool, nowNS, periodNS float64, limit, burst, quantity int) (res Result, newTAT float64, ttlSeconds int64, store bool) {
	return Result{}, 0, 0, false
}
