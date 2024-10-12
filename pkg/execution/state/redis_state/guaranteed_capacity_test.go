package redis_state

import (
	"context"
	"encoding/json"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestSerializeGuaranteedCapacity(t *testing.T) {
	cases := []struct {
		name string
		got  GuaranteedCapacity
		want json.RawMessage
	}{
		{
			name: "leased gc",
			got: GuaranteedCapacity{
				Scope:              enums.GuaranteedCapacityScopeAccount,
				AccountID:          uuid.MustParse("c06e5559-74fd-4404-8754-d06b6f342d10"),
				Priority:           0,
				GuaranteedCapacity: 1,
				Leases:             []ulid.ULID{ulid.MustParse("01J9S37W106HK23TGK5MNPY09J")},
			},
			want: json.RawMessage(`{"s":"Account","a":"c06e5559-74fd-4404-8754-d06b6f342d10","p":0,"gc":1,"leases":["01J9S37W106HK23TGK5MNPY09J"]}`),
		},
		{
			name: "non-leased gc",
			got: GuaranteedCapacity{
				Scope:              enums.GuaranteedCapacityScopeAccount,
				AccountID:          uuid.MustParse("c06e5559-74fd-4404-8754-d06b6f342d10"),
				Priority:           0,
				GuaranteedCapacity: 1,
			},
			want: json.RawMessage(`{"s":"Account","a":"c06e5559-74fd-4404-8754-d06b6f342d10","p":0,"gc":1,"leases":null}`),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := json.Marshal(tc.got)
			require.NoError(t, err)
			require.JSONEq(t, string(tc.want), string(got))
		})
	}
}

func TestDeserializeGuaranteedCapacity(t *testing.T) {
	cases := []struct {
		name string
		got  json.RawMessage
		want GuaranteedCapacity
	}{
		{
			name: "empty gc",
			got:  json.RawMessage(`{}`),
			want: GuaranteedCapacity{},
		},
		{
			name: "gc with empty leases obj",
			got: json.RawMessage(`{
				"s": "Account",
				"a": "c06e5559-74fd-4404-8754-d06b6f342d10",
				"p": 0,
				"gc": 1,
				"leases": {}
			}`),
			want: GuaranteedCapacity{
				Scope:              enums.GuaranteedCapacityScopeAccount,
				AccountID:          uuid.MustParse("c06e5559-74fd-4404-8754-d06b6f342d10"),
				Priority:           0,
				GuaranteedCapacity: 1,
				Leases:             nil,
			},
		},
		{
			name: "gc with empty leases obj",
			got: json.RawMessage(`{
				"s": "Account",
				"a": "c06e5559-74fd-4404-8754-d06b6f342d10",
				"p": 0,
				"gc": 1,
				"leases": []
			}`),
			want: GuaranteedCapacity{
				Scope:              enums.GuaranteedCapacityScopeAccount,
				AccountID:          uuid.MustParse("c06e5559-74fd-4404-8754-d06b6f342d10"),
				Priority:           0,
				GuaranteedCapacity: 1,
				Leases:             []ulid.ULID{},
			},
		},
		{
			name: "gc with non-empty leases obj",
			got: json.RawMessage(`{
				"s": "Account",
				"a": "c06e5559-74fd-4404-8754-d06b6f342d10",
				"p": 0,
				"gc": 2,
				"leases": ["01J9S37W106HK23TGK5MNPY09J", "01J9S37YW0HHABTVCJ7WNFAV5N"]
			}`),
			want: GuaranteedCapacity{
				Scope:              enums.GuaranteedCapacityScopeAccount,
				AccountID:          uuid.MustParse("c06e5559-74fd-4404-8754-d06b6f342d10"),
				Priority:           0,
				GuaranteedCapacity: 2,
				Leases: []ulid.ULID{
					ulid.MustParse("01J9S37W106HK23TGK5MNPY09J"),
					ulid.MustParse("01J9S37YW0HHABTVCJ7WNFAV5N"),
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got GuaranteedCapacity
			err := json.Unmarshal(tc.got, &got)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestGetGuaranteedCapacityMap(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()
	ctx := context.Background()

	accountId := uuid.New()
	enableGuaranteedCapacity := true // indicate whether to enable guaranteed capacity in tests
	guaranteedCapacity := GuaranteedCapacity{
		Scope:              enums.GuaranteedCapacityScopeAccount,
		AccountID:          accountId,
		GuaranteedCapacity: 1,
	}

	sf := func(ctx context.Context, accountId uuid.UUID) *GuaranteedCapacity {
		if !enableGuaranteedCapacity {
			return nil
		}
		return &guaranteedCapacity
	}
	q := NewQueue(
		NewQueueClient(rc, QueueDefaultKey),
		WithRunMode(QueueRunMode{
			Sequential:         false,
			Scavenger:          false,
			Partition:          false,
			Account:            false,
			AccountWeight:      0,
			GuaranteedCapacity: false,
		}),
		WithGuaranteedCapacityFinder(sf),
	)
	require.NotNil(t, sf(ctx, accountId))

	empty := q.getAccountLeases()
	require.Len(t, empty, 0, "expected leases to be empty")

	gc, err := q.getGuaranteedCapacityMap(ctx)
	require.NoError(t, err)
	require.Len(t, gc, 0)

	fnId := uuid.New()
	//	randomUlid := ulid.MustNew(ulid.Now(), rand.Reader)
	_, err = q.EnqueueItem(ctx, QueueItem{
		FunctionID: fnId,
		Data: osqueue.Item{
			Identifier: state.Identifier{
				AccountID: accountId,
			},
		},
	}, time.Now())
	require.NoError(t, err)

	// Empty state: guaranteed capacity exists as unleased item
	var gcToLease GuaranteedCapacity
	{
		gc, err = q.getGuaranteedCapacityMap(ctx)
		require.NoError(t, err)
		require.Len(t, gc, 1)
		require.NotNil(t, gc[guaranteedCapacity.Key()], "expected guaranteed capacity for account to be set", gc)
		require.Equal(t, guaranteedCapacity, gc[guaranteedCapacity.Key()])

		filtered, err := q.filterUnleasedAccounts(gc)
		require.NoError(t, err)
		require.Len(t, filtered, 1)
		require.Equal(t, guaranteedCapacity, filtered[0])
		require.Len(t, filtered[0].Leases, 0)

		localLeases := q.getAccountLeases()
		require.Len(t, localLeases, 0, "expected leases to be empty")

		// Set guaranteed capacity to lease for remaining test
		gcToLease = filtered[0]
	}

	// Lease guaranteed capacity
	var leaseId *ulid.ULID
	{
		// lease index = 0
		leaseId, err = q.leaseAccount(ctx, gcToLease, time.Second, 0)
		require.NoError(t, err)
		require.NotNil(t, leaseId)

		gc, err = q.getGuaranteedCapacityMap(ctx)
		require.NoError(t, err)
		require.Len(t, gc, 1)
		require.NotNil(t, gc[guaranteedCapacity.Key()], "expected guaranteed capacity for account to be set", gc)
		require.Equal(t, []ulid.ULID{*leaseId}, gc[guaranteedCapacity.Key()].Leases)

		filtered, err := q.filterUnleasedAccounts(gc)
		require.NoError(t, err)
		require.Len(t, filtered, 0)

		q.addLeasedAccount(guaranteedCapacity, *leaseId)

		localLeases := q.getAccountLeases()
		require.Len(t, localLeases, 1)
		require.Equal(t, *leaseId, localLeases[0].Lease)
	}

	{
		newLeaseId, err := q.renewAccountLease(ctx, gcToLease, time.Second, *leaseId)
		require.NoError(t, err)
		require.NotNil(t, newLeaseId)

		gc, err = q.getGuaranteedCapacityMap(ctx)
		require.NoError(t, err)
		require.Len(t, gc, 1)
		require.NotNil(t, gc[guaranteedCapacity.Key()], "expected guaranteed capacity for account to be set", gc)
		require.Equal(t, []ulid.ULID{*newLeaseId}, gc[guaranteedCapacity.Key()].Leases)

		filtered, err := q.filterUnleasedAccounts(gc)
		require.NoError(t, err)
		require.Len(t, filtered, 0)

		q.addLeasedAccount(guaranteedCapacity, *newLeaseId)

		localLeases := q.getAccountLeases()
		require.Len(t, localLeases, 1)
		require.Equal(t, *newLeaseId, localLeases[0].Lease)

		// Lease has been renewed, so the leaseId should be updated
		leaseId = newLeaseId
	}

	// Expire again
	{
		err = q.expireAccountLease(ctx, gcToLease, *leaseId)
		require.NoError(t, err)

		q.removeLeasedAccount(guaranteedCapacity)

		gc, err = q.getGuaranteedCapacityMap(ctx)
		require.NoError(t, err)
		require.Len(t, gc, 1)
		require.NotNil(t, gc[guaranteedCapacity.Key()], "expected guaranteed capacity for account to be set", gc)
		require.NotEqual(t, []ulid.ULID{*leaseId}, gc[guaranteedCapacity.Key()].Leases)
		require.Len(t, gc[guaranteedCapacity.Key()].Leases, 0)

		filtered, err := q.filterUnleasedAccounts(gc)
		require.NoError(t, err)
		require.Len(t, filtered, 1)
		require.Len(t, filtered[0].Leases, 0)
	}

	// Test scanAndLeaseUnleasedAccounts
	{
		retry, err := q.scanAndLeaseUnleasedAccounts(ctx)
		require.NoError(t, err)
		require.False(t, retry)

		localLeases := q.getAccountLeases()
		require.Len(t, localLeases, 1)
		leaseId := localLeases[0].Lease
		leaseGc := localLeases[0].GuaranteedCapacity

		gc, err = q.getGuaranteedCapacityMap(ctx)
		require.Equal(t, leaseGc, gc[guaranteedCapacity.Key()])
		require.NoError(t, err)
		require.Len(t, gc, 1)
		require.NotNil(t, gc[guaranteedCapacity.Key()], "expected guaranteed capacity for account to be set", gc)
		require.Equal(t, []ulid.ULID{leaseId}, gc[guaranteedCapacity.Key()].Leases)
		require.Len(t, gc[guaranteedCapacity.Key()].Leases, 1)

		filtered, err := q.filterUnleasedAccounts(gc)
		require.NoError(t, err)
		require.Len(t, filtered, 0)

		err = q.expireAccountLease(ctx, leaseGc, leaseId)
		require.NoError(t, err, "expected lease to be expired", r.Dump(), leaseId, leaseGc)

		q.removeLeasedAccount(guaranteedCapacity)
	}

	// Test claimUnleasedGuaranteedCapacity
	{
		require.Len(t, q.getAccountLeases(), 0)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go q.claimUnleasedGuaranteedCapacity(ctx, time.Second, 2*time.Second)

		<-time.After(2 * time.Second)
		require.Len(t, q.getAccountLeases(), 1)

		cancel()
		<-time.After(2 * time.Second)
		require.Len(t, q.getAccountLeases(), 0)
	}
}
