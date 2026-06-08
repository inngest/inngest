package queue

import (
	"crypto/rand"
	"sync"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

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
