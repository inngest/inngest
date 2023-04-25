package defaults

import (
	// Import the default drivers, queues, and state stores.
	_ "github.com/inngest/inngest/pkg/coredata/inmemory"
	_ "github.com/inngest/inngest/pkg/coredata/postgres"
	_ "github.com/inngest/inngest/pkg/execution/driver/httpdriver"
	_ "github.com/inngest/inngest/pkg/execution/driver/mockdriver"
	_ "github.com/inngest/inngest/pkg/execution/queue/inmemoryqueue"
	_ "github.com/inngest/inngest/pkg/execution/queue/sqsqueue"
	_ "github.com/inngest/inngest/pkg/execution/state/inmemory"
	_ "github.com/inngest/inngest/pkg/execution/state/redis_state"
)
