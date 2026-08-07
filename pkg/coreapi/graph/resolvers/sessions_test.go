package resolvers

import (
	"testing"

	"github.com/inngest/inngest/pkg/headers"
	"github.com/stretchr/testify/require"
)

func TestRequireDevSessions(t *testing.T) {
	require.NoError(t, (&queryResolver{Resolver: &Resolver{ServerKind: headers.ServerKindDev}}).requireDevSessions())
	require.EqualError(
		t,
		(&queryResolver{Resolver: &Resolver{ServerKind: headers.ServerKindCloud}}).requireDevSessions(),
		"sessions are only available in inngest dev",
	)
}
