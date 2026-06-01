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
	t.Run("uses explicit roles", func(t *testing.T) {
		role := queueRole{name: "custom", leaseDuration: RoleLeaseDuration}
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

	t.Run("omits default sequential role for allowlisted workers", func(t *testing.T) {
		opts := NewQueueOptions(WithAllowQueueNames("critical"))

		names := map[string]struct{}{}
		for _, role := range opts.roles {
			names[role.Name()] = struct{}{}
		}

		require.NotContains(t, names, QueueRoleSequential)
		require.Contains(t, names, QueueRoleScavenger)
		require.Contains(t, names, QueueRoleInstrumentation)
	})

	t.Run("filters explicit sequential role for allowlisted workers", func(t *testing.T) {
		custom := queueRole{name: "custom", leaseDuration: RoleLeaseDuration}
		opts := NewQueueOptions(
			WithQueueRoles(NewSequentialRole(), custom),
			WithAllowQueueNames("critical"),
		)

		require.Len(t, opts.roles, 1)
		require.Equal(t, "custom", opts.roles[0].Name())
	})
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
