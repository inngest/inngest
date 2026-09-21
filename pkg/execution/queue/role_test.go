package queue

import (
	"context"
	"crypto/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

type pooledRoleTestShard struct {
	*mockShardForIterator
	mu     sync.Mutex
	leases map[string]ulid.ULID
}

func (s *pooledRoleTestShard) RoleLease(_ context.Context, key string, duration time.Duration, existingLeaseID ...*ulid.ULID) (*ulid.ULID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if current, ok := s.leases[key]; ok {
		if len(existingLeaseID) == 0 || existingLeaseID[0] == nil || current.Compare(*existingLeaseID[0]) != 0 {
			return nil, ErrRoleAlreadyLeased
		}
	}

	leaseID, err := ulid.New(uint64(time.Now().Add(duration).UnixMilli()), rand.Reader)
	if err != nil {
		return nil, err
	}
	s.leases[key] = leaseID
	return &leaseID, nil
}

type roleProvidingScanner struct {
	roles []QueueRole
}

func (roleProvidingScanner) Run(context.Context, QueueScannerRuntime) error { return nil }
func (s roleProvidingScanner) QueueScannerRoles() []QueueRole               { return s.roles }

func TestWithQueueRoles(t *testing.T) {
	t.Run("appends custom roles to defaults", func(t *testing.T) {
		role := queueRole{name: "custom", leaseDuration: RoleLeaseDuration}
		opts := configuredRoleOptions(WithQueueRoles(role))

		names := roleNames(opts.roles)
		require.Contains(t, names, QueueRoleSequential)
		require.Contains(t, names, QueueRoleScavenger)
		require.Contains(t, names, QueueRoleInstrumentation)
		require.Contains(t, names, "custom")
	})

	t.Run("defaults from run mode and latency config", func(t *testing.T) {
		opts := configuredRoleOptions(WithLatencyPartition(LatencyPartitionOptions{
			Interval: time.Second,
		}))

		names := roleNames(opts.roles)

		require.Contains(t, names, QueueRoleSequential)
		require.Contains(t, names, QueueRoleScavenger)
		require.Contains(t, names, QueueRoleInstrumentation)
		require.Contains(t, names, QueueRoleLatencyTracker)
	})

	t.Run("omits default sequential role for allowlisted workers", func(t *testing.T) {
		opts := configuredRoleOptions(WithAllowQueueNames("critical"))

		names := roleNames(opts.roles)

		require.NotContains(t, names, QueueRoleSequential)
		require.Contains(t, names, QueueRoleScavenger)
		require.Contains(t, names, QueueRoleInstrumentation)
	})

	t.Run("filters custom sequential role for allowlisted workers", func(t *testing.T) {
		custom := queueRole{name: "custom", leaseDuration: RoleLeaseDuration}
		opts := configuredRoleOptions(
			WithQueueRoles(NewSequentialRole(), custom),
			WithAllowQueueNames("critical"),
		)

		names := roleNames(opts.roles)
		require.NotContains(t, names, QueueRoleSequential)
		require.Contains(t, names, QueueRoleScavenger)
		require.Contains(t, names, QueueRoleInstrumentation)
		require.Contains(t, names, "custom")
	})

	t.Run("filters nil roles", func(t *testing.T) {
		custom := queueRole{name: "custom", leaseDuration: RoleLeaseDuration}
		opts := configuredRoleOptions(WithQueueRoles(nil, custom))

		names := roleNames(opts.roles)
		require.Contains(t, names, QueueRoleSequential)
		require.Contains(t, names, QueueRoleScavenger)
		require.Contains(t, names, QueueRoleInstrumentation)
		require.Contains(t, names, "custom")
	})
}

func TestConfigureScannerRolesAppendsUniqueRoles(t *testing.T) {
	opts := NewQueueOptions()
	qp := &queueProcessor{QueueOptions: opts}
	qp.configureQueueRoles()

	require.NoError(t, qp.configureScannerRoles(roleProvidingScanner{roles: []QueueRole{NewSequentialRole()}}))
	require.Contains(t, roleNames(qp.roles), QueueRoleSequential)
}

func TestConfigureScannerRolesRejectsDuplicateNames(t *testing.T) {
	opts := NewQueueOptions(WithQueueRoles(NewSequentialRole()))
	qp := &queueProcessor{QueueOptions: opts}
	qp.configureQueueRoles()

	err := qp.configureScannerRoles(roleProvidingScanner{roles: []QueueRole{NewSequentialRole()}})
	require.EqualError(t, err, `queue scanner role "sequential" is already configured`)
}

func TestConfigureScannerRolesHonorsSequentialFilter(t *testing.T) {
	opts := NewQueueOptions(WithAllowQueueNames("critical"))
	qp := &queueProcessor{QueueOptions: opts}
	qp.configureQueueRoles()
	require.NoError(t, qp.configureScannerRoles(roleProvidingScanner{roles: []QueueRole{NewSequentialRole()}}))
	require.NotContains(t, roleNames(qp.roles), QueueRoleSequential)
}

func roleNames(roles []QueueRole) map[string]struct{} {
	names := map[string]struct{}{}
	for _, role := range roles {
		names[role.Name()] = struct{}{}
	}
	return names
}

func configuredRoleOptions(options ...QueueOpt) *QueueOptions {
	opts := NewQueueOptions(options...)
	qp := &queueProcessor{QueueOptions: opts}
	qp.configureQueueRoles()
	if err := qp.configureScannerRoles(partitionQueueScanner{q: qp}); err != nil {
		panic(err)
	}
	return opts
}

func TestActiveRoles(t *testing.T) {
	role := queueRole{
		name:             "exclusive",
		leaseDuration:    RoleLeaseDuration,
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

func TestRoleLeasePoolAdjustsWithoutRestart(t *testing.T) {
	var desired atomic.Int64
	desired.Store(1)
	role := NewSequentialRole(WithRoleLeaseCount(func(context.Context, QueueShard) int {
		return int(desired.Load())
	}))
	shard := &pooledRoleTestShard{
		mockShardForIterator: &mockShardForIterator{name: "test"},
		leases:               map[string]ulid.ULID{},
	}
	registry, err := NewSingleShardRegistry(shard)
	require.NoError(t, err)

	newProcessor := func() *queueProcessor {
		return &queueProcessor{
			QueueOptions:  NewQueueOptions(),
			roleLeaseLock: &sync.RWMutex{},
			roleLeaseIDs:  map[string]*ulid.ULID{},
			quit:          make(chan error, 1),
			shards:        registry,
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	first := newProcessor()
	second := newProcessor()
	go first.runRole(ctx, role)
	go second.runRole(ctx, role)

	require.Eventually(t, func() bool {
		return first.isRoleActive(QueueRoleSequential) != second.isRoleActive(QueueRoleSequential)
	}, time.Second, time.Millisecond)

	desired.Store(2)
	require.Eventually(t, func() bool {
		return first.isRoleActive(QueueRoleSequential) && second.isRoleActive(QueueRoleSequential)
	}, 2*RoleLeaseDuration, 10*time.Millisecond)

	desired.Store(1)
	require.Eventually(t, func() bool {
		return first.isRoleActive(QueueRoleSequential) != second.isRoleActive(QueueRoleSequential)
	}, 2*RoleLeaseDuration, 10*time.Millisecond)
}

func TestQueueRoleLeaseNamePreservesOriginalFirstSlot(t *testing.T) {
	require.Equal(t, QueueRoleSequential, queueRoleLeaseName(QueueRoleSequential, 0))
	require.Equal(t, QueueRoleSequential+"-1", queueRoleLeaseName(QueueRoleSequential, 1))
}
