package runner

import (
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/stretchr/testify/require"
	mathRand "math/rand"
	"testing"
)

func TestDeduplicateMigration(t *testing.T) {
	t.Run("it should not affect non-migrating functions", func(t *testing.T) {
		r := mathRand.New(mathRand.NewSource(1234))

		trigger := inngest.MultipleTriggers{
			inngest.Trigger{
				EventTrigger: &inngest.EventTrigger{
					Event: "test-event",
				},
			},
		}

		fnA := inngest.Function{
			Slug:     "app-1-fn-1",
			Triggers: trigger,
		}

		fnB := inngest.Function{
			Slug:     "app-2-fn-1",
			Triggers: trigger,
		}

		fns := []inngest.Function{fnA, fnB}

		output, err := deduplicateMigrations(r, fns)
		require.NoError(t, err)

		require.Len(t, output, 2)
		require.Equal(t, fns, output)
	})

	t.Run("it should deduplicate migrating functions", func(t *testing.T) {
		r := mathRand.New(mathRand.NewSource(1234))

		trigger := inngest.MultipleTriggers{
			inngest.Trigger{
				EventTrigger: &inngest.EventTrigger{
					Event: "test-event",
				},
			},
		}

		fnA := inngest.Function{
			Slug:     "app-1-fn-1",
			Triggers: trigger,
		}

		fnB := inngest.Function{
			Slug:     "app-2-fn-1",
			Triggers: trigger,
			Migrate: &inngest.Migrate{
				FromFunction:   "app-1-fn-1",
				RolloutPercent: 100,
			},
		}

		fns := []inngest.Function{fnA, fnB}

		output, err := deduplicateMigrations(r, fns)
		require.NoError(t, err)

		require.Len(t, output, 1)
		require.Contains(t, output, fnB)
	})

	t.Run("it should deduplicate migrating functions", func(t *testing.T) {
		r := mathRand.New(mathRand.NewSource(1234))

		trigger := inngest.MultipleTriggers{
			inngest.Trigger{
				EventTrigger: &inngest.EventTrigger{
					Event: "test-event",
				},
			},
		}

		fnA := inngest.Function{
			Slug:     "app-1-fn-1",
			Triggers: trigger,
		}

		fnB := inngest.Function{
			Slug:     "app-2-fn-1",
			Triggers: trigger,
			Migrate: &inngest.Migrate{
				FromFunction:   "app-1-fn-1",
				RolloutPercent: 50,
			},
		}

		fns := []inngest.Function{fnA, fnB}

		output, err := deduplicateMigrations(r, fns)
		require.NoError(t, err)

		require.Len(t, output, 1)
		require.Contains(t, output, fnA)
	})

	t.Run("it should deduplicate migrating functions", func(t *testing.T) {
		r := mathRand.New(mathRand.NewSource(1234))

		trigger := inngest.MultipleTriggers{
			inngest.Trigger{
				EventTrigger: &inngest.EventTrigger{
					Event: "test-event",
				},
			},
		}

		fnA := inngest.Function{
			Slug:     "app-1-fn-1",
			Triggers: trigger,
		}

		fnB := inngest.Function{
			Slug:     "app-2-fn-1",
			Triggers: trigger,
			Migrate: &inngest.Migrate{
				FromFunction:   "app-1-fn-1",
				RolloutPercent: 20,
			},
		}

		fns := []inngest.Function{fnA, fnB}

		output, err := deduplicateMigrations(r, fns)
		require.NoError(t, err)

		require.Len(t, output, 1)
		require.Contains(t, output, fnA)
	})
}
