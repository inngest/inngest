package kv

import (
	"context"
	"encoding/json"
	"errors"
)

var (
	// ErrFull is returned by Push when the item would take the list past
	// WithMaxItems or WithMaxBytes.  nothing is written.
	ErrFull = errors.New("kv: list full")

	ErrInvalidItemID = errors.New("kv: item id must be 1 to 32 letters, digits, '-', and '_'")
)

// ListStore stores bounded lists of items.  each key is one list.  calls on
// one list take effect one at a time, so a Push checks the ID and both
// limits against every item that earlier calls left in the list.  a list
// that expired reads as missing, and its items do not count against a
// limit.  lists and Store values do not share keys, so the same Key in a
// Store and in a ListStore names two different things.
type ListStore interface {
	// Push adds the item to the end of the list and returns true.  an item
	// with an ID that is already in the list writes nothing and returns
	// false and no error.  ErrFull means the item would take the list past
	// WithMaxItems or WithMaxBytes, and nothing is written.  a push that
	// adds the item sets the expiry of the list from WithTTL.  without
	// WithTTL the list does not expire, and an earlier TTL is removed.
	Push(ctx context.Context, k Key, it Item, opts ...Option) (bool, error)
	// Peek returns the first limit items in the order they were pushed, or
	// all of them when limit is 0 or less.  it does not write.  a missing
	// list returns no items and no error.
	Peek(ctx context.Context, k Key, limit int) ([]Item, error)
	// Remove removes the items with the given ids.  ids that are not in the
	// list are ignored, and a missing list is not an error.  a remove that
	// removes an item sets the expiry of the list the same way a Push does.
	Remove(ctx context.Context, k Key, ids []string, opts ...Option) error
	// Clear deletes the list and all its items.  a missing list is not an
	// error.
	Clear(ctx context.Context, k Key) error
}

// Item is one value in a list.
type Item struct {
	// ID names the item in its list.  it follows the rules of a key part,
	// which ValidateItemID checks.
	ID string
	// Value is opaque bytes.  a store does not parse or change it, and Peek
	// returns exactly the bytes that were pushed.  WithMaxBytes counts
	// len(Value).
	Value json.RawMessage
}

// ValidateItemID checks that the id is non-empty, at most MaxPartLen bytes,
// and contains only ASCII letters, digits, '-', and '_'.
func ValidateItemID(id string) error {
	if id == "" || len(id) > MaxPartLen || !isIdent(id) {
		return ErrInvalidItemID
	}
	return nil
}
