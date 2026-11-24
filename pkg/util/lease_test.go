package util

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestLeaseStopsRenewalBeforeRevoking(t *testing.T) {
	started := make(chan struct{})
	stopped := make(chan struct{})
	var renewed atomic.Bool
	var revokedWhileRenewing atomic.Bool
	original := ulid.Make()
	updated := ulid.Make()
	var revokedID ulid.ULID
	err := Lease(context.Background(), "test lease",
		func(context.Context) (ulid.ULID, error) { return original, nil },
		func(ctx context.Context, id ulid.ULID) (ulid.ULID, error) {
			close(started)
			select {
			case <-ctx.Done():
			case <-time.After(time.Second):
			}
			renewed.Store(true)
			close(stopped)
			return updated, nil
		},
		func(ctx context.Context, id ulid.ULID) error {
			revokedWhileRenewing.Store(!renewed.Load())
			revokedID = id
			return nil
		},
		func(context.Context) error { <-started; return nil },
		time.Millisecond,
	)
	<-stopped
	require.NoError(t, err)
	require.False(t, revokedWhileRenewing.Load(), "revocation must wait for an in-flight renewal")
	require.Equal(t, updated, revokedID)
}

func TestLeaseRenewalFailureRevokesLastSuccessfulID(t *testing.T) {
	original := ulid.Make()
	var revokedID ulid.ULID
	err := Lease(context.Background(), "failed renewal",
		func(context.Context) (ulid.ULID, error) { return original, nil },
		func(context.Context, ulid.ULID) (ulid.ULID, error) { return ulid.ULID{}, errors.New("renewal failed") },
		func(ctx context.Context, id ulid.ULID) error {
			revokedID = id
			require.NoError(t, ctx.Err())
			return nil
		},
		func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() },
		time.Millisecond,
	)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, original, revokedID)
}

func TestLeaseDoesNotRenewAfterWorkCompletes(t *testing.T) {
	var renewals atomic.Int64
	var revoked bool
	workErr := errors.New("work failed")
	err := Lease(context.Background(), "short work",
		func(context.Context) (ulid.ULID, error) { return ulid.Make(), nil },
		func(context.Context, ulid.ULID) (ulid.ULID, error) { renewals.Add(1); return ulid.Make(), nil },
		func(ctx context.Context, id ulid.ULID) error {
			revoked = true
			require.NoError(t, ctx.Err())
			return nil
		},
		func(context.Context) error { return workErr },
		time.Hour,
	)
	require.ErrorIs(t, err, workErr)
	require.True(t, revoked)
	require.Zero(t, renewals.Load())
}
