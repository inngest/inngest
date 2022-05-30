package edges

#Edge: (#SyncEdge | #AsyncEdge)

#SyncEdge: close({
	type:  "edge"
	name?: string
	if?:   string
	wait?: string
})

// An AsyncEdge represents an edge that can be traversed at some future point in time
// as soon as an event is received that matches the given expression.
#AsyncEdge: close({
	type:  "async"
	name?: string
	if?:   string
	async: close({
		ttl:    string
		event:  string
		match?: string
		// onTimeout specifies that this edge should be traversed on timeout only,
		// if the event is not received within the TTL.
		onTimeout?: bool
	})
})
