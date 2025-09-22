package loader

import (
	"context"
	"fmt"

	"github.com/graph-gophers/dataloader"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/oklog/ulid/v2"
)

type eventReader struct {
	loaders *Loaders
	reader  cqrs.EventReader
}

func (er *eventReader) GetEvents(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	results := make([]*dataloader.Result, len(keys))

	// Give an initial length of 0 because we could possibly end up with fewer than len(keys) if any are
	// invalid ULIDs
	eventIds := make([]ulid.ULID, 0, len(keys))
	for _, key := range keys.Keys() {
		eventId, err := ulid.Parse(key)
		if err != nil {
			// Invalid ULID, but we will set results entry accordingly later
			continue
		}
		eventIds = append(eventIds, eventId)
	}

	events, err := er.reader.GetEventsByInternalIDs(ctx, eventIds)
	if err != nil {
		// The whole read from the DB failed, so return the same error for all of them
		for i := range results {
			results[i] = &dataloader.Result{
				Error: err,
			}
		}
		return results
	}

	eventMap := make(map[string]*cqrs.Event, len(events))
	for _, event := range events {
		eventMap[event.InternalID().String()] = event
	}

	// Need to iterate over keys again to maintain same order in return slice
	for i, eventId := range keys.Keys() {
		if event, found := eventMap[eventId]; found {
			results[i] = &dataloader.Result{
				Data:  event,
				Error: nil,
			}
		} else {
			results[i] = &dataloader.Result{
				Data:  nil,
				Error: fmt.Errorf("event not found: %s", eventId),
			}
		}
	}

	return results
}
