package queue

const (
	// KindStart represents a queue state that the function state has been created but not started yet.
	// Essentially a status that represents the backlog.
	KindStart           = "start"
	KindEdge            = "edge"
	KindSleep           = "sleep"
	KindPause           = "pause"
	KindDebounce        = "debounce"
	KindScheduleBatch   = "schedule-batch"
	KindEdgeError       = "edge-error" // KindEdgeError is used to indicate a final step error attempting a graceful save.
	KindQueueMigrate    = "queue-migrate"
	KindPauseBlockFlush = "pbf"    // Flushes pauses from the buffer to blocks.
	KindJobPromote      = "jps"    // job promotion service
	KindCancel          = "cancel" // cancel signals eager cancellation of queue items
	KindFunctionPause   = "fp"     // pause function
	KindFunctionUnpause = "fup"    // unpause function
)
