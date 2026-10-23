package devserver

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/util"
	"github.com/stretchr/testify/require"
)

func TestShouldPollSDK(t *testing.T) {
	tests := []struct {
		name      string
		poll      bool
		appError  string
		firstSync bool
		want      bool
	}{
		{name: "no-poll healthy app still syncs once for -u", poll: false, firstSync: true, want: true},
		{name: "no-poll healthy app skipped after that sync", poll: false, want: false},
		{name: "no-poll app with an error still syncs", poll: false, appError: "unreachable", want: true},
		{name: "poll syncs a healthy app", poll: true, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldPollSDK(tt.poll, tt.appError, tt.firstSync)
			if got != tt.want {
				t.Fatalf("shouldPollSDK(%v, %q, %v) = %v, want %v", tt.poll, tt.appError, tt.firstSync, got, tt.want)
			}
		})
	}
}

// TestPollSDKs_FirstSyncPingsOnceUnderNoPoll exercises the real pollSDKs loop
// (not just the extracted shouldPollSDK helper): a -u URL must still be
// pinged once at startup under --no-poll, and never again once that first
// sync lands, even though pollSDKs keeps looping.
func TestPollSDKs_FirstSyncPingsOnceUnderNoPoll(t *testing.T) {
	var hits atomic.Int32

	ds := newTestDevServer(t)

	var appID uuid.UUID
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		// A real SDK would call back into /fn/register on receiving this
		// ping, which clears the placeholder error. Simulate that so the
		// next pollSDKs iteration sees a healthy, errorless app.
		_, _ = ds.Data.UpdateAppError(r.Context(), cqrs.UpdateAppErrorParams{
			ID:    appID,
			Error: sql.NullString{},
		})
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	appID = inngest.DeterministicAppUUID(util.NormalizeAppURL(srv.URL, false))

	ds.Opts = StartOpts{
		URLs:         []string{srv.URL},
		Poll:         false,
		PollInterval: 0, // tight loop so several iterations run during the test
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	defer func() {
		cancel()
		<-done
	}()
	go func() {
		ds.pollSDKs(ctx)
		close(done)
	}()

	require.Eventually(t, func() bool { return hits.Load() >= 1 }, 2*time.Second, 10*time.Millisecond,
		"the -u URL must be pinged once at startup even under --no-poll")

	// Let the tight loop (PollInterval=0) run many more iterations. A
	// regression that keeps treating this as a first sync, or that never
	// clears the saved error, would ping again on every one of them.
	time.Sleep(150 * time.Millisecond)
	cancel()
	<-done

	require.Equal(t, int32(1), hits.Load(),
		"an app with no error must not be pinged again under --no-poll once its first sync has landed")
}
