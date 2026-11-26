package loops

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/domain"
	"github.com/inngest/inngest/pkg/execution/state/kv"
	"github.com/oklog/ulid/v2"
)

const (
	// InboxNamespace is the kv namespace that prefixes every inbox list.
	InboxNamespace = "loop_inbox"

	// InboxTTL is the ttl that every write to an inbox sets.  it removes an
	// inbox that nothing deletes, for example the inbox of an instance that
	// expired.
	InboxTTL = consts.MaxWaitForEventTimeout
)

var (
	// ErrInboxFull means the inbox has no room for the event.  the event is
	// dropped and nothing is written.
	ErrInboxFull = errors.New("loop: inbox full")
	// ErrEmptyInboxName means the Inbox has no Name.
	ErrEmptyInboxName = errors.New("loop: inbox name must not be empty")
	// ErrNoInboxOpener means the Inbox has a zero OpenedBy.  every instance
	// record has an opener, so no run reads an inbox without one.
	ErrNoInboxOpener = errors.New("loop: inbox has no opener")
	// ErrInvalidInboxEvent means the event has a zero ID or is not valid
	// JSON.  nothing is written.
	ErrInvalidInboxEvent = errors.New("loop: invalid inbox event")
)

// InboxStore stores the unread events of loop inboxes.  each call is atomic
// for one inbox, so pushes and removes that run at the same time do not lose
// events.
type InboxStore interface {
	// Push adds the event to the end of the inbox.  an event with an ID that
	// is already in the inbox writes nothing and returns nil.  after the
	// event is removed, a push with the same ID adds it again.  ErrInboxFull
	// means the event would take the inbox past consts.MaxLoopInboxEvents
	// events or consts.MaxLoopInboxSize bytes, and the event is dropped.
	Push(context.Context, Inbox, InboxEvent) error
	// Peek returns the first limit events in the order they arrived, or all
	// of them when limit is 0 or less.  it does not write.  a missing inbox
	// returns no events and no error.
	Peek(ctx context.Context, in Inbox, limit int) ([]InboxEvent, error)
	// Remove removes the events with the given ids.  ids that are not in the
	// inbox are ignored and a missing inbox is not an error, so a remove
	// that is done again after a crash returns nil and writes nothing.
	Remove(ctx context.Context, in Inbox, ids []ulid.ULID) error
	// Delete deletes the inbox and all its events.  a missing inbox is not
	// an error.  a push that runs at the same time can create the inbox
	// again.
	Delete(context.Context, Inbox) error
}

// Inbox represents an inbox of events for a specific loop instance.
type Inbox struct {
	Loop ID
	// OpenedBy is the OpenedBy of the instance record that subscribed.  a
	// later instance of the same key has another opener, so it does not read
	// the events of an earlier instance.
	OpenedBy domain.RunID
	// Name is the inbox id that the SDK passed to loop.Subscribe.
	Name string
}

// InboxEvent is an unread event in an inbox.
type InboxEvent struct {
	// ID is the internal id of the event.  a push with an ID that is already
	// in the inbox writes nothing, so an event that is pushed twice before it
	// is read is in the inbox once.
	ID ulid.ULID
	// Event is the event as JSON, in the form the SDK receives it.  a store
	// returns the same bytes that were pushed.
	Event json.RawMessage
}

// kvKey returns the kv key for the inbox.  the key part is
// uuid5(InstanceID, OpenedBy + Name) as 32 hex characters.  OpenedBy is
// always 16 bytes, so two different pairs of opener and name never give the
// same hash input.
func (in Inbox) kvKey() (kv.Key, error) {
	if in.Name == "" {
		return kv.Key{}, ErrEmptyInboxName
	}
	if in.OpenedBy == (domain.RunID{}) {
		return kv.Key{}, ErrNoInboxOpener
	}
	name := make([]byte, 0, len(in.OpenedBy)+len(in.Name))
	name = append(name, in.OpenedBy[:]...)
	name = append(name, in.Name...)
	id := uuid.NewSHA1(in.Loop.InstanceID(), name)
	return kv.Key{
		Tenant:    in.Loop.Tenant,
		Namespace: InboxNamespace,
		Key:       hex.EncodeToString(id[:]),
	}, nil
}

// inboxStore keeps one kv list per inbox.  the item id is the event id as
// text and the item value is the event.
type inboxStore struct {
	lists kv.ListStore
}

func NewInboxStore(lists kv.ListStore) InboxStore {
	return &inboxStore{lists: lists}
}

func (s *inboxStore) Push(ctx context.Context, in Inbox, ev InboxEvent) error {
	key, err := in.kvKey()
	if err != nil {
		return err
	}
	if ev.ID == (ulid.ULID{}) || !json.Valid(ev.Event) {
		return ErrInvalidInboxEvent
	}
	item := kv.Item{
		ID:    ev.ID.String(),
		Value: ev.Event,
	}
	_, err = s.lists.Push(
		ctx,
		key,
		item,
		kv.WithMaxItems(consts.MaxLoopInboxEvents),
		kv.WithMaxBytes(consts.MaxLoopInboxSize),
		kv.WithTTL(InboxTTL),
	)
	if errors.Is(err, kv.ErrFull) {
		return ErrInboxFull
	}
	return err
}

func (s *inboxStore) Peek(ctx context.Context, in Inbox, limit int) ([]InboxEvent, error) {
	key, err := in.kvKey()
	if err != nil {
		return nil, err
	}
	items, err := s.lists.Peek(ctx, key, limit)
	if err != nil {
		return nil, err
	}
	events := make([]InboxEvent, len(items))
	for i, it := range items {
		id, err := ulid.ParseStrict(it.ID)
		if err != nil {
			return nil, fmt.Errorf("loop: decoding inbox event id %q: %w", it.ID, err)
		}
		events[i] = InboxEvent{
			ID:    id,
			Event: it.Value,
		}
	}
	return events, nil
}

func (s *inboxStore) Remove(ctx context.Context, in Inbox, ids []ulid.ULID) error {
	key, err := in.kvKey()
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	itemIDs := make([]string, len(ids))
	for i, id := range ids {
		itemIDs[i] = id.String()
	}
	return s.lists.Remove(ctx, key, itemIDs, kv.WithTTL(InboxTTL))
}

func (s *inboxStore) Delete(ctx context.Context, in Inbox) error {
	key, err := in.kvKey()
	if err != nil {
		return err
	}
	return s.lists.Clear(ctx, key)
}
