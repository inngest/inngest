package function

import (
	"context"
	"fmt"
	"strings"

	"github.com/gosimple/slug"
	"github.com/inngest/inngest/inngest"
	"github.com/inngest/inngest/inngest/clistate"
)

var (
	defaultRuntime = inngest.RuntimeWrapper{Runtime: inngest.RuntimeDocker{}}
	defaultAfter   = After{Step: inngest.TriggerName}
)

func DefaultRuntime() *inngest.RuntimeWrapper {
	copied := defaultRuntime
	return &copied
}

// Step represents a single unit of code (action) which runs as part of a step function, in a DAG.
type Step struct {
	ID      string                     `json:"id"`
	Path    string                     `json:"path"`
	Name    string                     `json:"name"`
	Runtime *inngest.RuntimeWrapper    `json:"runtime,omitempty"`
	After   []After                    `json:"after,omitempty"`
	Version *inngest.VersionConstraint `json:"version,omitempty"`
	Retries *inngest.RetryOptions      `json:"retries,omitempty"`
}

func (s Step) DSN(ctx context.Context, f Function) string {
	suffix := "test"
	if clistate.IsProd() {
		suffix = "prod"
	}

	slug := strings.ToLower(slug.Make(s.ID))

	id := fmt.Sprintf("%s-step-%s-%s", f.ID, slug, suffix)
	if prefix, err := clistate.AccountIdentifier(ctx); err == nil && prefix != "" {
		id = fmt.Sprintf("%s/%s", prefix, id)
	}

	return id
}
