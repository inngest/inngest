package executor

import (
	"context"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestRunValidatorTimeouts(t *testing.T) {
	anHourAgo := time.Now().Add(-time.Hour)

	for _, tc := range []struct {
		name     string
		timeouts *inngest.Timeouts
		started  bool
		stopped  bool
	}{
		{name: "no timeouts"},
		{name: "unset timeouts", timeouts: &inngest.Timeouts{}, started: true},

		{name: "empty start timeout", timeouts: &inngest.Timeouts{Start: new("")}},
		{name: "unparseable start timeout", timeouts: &inngest.Timeouts{Start: new("1 hour")}},
		{name: "zero start timeout", timeouts: &inngest.Timeouts{Start: new("0s")}},
		{name: "start timeout not reached", timeouts: &inngest.Timeouts{Start: new("2h")}},
		{name: "start timeout reached", timeouts: &inngest.Timeouts{Start: new("1s")}, stopped: true},
		{name: "start timeout reached by a started run", timeouts: &inngest.Timeouts{Start: new("1s")}, started: true},

		{name: "empty finish timeout", timeouts: &inngest.Timeouts{Finish: new("")}, started: true},
		{name: "unparseable finish timeout", timeouts: &inngest.Timeouts{Finish: new("1 hour")}, started: true},
		{name: "zero finish timeout", timeouts: &inngest.Timeouts{Finish: new("0s")}, started: true},
		{name: "finish timeout not reached", timeouts: &inngest.Timeouts{Finish: new("2h")}, started: true},
		{name: "finish timeout reached", timeouts: &inngest.Timeouts{Finish: new("1s")}, started: true, stopped: true},
		{name: "finish timeout on a run that has not started", timeouts: &inngest.Timeouts{Finish: new("1s")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			md := sv2.Metadata{
				ID: sv2.ID{RunID: ulid.MustNew(ulid.Timestamp(anHourAgo), ulid.DefaultEntropy())},
			}
			if tc.started {
				md.Config.StartedAt = anHourAgo
			}

			v := &runValidator{
				md: md,
				f:  &inngest.Function{Timeouts: tc.timeouts},
				e: &executor{
					log: logger.VoidLogger(),
					// Cancelling a run that's already gone succeeds without
					// needing any further executor dependencies.
					smv2: &missingMetadataRunService{err: state.ErrRunNotFound},
				},
			}

			require.NoError(t, v.checkStartTimeout(context.Background()))
			require.NoError(t, v.checkFinishTimeout(context.Background()))
			require.Equal(t, tc.stopped, v.stopWithoutRetry)
		})
	}
}
