package constraintapi

import (
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/require"
)

// FuzzGCRA runs random request sequences through the Lua helper in miniredis
// and the Go port and demands identical results at every step.
//
// the inputs are shaped so every value is an integer on both sides: the
// period is limit times k, so the emission interval is a whole unit, and the
// clock starts at the Unix epoch, so a TAT stays within the 14 significant
// digits Lua keeps when it writes a number to Redis.  fractional emissions are
// covered by the table cases, which compare truncated values.
//
//	go test ./pkg/constraintapi/ -run '^$' -fuzz FuzzGCRA -fuzztime 30s
func FuzzGCRA(f *testing.F) {
	f.Add(false, uint16(9), uint32(5_999), uint16(0), []byte{0, 1, 0, 1, 3, 0, 8, 2, 0, 5})
	f.Add(true, uint16(119), uint32(999_999), uint16(11), []byte{0, 1, 1, 3, 40, 0, 2, 2, 63, 1})
	f.Add(false, uint16(0), uint32(0), uint16(0), []byte{0, 1, 0, 1, 0, 1, 9, 0})
	f.Add(true, uint16(2_999), uint32(7), uint16(2_999), []byte{5, 5, 5, 5, 5, 5, 60, 0, 60, 0})

	f.Fuzz(func(t *testing.T, rateLimit bool, limitX uint16, kX uint32, burstX uint16, ops []byte) {
		if len(ops) < 2 || len(ops) > 128 {
			t.Skip()
		}
		limit := 1 + int(limitX)%1_000
		k := 1 + int64(kX)%1_000_000
		burst := int(burstX) % (limit + 1)

		kind := gcraThrottle
		unit := time.Millisecond
		if rateLimit {
			kind = gcraRateLimit
			unit = time.Nanosecond
		}
		period := time.Duration(int64(limit)*k) * unit

		lua := newLuaGCRARunner(t, kind)
		goR := &goGCRARunner{kind: kind, states: map[string]goGCRAState{}}
		clock := clockwork.NewFakeClockAt(time.Unix(0, 0))
		lua.advance(clock.Now(), 0)

		for i := 0; i+1 < len(ops); i += 2 {
			// up to eight emission intervals between requests
			if adv := time.Duration(int64(ops[i]%64)*k/8) * unit; adv > 0 {
				clock.Advance(adv)
				lua.advance(clock.Now(), adv)
			}
			opts := gcraScriptOptions{
				key:      "fuzz",
				now:      clock.Now(),
				period:   period,
				limit:    limit,
				burst:    burst,
				quantity: int(ops[i+1] % 6),
			}
			want := lua.run(t, opts)
			got := goR.run(t, opts)
			require.Equal(t, want, got, "op %d: limit %d burst %d k %d quantity %d", i/2, limit, burst, k, opts.quantity)
		}
	})
}
