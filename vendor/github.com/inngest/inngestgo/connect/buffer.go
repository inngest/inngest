package connect

import (
	"context"
	"fmt"
	connectproto "github.com/inngest/inngest/proto/gen/connect/v1"
	"log/slog"
	"sync"
	"time"
)

type messageBuffer struct {
	buffered   map[string]*connectproto.SDKResponse
	pendingAck map[string]*connectproto.SDKResponse
	lock       sync.Mutex
	logger     *slog.Logger
	apiClient  *workerApiClient
}

func newMessageBuffer(apiClient *workerApiClient, logger *slog.Logger) *messageBuffer {
	return &messageBuffer{
		logger:     logger,
		buffered:   make(map[string]*connectproto.SDKResponse),
		pendingAck: make(map[string]*connectproto.SDKResponse),
		lock:       sync.Mutex{},
		apiClient:  apiClient,
	}
}

func (m *messageBuffer) flush(currentHashedSigningKey []byte) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	attempt := 0
	for {
		if attempt == 5 {
			return fmt.Errorf("could not send %d buffered messages", len(m.buffered))
		}

		for id, msg := range m.buffered {
			err := m.apiClient.sendBufferedMessage(context.Background(), currentHashedSigningKey, msg)
			if err != nil {
				m.logger.Error("could not send buffered message via API", "err", err, "req_id", msg.RequestId)
				break
			}

			m.logger.Debug("sent buffered message via API", "msg", msg)
			delete(m.buffered, id)
		}

		if len(m.buffered) == 0 {
			break
		}

		attempt++
	}

	return nil
}

func (m *messageBuffer) hasMessages() bool {
	m.lock.Lock()
	defer m.lock.Unlock()

	return len(m.buffered) > 0
}

func (m *messageBuffer) append(msg *connectproto.SDKResponse) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.buffered[msg.RequestId] = msg

	// In case message was still marked as pending, remove it
	delete(m.pendingAck, msg.RequestId)
}

func (m *messageBuffer) addPending(ctx context.Context, resp *connectproto.SDKResponse, timeout time.Duration) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.pendingAck[resp.RequestId] = resp

	go func() {
		for {
			select {
			case <-ctx.Done():
				break
			case <-time.After(timeout):
				break
			}

			m.lock.Lock()
			// If message is still in outgoing messages, it wasn't acknowledged. Add to buffer.
			if _, ok := m.pendingAck[resp.RequestId]; ok {
				m.buffered[resp.RequestId] = resp
				delete(m.pendingAck, resp.RequestId)
			}
			m.lock.Unlock()
		}
	}()
}

func (m *messageBuffer) acknowledge(id string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	delete(m.pendingAck, id)
}
