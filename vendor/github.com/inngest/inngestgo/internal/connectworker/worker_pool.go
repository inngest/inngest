package connectworker

import (
	"context"
	connectproto "github.com/inngest/inngest/proto/gen/connect/v1"
	"sync"
)

type workerPoolMsg struct {
	req          *connectproto.GatewayExecutorRequestData
	preparedConn *connection
}

type workerPool struct {
	concurrency    int
	handler        func(msg workerPoolMsg)
	inProgress     sync.WaitGroup
	workerPoolMsgs chan workerPoolMsg

	inProgressLeasesLock sync.Mutex
	inProgressLeases     map[string]string

	// stopLock orders Add against stop.  a message that Add queues before
	// stop is served by a worker, and a message after stop runs on its own
	// goroutine, so no admitted request is left in the queue.
	stopLock sync.RWMutex
	stopped  bool
}

// NewWorkerPool starts concurrency workers.  when ctx is done, each worker
// serves what is left in the queue and then exits.
func NewWorkerPool(ctx context.Context, concurrency int, handler func(msg workerPoolMsg)) *workerPool {
	wp := &workerPool{
		// Should this use the same buffer size as the worker pool?
		workerPoolMsgs:       make(chan workerPoolMsg, concurrency),
		concurrency:          concurrency,
		inProgress:           sync.WaitGroup{},
		inProgressLeases:     make(map[string]string),
		inProgressLeasesLock: sync.Mutex{},
		handler:              handler,
	}
	stopped := make(chan struct{})
	go func() {
		<-ctx.Done()
		wp.stop()
		close(stopped)
	}()
	for i := 0; i < wp.concurrency; i++ {
		go wp.workerPool(stopped)
	}

	return wp
}

func (w *workerPool) workerPool(stopped <-chan struct{}) {
	for {
		select {
		case msg := <-w.workerPoolMsgs:
			w.handler(msg)
		case <-stopped:
			// stop holds new messages out of the queue, so once the queue is
			// empty it stays empty.
			for {
				select {
				case msg := <-w.workerPoolMsgs:
					w.handler(msg)
				default:
					return
				}
			}
		}
	}
}

// stop makes Add run each later message on its own goroutine.
func (w *workerPool) stop() {
	w.stopLock.Lock()
	defer w.stopLock.Unlock()
	w.stopped = true
}

func (w *workerPool) Add(msg workerPoolMsg) {
	w.inProgress.Add(1)

	w.stopLock.RLock()
	defer w.stopLock.RUnlock()
	if w.stopped {
		go w.handler(msg)
		return
	}
	w.workerPoolMsgs <- msg
}

func (w *workerPool) Wait() {
	w.inProgress.Wait()
}

func (w *workerPool) Done() {
	w.inProgress.Done()
}
