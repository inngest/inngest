package batcher

import (
	"context"
	"sync"
	"time"
)

const (
	phaseUninitialized = iota
	phaseStarted
	phasePaused
	phaseStopped
)

type Batcher interface {
	Eventer
	WithRateLimiter(rl RateLimiter) Batcher
	WithFlushInterval(val time.Duration) Batcher
	WithCapacityInterval(val time.Duration) Batcher
	WithAuditInterval(val time.Duration) Batcher
	WithMaxOperationTime(val time.Duration) Batcher
	WithPauseTime(val time.Duration) Batcher
	WithErrorOnFullBuffer() Batcher
	WithEmitBatch() Batcher
	WithEmitFlush() Batcher
	WithEmitRequest() Batcher
	WithMaxConcurrentBatches(val uint32) Batcher
	Enqueue(op Operation) error
	Pause()
	Flush()
	Inflight() uint32
	OperationsInBuffer() uint32
	NeedsCapacity() uint32
	Start(ctx context.Context) (err error)
}

type batcher struct {
	EventerBase

	// configuration items that should not change after Start()
	ratelimiter          RateLimiter
	flushInterval        time.Duration
	capacityInterval     time.Duration
	auditInterval        time.Duration
	maxOperationTime     time.Duration
	pauseTime            time.Duration
	errorOnFullBuffer    bool
	emitBatch            bool
	emitFlush            bool
	emitRequest          bool
	maxConcurrentBatches uint32

	// used for internal operations
	buffer               ibuffer       // operations that are in the queue
	pause                chan struct{} // contains a record if batcher is paused
	flush                chan struct{} // contains a record if batcher should flush
	inflight             chan struct{} // tracks the number of inflight batches
	lastFlushWithRecords time.Time     // tracks the last time records were flushed

	// manage the phase
	phaseMutex sync.Mutex
	phase      int

	// target needs to be threadsafe and changes frequently
	targetMutex sync.RWMutex
	target      uint32
}

// This method creates a new Batcher with a buffer that can contain up to 10,000 Operations. Generally you should have 1 Batcher per datastore.
// Commonly after calling NewBatcher() you will chain some WithXXXX methods, for instance... `NewBatcher().WithRateLimiter(limiter)`.
func NewBatcher() Batcher {
	return NewBatcherWithBuffer(10000)
}

// This method creates a new Batcher with a buffer that can contain up to a user-defined number of Operations. Generally you should have 1
// Batcher per datastore. Commonly after calling NewBatcherWithBuffer() you will chain some WithXXXX methods, for instance...
// `NewBatcherWithBuffer().WithRateLimiter(limiter)`.
func NewBatcherWithBuffer(maxBufferSize uint32) Batcher {
	r := &batcher{}
	r.buffer = newBuffer(maxBufferSize)
	r.pause = make(chan struct{}, 1)
	r.flush = make(chan struct{}, 1)
	return r
}

// Use SharedResource as a rate limiter with Batcher to throttle the requests made against a datastore. This is
// optional; the default behavior does not rate limit.
func (r *batcher) WithRateLimiter(rl RateLimiter) Batcher {
	r.phaseMutex.Lock()
	defer r.phaseMutex.Unlock()
	if r.phase != phaseUninitialized {
		panic(InitializationOnlyError)
	}
	r.ratelimiter = rl
	return r
}

// The FlushInterval determines how often the processing loop attempts to flush buffered Operations. The default is `100ms`. If a rate limiter
// is being used, the interval determines the capacity that each flush has to work with. For instance, with the default 100ms and 10,000
// available capacity, there would be 10 flushes per second, each dispatching one or more batches of Operations that aim for 1,000 total
// capacity. If no rate limiter is used, each flush will attempt to empty the buffer.
func (r *batcher) WithFlushInterval(val time.Duration) Batcher {
	r.phaseMutex.Lock()
	defer r.phaseMutex.Unlock()
	if r.phase != phaseUninitialized {
		panic(InitializationOnlyError)
	}
	r.flushInterval = val
	return r
}

// The CapacityInterval determines how often the processing loop asks the rate limiter for capacity by calling GiveMe(). The default is
// `100ms`. The Batcher asks for capacity equal to every Operation's cost that has not been marked done. In other words, when you Enqueue()
// an Operation it increments a target based on cost. When you call done() on a batch (or the MaxOperationTime is exceeded), the target is
// decremented by the cost of all Operations in the batch. If there is no rate limiter attached, this interval does nothing.
func (r *batcher) WithCapacityInterval(val time.Duration) Batcher {
	r.phaseMutex.Lock()
	defer r.phaseMutex.Unlock()
	if r.phase != phaseUninitialized {
		panic(InitializationOnlyError)
	}
	r.capacityInterval = val
	return r
}

// The AuditInterval determines how often the target capacity is audited to ensure it still seems legitimate. The default is `10s`. The
// target capacity is the amount of capacity the Batcher thinks it needs to process all outstanding Operations. Only atomic operatios are
// performed on the target and there are other failsafes such as MaxOperationTime, however, since it is critical that the target capacity
// be correct, this is one final failsafe to ensure the Batcher isn't asking for the wrong capacity. Generally you should leave this set
// at the default.
func (r *batcher) WithAuditInterval(val time.Duration) Batcher {
	r.phaseMutex.Lock()
	defer r.phaseMutex.Unlock()
	if r.phase != phaseUninitialized {
		panic(InitializationOnlyError)
	}
	r.auditInterval = val
	return r
}

// The MaxOperationTime determines how long Batcher waits until marking a batch done after releasing it to the Watcher. The default is `1m`.
// You should always call the done() func when your batch has completed processing instead of relying on MaxOperationTime. The MaxOperationTime
// on Batcher will be superceded by MaxOperationTime on Watcher if provided.
func (r *batcher) WithMaxOperationTime(val time.Duration) Batcher {
	r.phaseMutex.Lock()
	defer r.phaseMutex.Unlock()
	if r.phase != phaseUninitialized {
		panic(InitializationOnlyError)
	}
	r.maxOperationTime = val
	return r
}

// The PauseTime determines how long Batcher suspends the processing loop once Pause() is called. The default is `500ms`. Typically, Pause()
// is called because errors are being received from the datastore such as TooManyRequests or Timeout. Pausing hopefully allows the datastore
// to catch up without making the problem worse.
func (r *batcher) WithPauseTime(val time.Duration) Batcher {
	r.phaseMutex.Lock()
	defer r.phaseMutex.Unlock()
	if r.phase != phaseUninitialized {
		panic(InitializationOnlyError)
	}
	r.pauseTime = val
	return r
}

// Setting this option changes Enqueue() such that it throws an error if the buffer is full. Normal behavior is for the Enqueue() func to
// block until it is able to add to the buffer.
func (r *batcher) WithErrorOnFullBuffer() Batcher {
	r.phaseMutex.Lock()
	defer r.phaseMutex.Unlock()
	if r.phase != phaseUninitialized {
		panic(InitializationOnlyError)
	}
	r.errorOnFullBuffer = true
	return r
}

// DO NOT SET THIS IN PRODUCTION. For unit tests, it may be beneficial to raise an event for each batch of operations.
func (r *batcher) WithEmitBatch() Batcher {
	r.phaseMutex.Lock()
	defer r.phaseMutex.Unlock()
	if r.phase != phaseUninitialized {
		panic(InitializationOnlyError)
	}
	r.emitBatch = true
	return r
}

// Generally you do not want this setting for production, but it can be helpful for unit tests to raise an event every time
// a flush is started and completed.
func (r *batcher) WithEmitFlush() Batcher {
	r.emitFlush = true
	return r
}

// Generally you do not want this setting for production, but it can be helpful for unit tests to raise an event every time
// a request is made for capacity.
func (r *batcher) WithEmitRequest() Batcher {
	r.emitRequest = true
	return r
}

// Setting this option limits the number of batches that can be processed at a time to the provided value.
func (r *batcher) WithMaxConcurrentBatches(val uint32) Batcher {
	r.maxConcurrentBatches = val
	r.inflight = make(chan struct{}, val)
	return r
}

func (r *batcher) applyDefaults() {
	if r.flushInterval <= 0 {
		r.flushInterval = 100 * time.Millisecond
	}
	if r.capacityInterval <= 0 {
		r.capacityInterval = 100 * time.Millisecond
	}
	if r.auditInterval <= 0 {
		r.auditInterval = 10 * time.Second
	}
	if r.maxOperationTime <= 0 {
		r.maxOperationTime = 1 * time.Minute
	}
	if r.pauseTime <= 0 {
		r.pauseTime = 500 * time.Millisecond
	}
}

// Call this method to add an Operation into the buffer.
func (r *batcher) Enqueue(op Operation) error {

	// ensure an operation was provided
	if op == nil {
		return NoOperationError
	}

	// ensure there is a watcher associated with the call
	watcher := op.Watcher()
	if op.Watcher() == nil {
		return NoWatcherError
	}

	// ensure the cost doesn't exceed max capacity
	if r.ratelimiter != nil && op.Cost() > r.ratelimiter.MaxCapacity() {
		return TooExpensiveError
	}

	// ensure there are not too many attempts
	maxAttempts := watcher.MaxAttempts()
	if maxAttempts > 0 && op.Attempt() >= maxAttempts {
		return TooManyAttemptsError
	}

	// increment the target
	r.incTarget(int(op.Cost()))

	// put into the buffer
	return r.buffer.enqueue(op, r.errorOnFullBuffer)
}

// Call this method when your datastore is throwing transient errors. This pauses the processing loop to ensure that you are not flooding
// the datastore with additional data it cannot process making the situation worse.
func (r *batcher) Pause() {

	// ensure pausing only happens when it is running
	r.phaseMutex.Lock()
	defer r.phaseMutex.Unlock()
	if r.phase != phaseStarted {
		// simply ignore an invalid pause
		return
	}

	// pause
	select {
	case r.pause <- struct{}{}:
		// successfully set the pause
	default:
		// pause was already set
	}

	// switch to paused phase
	r.phase = phasePaused

}

func (r *batcher) resume() {
	r.phaseMutex.Lock()
	defer r.phaseMutex.Unlock()
	if r.phase == phasePaused {
		r.phase = phaseStarted
	}
}

// Call this method to manually flush as if the flushInterval were triggered.
func (r *batcher) Flush() {

	// flush
	select {
	case r.flush <- struct{}{}:
		// successfully set the flush
	default:
		// flush was already set
	}

}

// This tells you how many operations are still in the buffer. This does not include operations that have been sent back to the Watcher as part
// of a batch for processing.
func (r *batcher) OperationsInBuffer() uint32 {
	return r.buffer.size()
}

// This tells you how much capacity the Batcher believes it needs to process everything outstanding. Outstanding operations include those in
// the buffer and operations and any that have been sent as a batch but not marked done yet.
func (r *batcher) NeedsCapacity() uint32 {
	r.targetMutex.RLock()
	defer r.targetMutex.RUnlock()
	return r.target
}

func (r *batcher) confirmTargetIsZero() bool {
	r.targetMutex.Lock()
	defer r.targetMutex.Unlock()
	if r.target > 0 {
		r.target = 0
		return false
	} else {
		return true
	}
}

func (r *batcher) incTarget(val int) {
	r.targetMutex.Lock()
	defer r.targetMutex.Unlock()
	if val < 0 && r.target >= uint32(-val) {
		r.target += uint32(val)
	} else if val < 0 {
		r.target = 0
	} else if val > 0 {
		r.target += uint32(val)
	} // else is val=0, do nothing
}

func (r *batcher) tryReserveBatchSlot() bool {
	if r.maxConcurrentBatches == 0 {
		return true
	}
	select {
	case r.inflight <- struct{}{}:
		return true
	default:
		return false
	}
}

func (r *batcher) releaseBatchSlot() {
	if r.maxConcurrentBatches > 0 {
		<-r.inflight
	}
}

func (r *batcher) confirmInflightIsZero() bool {
	isZero := true
	for {
		select {
		case <-r.inflight:
			isZero = false
		default:
			return isZero
		}
	}
}

func (r *batcher) Inflight() uint32 {
	return uint32(len(r.inflight))
}

func (r *batcher) processBatch(watcher Watcher, batch []Operation) {
	if len(batch) == 0 {
		return
	}
	r.lastFlushWithRecords = time.Now()

	// raise event
	if r.emitBatch {
		r.Emit(BatchEvent, len(batch), "", batch)
	}

	go func() {

		// increment an attempt
		for _, op := range batch {
			op.MakeAttempt()
		}

		// process the batch
		waitForDone := make(chan struct{})
		go func() {
			defer close(waitForDone)
			watcher.ProcessBatch(batch)
		}()

		// the batch is "done" when the ProcessBatch func() finishes or the maxOperationTime is exceeded
		maxOperationTime := r.maxOperationTime
		if watcher.MaxOperationTime() > 0 {
			maxOperationTime = watcher.MaxOperationTime()
		}
		select {
		case <-waitForDone:
		case <-time.After(maxOperationTime):
		}

		// decrement target
		var total int = 0
		for _, op := range batch {
			total += int(op.Cost())
		}
		r.incTarget(-total)

		// remove from inflight
		r.releaseBatchSlot()

	}()
}

// Call this method to start the processing loop. The processing loop requests capacity at the CapacityInterval, organizes operations into
// batches at the FlushInterval, and audits the capacity target at the AuditInterval.
func (r *batcher) Start(ctx context.Context) (err error) {

	// only allow one phase at a time
	r.phaseMutex.Lock()
	defer r.phaseMutex.Unlock()
	if r.phase != phaseUninitialized {
		err = ImproperOrderError
		return
	}

	// apply defaults
	r.applyDefaults()

	// start the timers
	capacityTimer := time.NewTicker(r.capacityInterval)
	flushTimer := time.NewTicker(r.flushInterval)
	auditTimer := time.NewTicker(r.auditInterval)

	// process
	go func() {

		// loop
		for {
			select {

			case <-ctx.Done():
				// shutdown when context is cancelled
				capacityTimer.Stop()
				flushTimer.Stop()
				auditTimer.Stop()
				r.shutdown()
				return

			case <-r.pause:
				// pause; typically this is requested because there is too much pressure on the datastore
				r.Emit(PauseEvent, int(r.pauseTime.Milliseconds()), "", nil)
				time.Sleep(r.pauseTime)
				r.resume()
				r.Emit(ResumeEvent, 0, "", nil)

			case <-auditTimer.C:
				// ensure that if the buffer is empty and everything should have been flushed, that target is set to 0
				if r.buffer.size() == 0 && time.Since(r.lastFlushWithRecords) > r.maxOperationTime {
					targetIsZero := r.confirmTargetIsZero()
					inflightIsZero := r.confirmInflightIsZero()
					switch {
					case !targetIsZero && !inflightIsZero:
						r.Emit(AuditFailEvent, 0, AuditMsgFailureOnTargetAndInflight, nil)
					case !targetIsZero:
						r.Emit(AuditFailEvent, 0, AuditMsgFailureOnTarget, nil)
					case !inflightIsZero:
						r.Emit(AuditFailEvent, 0, AuditMsgFailureOnInflight, nil)
					default:
						r.Emit(AuditPassEvent, 0, "", nil)
					}
				} else {
					r.Emit(AuditSkipEvent, 0, "", nil)
				}

			case <-capacityTimer.C:
				// ask for capacity
				if r.ratelimiter != nil {
					request := r.NeedsCapacity()
					if r.emitRequest {
						r.Emit(RequestEvent, int(request), "", nil)
					}
					r.ratelimiter.GiveMe(request)
				}

			case <-flushTimer.C:
				r.Flush()

			case <-r.flush:
				// flush a percentage of the capacity (by default 10%)
				if r.emitFlush {
					r.Emit(FlushStartEvent, 0, "", nil)
				}

				// determine the capacity
				enforceCapacity := r.ratelimiter != nil
				var capacity uint32
				if enforceCapacity {
					capacity += uint32(float64(r.ratelimiter.Capacity()) / 1000.0 * float64(r.flushInterval.Milliseconds()))
				}

				// if there are operations in the buffer, go up to the capacity
				batches := make(map[Watcher][]Operation)
				var consumed uint32 = 0

				// reset the buffer cursor to the top of the buffer
				op := r.buffer.top()

				for {

					// the buffer is empty or we are at the end
					if op == nil {
						break
					}

					// enforce capacity
					if enforceCapacity && consumed >= capacity {
						break
					}

					// batch
					switch {
					case op.IsBatchable():
						watcher := op.Watcher()
						batch, ok := batches[watcher]
						if (batch == nil || !ok) && !r.tryReserveBatchSlot() {
							op = r.buffer.skip()
							continue // there is no batch slot available
						}
						consumed += op.Cost()
						batch = append(batch, op)
						max := watcher.MaxBatchSize()
						if max > 0 && len(batch) >= int(max) {
							r.processBatch(watcher, batch)
							batches[watcher] = nil
						} else {
							batches[watcher] = batch
						}
						op = r.buffer.remove()
					case r.tryReserveBatchSlot():
						consumed += op.Cost()
						watcher := op.Watcher()
						r.processBatch(watcher, []Operation{op})
						op = r.buffer.remove()
					default:
						// there is no batch slot available
						op = r.buffer.skip()
					}

				}

				// flush all batches that were seen
				for watcher, batch := range batches {
					r.processBatch(watcher, batch)
				}

				if r.emitFlush {
					r.Emit(FlushDoneEvent, 0, "", nil)
				}
			}
		}

	}()

	// end starting
	r.phase = phaseStarted

	return
}

func (r *batcher) shutdown() {

	// only allow one phase at a time
	r.phaseMutex.Lock()
	defer r.phaseMutex.Unlock()

	// clear the buffer
	r.buffer.shutdown()

	// update the phase
	r.phase = phaseStopped

	// emit the shutdown event
	r.Emit(ShutdownEvent, 0, "", nil)

}
