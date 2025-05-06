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
		defer cancel()

		go func() {
			for {
				select {
				case <-cancelCtx.Done():
					// Stop renewing the lease.
					return
				case <-time.After(interval):
					if leaseID, err = renew(ctx, leaseID); err != nil {
						// cancel the ctx, as the lease failed.
						cancel()
						logger.StdlibLogger(ctx).Error(
							"failed to renew lease",
							"error", err,
							"lease_operation", name,
						)
					}
				}
			}
		}()

		defer func() {
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
