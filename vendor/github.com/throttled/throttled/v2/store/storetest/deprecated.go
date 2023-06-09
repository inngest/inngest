package storetest

import (
	"github.com/throttled/throttled/v2"
	"testing"
)

// TestGCRAStore provides an adapter for TestGCRAStoreCtx
//
// Deprecated: implement GCRAStoreCtx and use TestGCRAStoreCtx instead.
func TestGCRAStore(t *testing.T, st throttled.GCRAStore) {
	TestGCRAStoreCtx(t, throttled.WrapStoreWithContext(st))
}

// TestGCRAStoreTTL provides an adapter for TestGCRAStoreTTLCtx
//
// Deprecated: implement GCRAStoreCtx and use TestGCRAStoreTTLCtx instead.
func TestGCRAStoreTTL(t *testing.T, st throttled.GCRAStore) {
	TestGCRAStoreTTLCtx(t, throttled.WrapStoreWithContext(st))
}

// BenchmarkGCRAStore provides an adapter for BenchmarkGCRAStoreCtx
//
// Deprecated: implement GCRAStoreCtx and use BenchmarkGCRAStoreCtx instead.
func BenchmarkGCRAStore(b *testing.B, st throttled.GCRAStore) {
	BenchmarkGCRAStoreCtx(b, throttled.WrapStoreWithContext(st))
}
