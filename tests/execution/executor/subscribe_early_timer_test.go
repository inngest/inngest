package executor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// The timer of a read is queued before the read is recorded. A timer that is
// due at once can run in between, find no read and be acknowledged. The read
// that is recorded afterwards must still end with its timeout.
func TestSubscribeTimerRunsBeforeTheReadIsRecorded(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))
	h.crash.hold("before:RecordInboxWait", "timer-ran")

	reported := make(chan struct{})
	go func() {
		defer close(reported)
		h.report(subscribeWaitTimeoutOp("next-1", "inbox", "inbox", "1ms"))
	}()

	require.Eventually(t, func() bool {
		return timesIn(h.jobIDs(), h.timerJobID("next-1")) == 1
	}, 5*time.Second, 5*time.Millisecond, "the timer was queued")

	time.Sleep(10 * time.Millisecond)
	require.NoError(t, h.runTimer(h.timerOf("next-1")))
	require.NoError(t, h.crash.fire("timer-ran"))
	<-reported

	require.Equal(t, "null", h.steps()["next-1"], "the read got its timeout")
	require.Equal(t, 1, timesIn(h.jobIDs(), h.timeoutJobID("next-1")), "the run was woken once")
}
