// Copyright (c) 2023, Peter Ohler, All rights reserved.

package jp

// Keyed describes an interface for a collection that is indexed by a string
// key similar to a map[string]any.
type Keyed interface {

	// ValueForKey should return the value associated with the key or nil if
	// no entry exists for the key.
	ValueForKey(key string) (value any, has bool)

	// SetValueForKey sets the value for a key in the collection.
	SetValueForKey(key string, value any)

	// RemoveValueForKey removes the value for a key in the collection.
	RemoveValueForKey(key string)

	// Keys should return an list of the keys for all the entries in the
	// collection.
	Keys() []string
}
