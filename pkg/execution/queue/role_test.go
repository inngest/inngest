package queue

import (
	"context"
	"crypto/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestWithQueueRoles(t *testing.T) {
	t.Run("uses explicit roles", func(t *testing.T) {
		role := queueRole{name: "custom", leaseDuration: ConfigLeaseDuration}
		opts := NewQueueOptions(WithQueueRoles(role))

		require.Len(t, opts.roles, 1)
		require.Equal(t, "custom", opts.roles[0].Name())
	})

	t.Run("defaults from run mode and latency config", func(t *testing.T) {
		opts := NewQueueOptions(WithLatencyPartition(LatencyPartitionOptions{
			Interval: time.Second,
		}))

		names := map[string]struct{}{}
		for _, role := range opts.roles {
			names[role.Name()] = struct{}{}
		}

		require.Contains(t, names, QueueRoleSequential)
		require.Contains(t, names, QueueRoleScavenger)
		require.Contains(t, names, QueueRoleInstrumentation)
		require.Contains(t, names, QueueRoleLatencyTracker)
	})
}

func TestQueueRoleScanExclusion(t *testing.T) {
	role := queueRole{
		name:             "exclusive",
		leaseDuration:    ConfigLeaseDuration,
		excludesScanning: true,
	}

	shard := &scanCountingShard{mockShardForIterator: mockShardForIterator{name: "test"}}
	shards, err := NewSingleShardRegistry(shard)
	require.NoError(t, err)

	qp := &queueProcessor{
		QueueOptions:  NewQueueOptions(WithQueueRoles(role)),
		roleLeaseLock: &sync.RWMutex{},
		roleLeaseIDs:  map[string]*ulid.ULID{},
		shards:        shards,
	}

	leaseID, err := ulid.New(uint64(time.Now().Add(time.Minute).UnixMilli()), rand.Reader)
	require.NoError(t, err)
	qp.roleLeaseIDs[role.Name()] = &leaseID

	require.NoError(t, qp.scan(context.Background()))
	require.Equal(t, int32(0), shard.partitionPeekCalls.Load())
}

func TestActiveRoles(t *testing.T) {
	role := queueRole{
		name:             "exclusive",
		leaseDuration:    ConfigLeaseDuration,
		excludesScanning: true,
	}

	qp := &queueProcessor{
		QueueOptions:  NewQueueOptions(WithQueueRoles(role)),
		roleLeaseLock: &sync.RWMutex{},
		roleLeaseIDs:  map[string]*ulid.ULID{},
	}

	expired, err := ulid.New(uint64(time.Now().Add(-time.Minute).UnixMilli()), rand.Reader)
	require.NoError(t, err)
	active, err := ulid.New(uint64(time.Now().Add(time.Minute).UnixMilli()), rand.Reader)
	require.NoError(t, err)

	qp.roleLeaseIDs["expired"] = &expired
	qp.roleLeaseIDs[role.Name()] = &active

	statuses := qp.ActiveRoles()
	require.Len(t, statuses, 1)
	require.Equal(t, role.Name(), statuses[0].Name)
	require.Equal(t, active, statuses[0].LeaseID)
	require.True(t, statuses[0].LeaseExpiresAt.After(time.Now()))
	require.True(t, statuses[0].ExcludesScanning)
}

func TestRunRole(t *testing.T) {
	fakeClock := clockwork.NewFakeClock()
	role := queueRole{
		name:          "custom",
		leaseDuration: 3 * time.Second,
		runInterval:   time.Second,
		run: func(ctx context.Context, q QueueRoleProcessor, shard QueueShard) error {
			return nil
		},
	}

	shard := &leasedRoleShard{
		mockShardForIterator: mockShardForIterator{name: "test"},
		clock:                fakeClock,
	}
	shards, err := NewSingleShardRegistry(shard)
	require.NoError(t, err)

	qp := &queueProcessor{
		QueueOptions:  NewQueueOptions(WithClock(fakeClock), WithQueueRoles(role)),
		quit:          make(chan error, 1),
		roleLeaseLock: &sync.RWMutex{},
		roleLeaseIDs:  map[string]*ulid.ULID{},
		shards:        shards,
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		qp.runRole(ctx, role)
	}()

	require.Eventually(t, func() bool {
		return shard.configLeaseCalls.Load() == 1
	}, time.Second, 10*time.Millisecond)
	require.True(t, qp.isRoleActive(role.Name()))

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("role runner did not stop")
	}
}

type scanCountingShard struct {
	mockShardForIterator
	partitionPeekCalls atomic.Int32
}

func (s *scanCountingShard) PartitionPeek(ctx context.Context, sequential bool, until time.Time, limit int64) ([]*QueuePartition, error) {
	s.partitionPeekCalls.Add(1)
	return nil, nil
}

type leasedRoleShard struct {
	mockShardForIterator
	clock            clockwork.Clock
	configLeaseCalls atomic.Int32
}

func (s *leasedRoleShard) ConfigLease(ctx context.Context, key string, duration time.Duration, existingLeaseID ...*ulid.ULID) (*ulid.ULID, error) {
	s.configLeaseCalls.Add(1)
	leaseID, err := ulid.New(uint64(s.clock.Now().Add(duration).UnixMilli()), rand.Reader)
	if err != nil {
		return nil, err
	}
	return &leaseID, nil
}
