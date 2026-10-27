package devserver

import (
	"fmt"
	"testing"
	"time"

	apiv2 "github.com/inngest/inngest/pkg/api/v2"
	"github.com/inngest/inngest/pkg/cqrs"
	cqrsmanager "github.com/inngest/inngest/pkg/cqrs/manager"
	dbsqlite "github.com/inngest/inngest/pkg/db/sqlite"
	pb "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestEventProviderSQLite(t *testing.T) {
	db, err := dbsqlite.Open(t.Context(), dbsqlite.Options{ForTest: true})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	store := cqrsmanager.New(dbsqlite.New(db))
	provider := NewEventProvider(store)
	svc := apiv2.NewService(apiv2.ServiceOptions{Events: provider})
	now := time.Now().UTC().Truncate(time.Millisecond)
	ids := make([]ulid.ULID, 9)
	names := []string{"order/placed", "order/paid", "order/placed", "order/paid", "order/placed", "order/paid", "other", "inngest/function.finished", "order/placed"}
	for i, name := range names {
		id := ulid.MustParse(fmt.Sprintf("01J0000000000000000000000%d", i))
		ids[i] = id
		require.NoError(t, store.InsertEvent(t.Context(), cqrs.Event{ID: id, EventName: name, EventID: fmt.Sprintf("sender-%d", i), EventTS: now.UnixMilli(), EventData: map[string]any{"position": i}}))
		received := now
		if i == 8 {
			received = now.Add(-2 * time.Hour)
		}
		_, err = db.ExecContext(t.Context(), "UPDATE events SET received_at = ? WHERE internal_id = ?", received, id)
		require.NoError(t, err)
	}
	from, until := timestamppb.New(now.Add(-time.Hour)), timestamppb.New(now)
	var got []string
	var cursor *string
	for page := 0; page < 3; page++ {
		result, err := svc.ListEvents(t.Context(), &pb.ListEventsRequest{Limit: new(int32(2)), Cursor: cursor, Name: []string{"order/placed", "order/paid"}, From: from, Until: until})
		require.NoError(t, err)
		require.Len(t, result.Data, 2)
		require.Equal(t, page < 2, result.Page.HasMore)
		for _, evt := range result.Data {
			got = append(got, evt.Id)
			require.Equal(t, now, evt.ReceivedAt.AsTime())
			require.Nil(t, evt.Payload)
		}
		cursor = result.Page.Cursor
		if page < 2 {
			require.Equal(t, result.Data[1].Id, *cursor)
		} else {
			require.Nil(t, cursor)
		}
	}
	require.Equal(t, []string{ids[5].String(), ids[4].String(), ids[3].String(), ids[2].String(), ids[1].String(), ids[0].String()}, got)
	found, err := provider.GetEvent(t.Context(), ids[3])
	require.NoError(t, err)
	require.Equal(t, "order/paid", found.EventName)
	require.Equal(t, "sender-3", found.EventID)
	_, err = provider.GetEvent(t.Context(), ulid.Make())
	require.ErrorIs(t, err, apiv2.ErrEventNotFound)

	all, err := svc.ListEvents(t.Context(), &pb.ListEventsRequest{From: from, Until: until, Include: []string{"payload"}})
	require.NoError(t, err)
	require.Len(t, all.Data, 7)
	require.Equal(t, ids[6].String(), all.Data[0].Id)
	require.Equal(t, float64(6), all.Data[0].Payload.AsMap()["data"].(map[string]any)["position"])
	require.Nil(t, all.Data[0].Source)
	require.Nil(t, all.Page.Cursor)
	require.False(t, all.Page.HasMore)
	for _, evt := range all.Data {
		require.NotEqual(t, "inngest/function.finished", evt.Name)
	}

	// seeded ids can be older than their received time
	_, err = db.ExecContext(t.Context(), "UPDATE events SET received_at = ? WHERE internal_id = ?", now.Add(time.Minute), ids[0])
	require.NoError(t, err)
	ordered, err := svc.ListEvents(t.Context(), &pb.ListEventsRequest{From: from, Until: timestamppb.New(now.Add(time.Minute))})
	require.NoError(t, err)
	require.Equal(t, ids[6].String(), ordered.Data[0].Id)
	require.Equal(t, ids[0].String(), ordered.Data[6].Id)
	require.True(t, ordered.Data[6].ReceivedAt.AsTime().After(ordered.Data[0].ReceivedAt.AsTime()))
}
