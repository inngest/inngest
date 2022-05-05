package runtime

import (
	"context"

	"github.com/inngest/inngestctl/inngest"
)

type Executor interface {
	Execute(ctx context.Context, action inngest.ActionVersion, state map[string]interface{}) (map[string]interface{}, error)
}
