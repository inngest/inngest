package constraintapi

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/jonboulle/clockwork"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestRedisCapacityManager(t *testing.T) {
	r := miniredis.RunT(t)
	ctx := context.Background()

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()

	cm, err := NewRedisCapacityManager(
		WithClient(rc),
		WithClock(clock),
	)
	require.NoError(t, err)
	require.NotNil(t, cm)

	// The following tests are essential functionality. We also have detailed test for each method,
	// to cover edge cases.

	t.Run("Acquire", func(t *testing.T) {
		resp, err := cm.Acquire(ctx, &CapacityAcquireRequest{
			CurrentTime: clock.Now(),
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("Check", func(t *testing.T) {
		resp, userErr, internalErr := cm.Check(ctx, &CapacityCheckRequest{})
		require.NoError(t, userErr)
		require.NoError(t, internalErr)
		require.NotNil(t, resp)
	})

	t.Run("Extend", func(t *testing.T) {
		resp, err := cm.ExtendLease(ctx, &CapacityExtendLeaseRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("Release", func(t *testing.T) {
		resp, err := cm.Release(ctx, &CapacityReleaseRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})
}

func TestRedisCapacityManager_keyPrefix(t *testing.T) {
	manager := &redisCapacityManager{
		rateLimitKeyPrefix:  "rate-limit-prefix",
		queueStateKeyPrefix: "queue-state-prefix",
	}

	tests := []struct {
		name        string
		constraints []ConstraintItem
		want        string
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "empty constraints - returns queue state prefix",
			constraints: []ConstraintItem{},
			want:        "queue-state-prefix",
			wantErr:     false,
		},
		{
			name: "only concurrency constraint - returns queue state prefix",
			constraints: []ConstraintItem{
				{Kind: ConstraintKindConcurrency},
			},
			want:    "queue-state-prefix",
			wantErr: false,
		},
		{
			name: "only throttle constraint - returns queue state prefix",
			constraints: []ConstraintItem{
				{Kind: ConstraintKindThrottle},
			},
			want:    "queue-state-prefix",
			wantErr: false,
		},
		{
			name: "only rate limit constraint - returns rate limit prefix",
			constraints: []ConstraintItem{
				{Kind: ConstraintKindRateLimit},
			},
			want:    "rate-limit-prefix",
			wantErr: false,
		},
		{
			name: "multiple queue constraints (concurrency + throttle) - returns queue state prefix",
			constraints: []ConstraintItem{
				{Kind: ConstraintKindConcurrency},
				{Kind: ConstraintKindThrottle},
			},
			want:    "queue-state-prefix",
			wantErr: false,
		},
		{
			name: "multiple concurrency constraints - returns queue state prefix",
			constraints: []ConstraintItem{
				{Kind: ConstraintKindConcurrency},
				{Kind: ConstraintKindConcurrency},
			},
			want:    "queue-state-prefix",
			wantErr: false,
		},
		{
			name: "multiple throttle constraints - returns queue state prefix",
			constraints: []ConstraintItem{
				{Kind: ConstraintKindThrottle},
				{Kind: ConstraintKindThrottle},
			},
			want:    "queue-state-prefix",
			wantErr: false,
		},
		{
			name: "multiple rate limit constraints - returns rate limit prefix",
			constraints: []ConstraintItem{
				{Kind: ConstraintKindRateLimit},
				{Kind: ConstraintKindRateLimit},
			},
			want:    "rate-limit-prefix",
			wantErr: false,
		},
		{
			name: "mixed: rate limit + concurrency - error",
			constraints: []ConstraintItem{
				{Kind: ConstraintKindRateLimit},
				{Kind: ConstraintKindConcurrency},
			},
			want:    "",
			wantErr: true,
			errMsg:  "mixed constraints are not allowed during the first stage",
		},
		{
			name: "mixed: rate limit + throttle - error",
			constraints: []ConstraintItem{
				{Kind: ConstraintKindRateLimit},
				{Kind: ConstraintKindThrottle},
			},
			want:    "",
			wantErr: true,
			errMsg:  "mixed constraints are not allowed during the first stage",
		},
		{
			name: "mixed: concurrency + rate limit - error",
			constraints: []ConstraintItem{
				{Kind: ConstraintKindConcurrency},
				{Kind: ConstraintKindRateLimit},
			},
			want:    "",
			wantErr: true,
			errMsg:  "mixed constraints are not allowed during the first stage",
		},
		{
			name: "mixed: throttle + rate limit - error",
			constraints: []ConstraintItem{
				{Kind: ConstraintKindThrottle},
				{Kind: ConstraintKindRateLimit},
			},
			want:    "",
			wantErr: true,
			errMsg:  "mixed constraints are not allowed during the first stage",
		},
		{
			name: "mixed: all three constraint types - error",
			constraints: []ConstraintItem{
				{Kind: ConstraintKindConcurrency},
				{Kind: ConstraintKindThrottle},
				{Kind: ConstraintKindRateLimit},
			},
			want:    "",
			wantErr: true,
			errMsg:  "mixed constraints are not allowed during the first stage",
		},
		{
			name: "mixed: multiple mixed constraints - error",
			constraints: []ConstraintItem{
				{Kind: ConstraintKindRateLimit},
				{Kind: ConstraintKindConcurrency},
				{Kind: ConstraintKindRateLimit},
				{Kind: ConstraintKindThrottle},
			},
			want:    "",
			wantErr: true,
			errMsg:  "mixed constraints are not allowed during the first stage",
		},
		{
			name: "unknown constraint type - returns queue state prefix (default)",
			constraints: []ConstraintItem{
				{Kind: ConstraintKind("unknown")},
			},
			want:    "queue-state-prefix",
			wantErr: false,
		},
		{
			name: "mix of unknown and known constraints - follows same rules",
			constraints: []ConstraintItem{
				{Kind: ConstraintKind("unknown")},
				{Kind: ConstraintKindConcurrency},
			},
			want:    "queue-state-prefix",
			wantErr: false,
		},
		{
			name: "mix of unknown and rate limit - error",
			constraints: []ConstraintItem{
				{Kind: ConstraintKind("unknown")},
				{Kind: ConstraintKindRateLimit},
			},
			want:    "rate-limit-prefix",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := manager.keyPrefix(tt.constraints)

			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errMsg)
				require.Equal(t, tt.want, got)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}
