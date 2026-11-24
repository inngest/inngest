package util

import (
	"context"
	"time"

	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
)

func Lease(
	ctx context.Context,
	name string,
	lease func(ctx context.Context) (ulid.ULID, error),
	renew func(ctx context.Context, leaseID ulid.ULID) (ulid.ULID, error),
	revoke func(ctx context.Context, leaseID ulid.ULID) error,
	do func(ctx context.Context) error,
	interval time.Duration,
) error {
	// Any lease is also a crit.
	return Crit(ctx, name, func(ctx context.Context) error {
		leaseID, err := lease(ctx)
		if err != nil {
			return err
		}

		cancelCtx, cancel := context.WithCancel(ctx)
		renewDone := make(chan struct{})

		go func() {
			defer close(renewDone)
			for {
				select {
				case <-cancelCtx.Done():
					// Stop renewing the lease.
					return
				case <-time.After(interval):
					if cancelCtx.Err() != nil {
						return
					}
					nextLeaseID, err := renew(cancelCtx, leaseID)
					if err != nil {
						if cancelCtx.Err() != nil {
							return
						}
						// cancel the ctx, as the lease failed.
						cancel()
						logger.StdlibLogger(ctx).Error(
							"failed to renew lease",
							"error", err,
							"lease_operation", name,
						)
						return
					}
					leaseID = nextLeaseID
				}
			}
		}()

		defer func() {
			// Join any in-flight renewal before reading the final ID and revoking.
			cancel()
			<-renewDone
			if err := revoke(ctx, leaseID); err != nil {
				logger.StdlibLogger(ctx).Warn(
					"failed to revoke lease",
					"error", err,
					"lease_operation", name,
				)
			}
		}()

		return do(cancelCtx)
	})
}
