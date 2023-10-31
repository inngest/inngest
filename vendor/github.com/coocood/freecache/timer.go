package freecache

import (
	"sync/atomic"
	"time"
)

// Timer holds representation of current time.
type Timer interface {
	// Give current time (in seconds)
	Now() uint32
}

// Timer that must be stopped.
type StoppableTimer interface {
	Timer

	// Release resources of the timer, functionality may or may not be affected
	// It is not called automatically, so user must call it just once
	Stop()
}

// Helper function that returns Unix time in seconds
func getUnixTime() uint32 {
	return uint32(time.Now().Unix())
}

// Default timer reads Unix time always when requested
type defaultTimer struct{}

func (timer defaultTimer) Now() uint32 {
	return getUnixTime()
}

// Cached timer stores Unix time every second and returns the cached value
type cachedTimer struct {
	now    uint32
	ticker *time.Ticker
	done   chan bool
}

// Create cached timer and start runtime timer that updates time every second
func NewCachedTimer() StoppableTimer {
	timer := &cachedTimer{
		now:    getUnixTime(),
		ticker: time.NewTicker(time.Second),
		done:   make(chan bool),
	}

	go timer.update()

	return timer
}

func (timer *cachedTimer) Now() uint32 {
	return atomic.LoadUint32(&timer.now)
}

// Stop runtime timer and finish routine that updates time
func (timer *cachedTimer) Stop() {
	timer.ticker.Stop()
	timer.done <- true
	close(timer.done)

	timer.done = nil
	timer.ticker = nil
}

// Periodically check and update  of time
func (timer *cachedTimer) update() {
	for {
		select {
		case <-timer.done:
			return
		case <-timer.ticker.C:
			atomic.StoreUint32(&timer.now, getUnixTime())
		}
	}
}
