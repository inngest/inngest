package defaults

import (
	// Import the default drivers, queues, and state stores.
	_ "github.com/inngest/inngest-cli/pkg/execution/driver/dockerdriver"
	_ "github.com/inngest/inngest-cli/pkg/execution/driver/httpdriver"
	_ "github.com/inngest/inngest-cli/pkg/execution/driver/mockdriver"
	_ "github.com/inngest/inngest-cli/pkg/execution/queue/inmemoryqueue"
	_ "github.com/inngest/inngest-cli/pkg/execution/queue/sqsqueue"
	_ "github.com/inngest/inngest-cli/pkg/execution/state/inmemory"
	_ "github.com/inngest/inngest-cli/pkg/execution/state/redis_state"
)
