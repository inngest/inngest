package queue

import "context"

// QueueItemIndex represends a set of indexes for a given queue item.  We currently allow
// up to 2 indexes per job item to be created.
//
// # What is an index?
//
// An index is a sorted ZSET of job items for a given key.  The ZSET stores all
// oustanding AND in-progress job IDs, scored by job time in milliseconds. Because this
// stores outstanding and in progress jobs, this _cannot_ be used to control concurrency.
// It is used to specifically list all jobs that exist for given keys for transparency.
//
// A nil slice or empty strings within the slice indicate nil indexes, ie. an index
// will not be created.
type QueueItemIndex [2]string

// QueueItemIndexer represents a function which generates indexes for a given queue item.
type QueueItemIndexer func(ctx context.Context, i QueueItem) QueueItemIndex
