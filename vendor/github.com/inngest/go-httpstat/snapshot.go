package httpstat

import "time"

// Snapshot reports the transport phases observed so far, including on failed requests.
// Durations reflect the timing callbacks that have run.
type Snapshot struct {
	DNSLookup       time.Duration
	TCPConnection   time.Duration
	TLSHandshake    time.Duration
	TimeToFirstByte time.Duration
	TLSStarted      bool
	TLSCompleted    bool
	RequestWritten  bool
	FirstByte       bool
}

// Snapshot copies the trace while holding the same lock as its callbacks.
func (r *Result) Snapshot() Snapshot {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.progress == nil {
		return Snapshot{}
	}
	return *r.progress
}
