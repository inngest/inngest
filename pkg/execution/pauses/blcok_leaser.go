package pauses

import (
	"context"

	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
)

type redisBlockLeaser struct {
	rc rueidis.Client
}

// Lease leases a given index, ensuring that only one worker can
// flush an index at a time.
func (r redisBlockLeaser) Lease(ctx context.Context, index Index) (leaseID ulid.ULID, err error) {
	// TODO: Key and lease.
}

// Renew renews a lease while we are flushing an index.
func (r redisBlockLeaser) Renew(ctx context.Context, index Index, leaseID ulid.ULID) (newLeaseID ulid.ULID, err error) {
}

// Revoke drops a lease, allowing any other worker to flush an index.
func (r redisBlockLeaser) Revoke(ctx context.Context, index Index, leaseID ulid.ULID) (err error) {
}
