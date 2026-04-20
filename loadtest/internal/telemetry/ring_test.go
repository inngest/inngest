package telemetry

import (
	"sync"
	"testing"
	"time"
)

func TestRingDropsOldestWhenFull(t *testing.T) {
	r := NewRing(3)
	for i := 0; i < 5; i++ {
		r.Push(Frame{Seq: uint64(i)})
	}
	if got := r.Dropped(); got != 2 {
		t.Errorf("dropped: got %d, want 2", got)
	}
	var seqs []uint64
	for i := 0; i < 3; i++ {
		f, ok := r.Pop()
		if !ok {
			t.Fatalf("pop %d: !ok", i)
		}
		seqs = append(seqs, f.Seq)
	}
	// Oldest dropped; we should see the 3 newest: 2, 3, 4.
	want := []uint64{2, 3, 4}
	for i, w := range want {
		if seqs[i] != w {
			t.Errorf("seqs[%d]: got %d, want %d", i, seqs[i], w)
		}
	}
}

func TestRingPopBlocksUntilPushOrClose(t *testing.T) {
	r := NewRing(1)
	var wg sync.WaitGroup
	wg.Add(1)
	popped := make(chan Frame, 1)
	go func() {
		defer wg.Done()
		f, ok := r.Pop()
		if ok {
			popped <- f
		}
	}()
	time.Sleep(10 * time.Millisecond)
	r.Push(Frame{Seq: 7})
	select {
	case f := <-popped:
		if f.Seq != 7 {
			t.Errorf("seq: got %d, want 7", f.Seq)
		}
	case <-time.After(time.Second):
		t.Fatal("pop did not unblock after push")
	}
	wg.Wait()
}

func TestRingCloseUnblocksPendingPop(t *testing.T) {
	r := NewRing(1)
	done := make(chan struct{})
	go func() {
		_, ok := r.Pop()
		if ok {
			t.Errorf("closed ring should pop ok=false")
		}
		close(done)
	}()
	time.Sleep(10 * time.Millisecond)
	r.Close()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Close did not unblock pending Pop")
	}
}
