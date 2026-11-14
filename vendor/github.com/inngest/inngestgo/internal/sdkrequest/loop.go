package sdkrequest

import (
	"encoding/json"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/inngest/inngestgo/internal/opcode"
	str2duration "github.com/xhit/go-str2duration/v2"
)

// LoopCollector gathers the wake, end, subscribe, and unsubscribe calls a
// loop handler makes during one run.  none of the calls hijack control.
// the SDK sends the result in the LoopReturn opcode when the loop handler
// returns nil and no step hijacked the run.
type LoopCollector struct {
	mu          sync.Mutex
	wake        *opcode.LoopWake
	end         bool
	subscribe   map[string]opcode.LoopSubscribe
	unsubscribe map[string]struct{}
}

func newLoopCollector() *LoopCollector {
	return &LoopCollector{
		subscribe:   map[string]opcode.LoopSubscribe{},
		unsubscribe: map[string]struct{}{},
	}
}

// Wake schedules the next run.  the last call in a run wins.
func (c *LoopCollector) Wake(id string, after time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.wake = &opcode.LoopWake{
		ID:    id,
		After: str2duration.String(after),
	}
}

// End finishes the instance.  it drops any wake from the result.
func (c *LoopCollector) End() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.end = true
}

// Subscribe records a subscription to event, filtered by expr.  a repeated
// id in one run keeps the last call.  it also drops an Unsubscribe for the
// same id made earlier in this run.  a zero ttl sends no ttl.  loop.Subscribe
// rejects a negative ttl before this call.
func (c *LoopCollector) Subscribe(id string, event, expr string, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.unsubscribe, id)
	sub := opcode.LoopSubscribe{
		ID:    id,
		Event: event,
		If:    expr,
	}
	if ttl > 0 {
		sub.TTL = str2duration.String(ttl)
	}
	c.subscribe[id] = sub
}

// Unsubscribe removes a subscription.  it also drops a Subscribe for the
// same id made earlier in this run.
func (c *LoopCollector) Unsubscribe(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.subscribe, id)
	c.unsubscribe[id] = struct{}{}
}

// Result builds the LoopReturn payload for the given state.  version is
// the version from the request's loop block and is sent back unchanged.
// subscribe entries and unsubscribe ids are sorted by id so the payload
// is stable.
func (c *LoopCollector) Result(state json.RawMessage, version int64) opcode.LoopReturn {
	c.mu.Lock()
	defer c.mu.Unlock()

	ret := opcode.LoopReturn{
		State:   state,
		Version: version,
		End:     c.end,
	}
	if !c.end && c.wake != nil {
		wake := *c.wake
		ret.Wake = &wake
	}
	for _, sub := range c.subscribe {
		ret.Subscribe = append(ret.Subscribe, sub)
	}
	slices.SortFunc(ret.Subscribe, func(a, b opcode.LoopSubscribe) int {
		return strings.Compare(a.ID, b.ID)
	})
	for id := range c.unsubscribe {
		ret.Unsubscribe = append(ret.Unsubscribe, id)
	}
	slices.Sort(ret.Unsubscribe)
	return ret
}
