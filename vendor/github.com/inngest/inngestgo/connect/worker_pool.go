package connect

import (
	"context"
	"github.com/coder/websocket"
	connectproto "github.com/inngest/inngest/proto/gen/connect/v1"
	"sync"
)

type workerPoolMsg struct {
	msg *connectproto.ConnectMessage
	ws  *websocket.Conn
}

type workerPool struct {
	concurrency    int
	handler        func(msg workerPoolMsg)
	inProgress     sync.WaitGroup
	workerPoolMsgs chan workerPoolMsg
}

func NewWorkerPool(ctx context.Context, concurrency int, handler func(msg workerPoolMsg)) *workerPool {
	wp := &workerPool{
		// Should this use the same buffer size as the worker pool?
		workerPoolMsgs: make(chan workerPoolMsg, concurrency),
		concurrency:    concurrency,
		inProgress:     sync.WaitGroup{},
		handler:        handler,
	}
	for i := 0; i < wp.concurrency; i++ {
		go wp.workerPool(ctx)
	}

	return wp
}

func (w *workerPool) workerPool(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-w.workerPoolMsgs:
			w.handler(msg)
		}
	}
}

func (w *workerPool) Add(msg workerPoolMsg) {
	w.inProgress.Add(1)
	w.workerPoolMsgs <- msg
}

func (w *workerPool) Wait() {
	w.inProgress.Wait()
}

func (w *workerPool) Done() {
	w.inProgress.Done()
}
