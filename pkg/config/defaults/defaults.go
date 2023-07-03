package defaults

import (
	// Import the default drivers, queues, and state stores.
	_ "github.com/inngest/inngest/pkg/execution/driver/httpdriver"
	_ "github.com/inngest/inngest/pkg/execution/driver/mockdriver"
	_ "github.com/inngest/inngest/pkg/execution/state/redis_state"
)
