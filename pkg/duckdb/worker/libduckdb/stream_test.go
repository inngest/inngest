package libduckdb

import (
	"errors"
	"testing"
)

func TestStreamQueueMemoryBound(t *testing.T) {
	st := RegisterStream("test-bound")
	defer UnregisterStream("test-bound")
	st.SetMaxQueueBytes(200)

	small := StreamChunk{
		RowCount: 1,
		Columns:  []StreamColumn{{Kind: StreamColumnVarchar, Bytes: make([]byte, 64), Ends: []uint32{64}}},
	}
	if err := st.Push(small); err != nil {
		t.Fatalf("first Push failed: %v", err)
	}

	// Second push exceeds the 200-byte limit and must fail without blocking.
	if err := st.Push(small); !errors.Is(err, ErrStreamQueueOverflow) {
		t.Fatalf("second Push = %v, want ErrStreamQueueOverflow", err)
	}

	// Consumer receives the overflow error cleanly on Next.
	ch, ok := st.Next()
	if !ok || !errors.Is(ch.Err, ErrStreamQueueOverflow) {
		t.Fatalf("Next = (ok=%v, err=%v), want ErrStreamQueueOverflow", ok, ch.Err)
	}
}
