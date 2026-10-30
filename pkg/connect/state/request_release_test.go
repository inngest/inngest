package state

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestReleaseRequestLease(t *testing.T) {
	for _, name := range []string{"missing", "matching", "renewed", "reassigned"} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			r := miniredis.RunT(t)
			rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{r.Addr()}, DisableCache: true})
			require.NoError(t, err)
			defer rc.Close()
			clock := clockwork.NewFakeClock()
			sm := NewRedisConnectionStateManager(rc, RedisStateManagerOpt{Clock: clock})
			envID := uuid.New()
			const requestID = "stable-job-id"
			originalIP := net.IPv4(127, 0, 0, 1)
			leaseID := ulid.Make()
			var expectedIP net.IP

			if name != "missing" {
				lease, err := sm.LeaseRequest(ctx, envID, requestID, consts.ConnectWorkerRequestLeaseDuration, originalIP)
				require.NoError(t, err)
				leaseID = *lease
			}
			switch name {
			case "renewed":
				clock.Advance(time.Second)
				_, err := sm.ExtendRequestLease(ctx, envID, "worker", requestID, leaseID, consts.ConnectWorkerRequestLeaseDuration, true)
				require.NoError(t, err)
				expectedIP = originalIP
			case "reassigned":
				clock.Advance(consts.ConnectWorkerRequestLeaseDuration + time.Second)
				expectedIP = net.IPv4(127, 0, 0, 2)
				_, err := sm.LeaseRequest(ctx, envID, requestID, consts.ConnectWorkerRequestLeaseDuration, expectedIP)
				require.NoError(t, err)
			}

			// Repeated cleanup is harmless, and an old owner cannot delete a
			// lease that has since been renewed or replaced.
			for range 2 {
				require.NoError(t, sm.ReleaseRequestLease(ctx, envID, requestID, leaseID))
				leased, err := sm.IsRequestLeased(ctx, envID, requestID)
				require.NoError(t, err)
				require.Equal(t, expectedIP != nil, leased)
				if expectedIP != nil {
					ip, err := sm.GetExecutorIP(ctx, envID, requestID)
					require.NoError(t, err)
					require.True(t, expectedIP.Equal(ip))
				}
			}
		})
	}
}
