package executor

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
)

// A step.subscribe subscription keeps its pause when an event matches and saves
// the event as a step of the run. The step ID is made of the subscription and
// the event ID, so saving the same event twice is one message and two events
// never contend for a slot. The run's stack (the order steps were saved in) is
// the order of the messages: the SDK reads them from there.

// subscriptionMessagePrefix starts the ID of every message step. The SDK looks
// for it in the stack.
const subscriptionMessagePrefix = "inngest-sub:"

func subscriptionKeyHash(key string) string {
	sum := sha1.Sum([]byte(key))
	return hex.EncodeToString(sum[:])
}

// subscriptionMessageID is the step ID of the message an event is in a
// subscription.
func subscriptionMessageID(key string, eventID string) string {
	return subscriptionMessagePrefix + subscriptionKeyHash(key) + ":" + eventID
}

// subscriptionMessageIDs returns the message step IDs of a subscription in
// stack order.
func subscriptionMessageIDs(stack []string, key string) []string {
	prefix := subscriptionMessagePrefix + subscriptionKeyHash(key) + ":"

	var ids []string
	for _, id := range stack {
		if strings.HasPrefix(id, prefix) {
			ids = append(ids, id)
		}
	}

	return ids
}

// subscriptionClosedID is the step that marks a subscription as closed. It
// starts with the message prefix, so the SDK does not wait for it, and its hash
// part is not a message of any subscription.
func subscriptionClosedID(key string) string {
	return subscriptionMessagePrefix + "closed:" + subscriptionKeyHash(key)
}

// subscriptionStepID is the step ID of the wait for message seq. The SDK gives
// its blocking next() the same ID.
func subscriptionStepID(key string, reader string, seq int, attempt int) string {
	id := key
	if reader != "" {
		id = fmt.Sprintf("%s:%s", key, reader)
	}
	id = fmt.Sprintf("%s:%d", id, seq)
	if attempt > 0 {
		id = fmt.Sprintf("%s:%d", id, attempt)
	}
	sum := sha1.Sum([]byte(id))
	return hex.EncodeToString(sum[:])
}

// subscriptionParkIndexName names the index of the parked markers of one run
// and subscription. It starts with consts.SubscriptionParkName. A delivery reads
// this one small index to find the marker without reading the run's stack.
func subscriptionParkIndexName(runID string, key string) string {
	return consts.SubscriptionParkName + ":" + runID + ":" + subscriptionKeyHash(key)
}

// subscriptionParkID is the ID of the marker of one park. It is made of the park
// step's ID, which is unique to the park, so the cleanup of an earlier marker can
// never delete a later one, however many messages were saved in between.
func subscriptionParkID(runID string, key string, stepID string) uuid.UUID {
	return inngest.DeterministicSha1UUID(fmt.Sprintf("%s:subscription-park:%s:%s", runID, key, stepID))
}

// subscriptionExpiryJobID is the queue job that expires a subscription pause.
func subscriptionExpiryJobID(md *sv2.Metadata, pauseID uuid.UUID) string {
	return fmt.Sprintf("%s-subscription-expiry-%s", md.IdempotencyKey(), pauseID)
}

// subscriptionPauseID is the ID of the persistent pause of one event (stream) of a
// subscription.
func subscriptionPauseID(runID string, key string, stream int) uuid.UUID {
	return inngest.DeterministicSha1UUID(fmt.Sprintf("%s:subscription:%s:%d", runID, key, stream))
}

// isSubscriptionParkPause reports whether a pause is the one a run blocks on while it
// waits for a subscription message. It has no timeout job.
func isSubscriptionParkPause(pause *state.Pause) bool {
	return pause.Event != nil && strings.HasPrefix(*pause.Event, consts.SubscriptionParkName)
}
