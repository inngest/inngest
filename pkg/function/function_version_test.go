package function

import (
	"context"
	"testing"

	"github.com/inngest/inngest/inngest"
	"github.com/stretchr/testify/require"
)

func TestActions(t *testing.T) {
	fn := Function{
		ID:   "hi",
		Name: "test",
		Triggers: []Trigger{{
			EventTrigger: &EventTrigger{
				Event: "test/foo.bar",
			},
		}},
		Steps: map[string]Step{
			"first": {
				ID:   "first",
				Name: "first",
				Runtime: inngest.RuntimeWrapper{
					Runtime: &stubdriver{},
				},
			},
			"second": {
				ID:   "second",
				Name: "second",
				Runtime: inngest.RuntimeWrapper{
					Runtime: &stubdriver{},
				},
			},
		},
	}
	err := fn.Validate(context.Background())
	require.NoError(t, err)

	fv := FunctionVersion{
		Function: fn,
		Version:  2,
	}
	avs, _, err := fv.Function.Actions(context.Background())
	require.NoError(t, err)

	require.NotNil(t, avs[0].Version)
	require.Equal(t, uint(1), avs[0].Version.Major)
	require.Equal(t, uint(2), avs[0].Version.Minor)
	require.Equal(t, uint(1), avs[1].Version.Major)
	require.Equal(t, uint(2), avs[1].Version.Minor)
}
