package dnscache

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// fakeLookup records the names it is asked for and answers from answers.
type fakeLookup struct {
	mu      sync.Mutex
	names   []string
	answers map[string]error
	delay   time.Duration
	calls   atomic.Int32
}

func (f *fakeLookup) lookup(ctx context.Context, host string) ([]net.IPAddr, error) {
	f.calls.Add(1)
	f.mu.Lock()
	f.names = append(f.names, host)
	err, ok := f.answers[host]
	f.mu.Unlock()
	time.Sleep(f.delay)
	if ok && err != nil {
		return nil, err
	}
	return []net.IPAddr{{IP: net.ParseIP("203.0.113.10")}}, nil
}

func notFound(name string) error {
	return &net.DNSError{Err: "no such host", Name: name, IsNotFound: true}
}

func TestLookupNames(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		answers map[string]error
		want    []string
		wantErr bool
	}{
		{
			name: "dotted host is looked up as an absolute name",
			host: "api.example.com",
			want: []string{"api.example.com."},
		},
		{
			name:    "falls back to the search list when the absolute name does not exist",
			host:    "my-svc.my-ns",
			answers: map[string]error{"my-svc.my-ns.": notFound("my-svc.my-ns.")},
			want:    []string{"my-svc.my-ns.", "my-svc.my-ns"},
		},
		{
			name:    "other errors do not fall back",
			host:    "api.example.com",
			answers: map[string]error{"api.example.com.": &net.DNSError{Err: "timeout", IsTimeout: true}},
			want:    []string{"api.example.com."},
			wantErr: true,
		},
		{
			name: "host without a dot uses the search list",
			host: "localhost",
			want: []string{"localhost"},
		},
		{
			name: "absolute host is unchanged",
			host: "api.example.com.",
			want: []string{"api.example.com."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeLookup{answers: tt.answers}
			r := New()
			r.lookupIPAddr = f.lookup

			_, err := r.Lookup(context.Background(), tt.host)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.want, f.names)
		})
	}
}

func TestLookupMergesConcurrentCalls(t *testing.T) {
	f := &fakeLookup{delay: 50 * time.Millisecond}
	r := New()
	r.lookupIPAddr = f.lookup

	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := r.Lookup(context.Background(), "api.example.com")
			require.NoError(t, err)
		}()
	}
	wg.Wait()

	require.EqualValues(t, 1, f.calls.Load())
}

func TestLookupUsesStaleEntryWhenLookupFails(t *testing.T) {
	f := &fakeLookup{}
	r := New(WithCacheTTL(10 * time.Millisecond))
	r.lookupIPAddr = f.lookup

	ips, err := r.Lookup(context.Background(), "api.example.com")
	require.NoError(t, err)

	time.Sleep(20 * time.Millisecond)
	f.mu.Lock()
	f.answers = map[string]error{"api.example.com.": errors.New("dns down")}
	f.mu.Unlock()

	stale, err := r.Lookup(context.Background(), "api.example.com")
	require.NoError(t, err)
	require.Equal(t, ips, stale)
	require.EqualValues(t, 2, f.calls.Load())
}
