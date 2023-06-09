// Package storetest provides a helper for testing throttled stores.
package storetest // import "github.com/throttled/throttled/v2/store/storetest"

import (
	"context"
	"math/rand"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/throttled/throttled/v2"
)

// TestGCRAStoreCtx tests the behavior of a GCRAStore implementation for
// compliance with the throttled API. It does not require support
// for TTLs.
func TestGCRAStoreCtx(t *testing.T, st throttled.GCRAStoreCtx) {
	ctx := context.Background()

	// GetWithTime a missing key
	if have, _, err := st.GetWithTime(ctx, "foo"); err != nil {
		t.Fatal(err)
	} else if have != -1 {
		t.Errorf("expected GetWithTime to return -1 for a missing key but got %d", have)
	}

	// SetIfNotExists on a new key
	want := int64(1)

	if set, err := st.SetIfNotExistsWithTTL(ctx, "foo", want, 0); err != nil {
		t.Fatal(err)
	} else if !set {
		t.Errorf("expected SetIfNotExists on an empty key to succeed")
	}

	before := time.Now()

	if have, now, err := st.GetWithTime(ctx, "foo"); err != nil {
		t.Fatal(err)
	} else if have != want {
		t.Errorf("expected GetWithTime to return %d but got %d", want, have)
	} else if now.UnixNano() <= 0 {
		t.Errorf("expected GetWithTime to return a time representable representable as a positive int64 of nanoseconds since the epoch")
	} else if now.Before(before) || now.After(time.Now()) {
		// Note that we make the assumption here that the store is running on
		// the same machine as this test and thus shares a clock. This can be a
		// little tricky in the case of Redis, which could be running
		// elsewhere. The test assumes that it's running either locally on on
		// Travis (where currently the Redis is available on localhost). If new
		// test environments are procured, this may need to be revisited.
		t.Errorf("expected GetWithTime to return a time between the time before the call and the time after the call")
	}

	// SetIfNotExists on an existing key
	if set, err := st.SetIfNotExistsWithTTL(ctx, "foo", 123, 0); err != nil {
		t.Fatal(err)
	} else if set {
		t.Errorf("expected SetIfNotExists on an existing key to fail")
	}

	if have, _, err := st.GetWithTime(ctx, "foo"); err != nil {
		t.Fatal(err)
	} else if have != want {
		t.Errorf("expected GetWithTime to return %d but got %d", want, have)
	}

	// SetIfNotExists on a different key
	if set, err := st.SetIfNotExistsWithTTL(ctx, "bar", 456, 0); err != nil {
		t.Fatal(err)
	} else if !set {
		t.Errorf("expected SetIfNotExists on an empty key to succeed")
	}

	// Returns the false on a missing key
	if swapped, err := st.CompareAndSwapWithTTL(ctx, "baz", 1, 2, 0); err != nil {
		t.Fatal(err)
	} else if swapped {
		t.Errorf("expected CompareAndSwap to fail on a missing key")
	}

	// Test a successful CAS
	want = int64(2)

	if swapped, err := st.CompareAndSwapWithTTL(ctx, "foo", 1, want, 0); err != nil {
		t.Fatal(err)
	} else if !swapped {
		t.Errorf("expected CompareAndSwap to succeed")
	}

	if have, _, err := st.GetWithTime(ctx, "foo"); err != nil {
		t.Fatal(err)
	} else if have != want {
		t.Errorf("expected GetWithTime to return %d but got %d", want, have)
	}

	// Test an unsuccessful CAS
	if swapped, err := st.CompareAndSwapWithTTL(ctx, "foo", 1, 2, 0); err != nil {
		t.Fatal(err)
	} else if swapped {
		t.Errorf("expected CompareAndSwap to fail")
	}

	if have, _, err := st.GetWithTime(ctx, "foo"); err != nil {
		t.Fatal(err)
	} else if have != want {
		t.Errorf("expected GetWithTime to return %d but got %d", want, have)
	}
}

// TestGCRAStoreTTLCtx tests the behavior of TTLs in a GCRAStore implementation.
func TestGCRAStoreTTLCtx(t *testing.T, st throttled.GCRAStoreCtx) {
	ttl := time.Second
	want := int64(1)
	key := "ttl"
	ctx := context.Background()

	if _, err := st.SetIfNotExistsWithTTL(ctx, key, want, ttl); err != nil {
		t.Fatal(err)
	}

	if have, _, err := st.GetWithTime(ctx, key); err != nil {
		t.Fatal(err)
	} else if have != want {
		t.Errorf("expected GetWithTime to return %d, got %d", want, have)
	}

	// I can't think of a generic way to test expiration without a sleep
	time.Sleep(ttl + time.Millisecond)

	if have, _, err := st.GetWithTime(ctx, key); err != nil {
		t.Fatal(err)
	} else if have != -1 {
		t.Errorf("expected GetWithTime to fail on an expired key but got %d", have)
	}
}

// BenchmarkGCRAStoreCtx runs parallel benchmarks against a GCRAStore implementation.
// Aside from being useful for performance testing, this is useful for finding
// race conditions with the Go race detector.
func BenchmarkGCRAStoreCtx(b *testing.B, st throttled.GCRAStoreCtx) {
	seed := int64(42)
	var attempts, updates int64

	b.RunParallel(func(pb *testing.PB) {
		// We need atomic behavior around the RNG or go detects a race in the test
		delta := int64(1)
		seedValue := atomic.AddInt64(&seed, delta) - delta
		gen := rand.New(rand.NewSource(seedValue))

		for pb.Next() {
			ctx := context.Background()
			key := strconv.FormatInt(gen.Int63n(50), 10)

			var v int64
			var updated bool

			v, _, err := st.GetWithTime(ctx, key)
			if v == -1 {
				updated, err = st.SetIfNotExistsWithTTL(ctx, key, gen.Int63(), 0)
				if err != nil {
					b.Error(err)
				}
			} else if err != nil {
				b.Error(err)
			} else {
				updated, err = st.CompareAndSwapWithTTL(ctx, key, v, gen.Int63(), 0)
				if err != nil {
					b.Error(err)
				}
			}

			atomic.AddInt64(&attempts, 1)
			if updated {
				atomic.AddInt64(&updates, 1)
			}
		}
	})

	b.Logf("%d/%d update operations succeeed", updates, attempts)
}
