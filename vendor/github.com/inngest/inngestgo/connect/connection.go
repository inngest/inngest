package connect

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/coder/websocket"
	"github.com/inngest/inngest/pkg/connect/types"
	"github.com/inngest/inngest/pkg/connect/wsproto"
	connectproto "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"golang.org/x/sync/errgroup"
	"io"
	"net"
	"time"
)

type connectReport struct {
	reconnect bool
	err       error
}

func (h *connectHandler) connect(ctx context.Context, data connectionEstablishData) {
	// Set up connection (including connect handshake protocol)
	preparedConn, err := h.prepareConnection(ctx, data)
	if err != nil {
		h.logger.Error("could not establish connection", "err", err)

		h.notifyConnectDoneChan <- connectReport{
			reconnect: shouldReconnect(err),
			err:       fmt.Errorf("could not establish connection: %w", err),
		}
		return
	}

	// Notify that the connection was established
	h.notifyConnectedChan <- struct{}{}

	// Set up connection lifecycle logic (receiving messages, handling requests, etc.)
	err = h.handleConnection(ctx, data, preparedConn.ws, preparedConn.gatewayHost)
	if err != nil {
		h.logger.Error("could not handle connection", "err", err)

		if errors.Is(err, errGatewayDraining) {
			// if the gateway is draining, the original connection was closed, and we already reconnected inside handleConnection
			return
		}

		h.notifyConnectDoneChan <- connectReport{
			reconnect: shouldReconnect(err),
			err:       fmt.Errorf("could not handle connection: %w", err),
		}
		return
	}

	h.notifyConnectDoneChan <- connectReport{}
}

type connectionEstablishData struct {
	hashedSigningKey      []byte
	numCpuCores           int32
	totalMem              int64
	marshaledFns          []byte
	marshaledCapabilities []byte
	manualReadinessAck    bool
}

type connection struct {
	ws           *websocket.Conn
	gatewayHost  string
	connectionId string
}

func (h *connectHandler) prepareConnection(ctx context.Context, data connectionEstablishData) (connection, error) {
	connectTimeout, cancelConnectTimeout := context.WithTimeout(ctx, 10*time.Second)
	defer cancelConnectTimeout()

	gatewayHost := h.hostsManager.pickAvailableGateway()
	if gatewayHost == "" {
		// All gateways have been tried, reset the internal state to retry
		h.hostsManager.resetGateways()

		return connection{}, reconnectError{fmt.Errorf("no available gateway hosts")}
	}

	// Establish WebSocket connection to one of the gateways
	ws, _, err := websocket.Dial(connectTimeout, gatewayHost, &websocket.DialOptions{
		Subprotocols: []string{
			types.GatewaySubProtocol,
		},
	})
	if err != nil {
		h.hostsManager.markUnreachableGateway(gatewayHost)
		return connection{}, reconnectError{fmt.Errorf("could not connect to gateway: %w", err)}
	}

	// Connection ID is unique per connection, reconnections should get a new ID
	connectionId := ulid.MustNew(ulid.Now(), rand.Reader)

	h.logger.Debug("websocket connection established", "gateway_host", gatewayHost)

	err = h.performConnectHandshake(ctx, connectionId.String(), ws, gatewayHost, data)
	if err != nil {
		return connection{}, reconnectError{fmt.Errorf("could not perform connect handshake: %w", err)}
	}

	return connection{
		ws:           ws,
		gatewayHost:  gatewayHost,
		connectionId: connectionId.String(),
	}, nil
}

func (h *connectHandler) handleConnection(ctx context.Context, data connectionEstablishData, ws *websocket.Conn, gatewayHost string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	defer func() {
		// This is a fallback safeguard to always close the WebSocket connection at the end of the function
		// Usually, we provide a specific reason, so this is only necessary for unhandled errors
		_ = ws.CloseNow()
	}()

	// When shutting down the worker, close the connection with a reason
	go func() {
		<-ctx.Done()
		_ = ws.Close(websocket.StatusNormalClosure, connectproto.WorkerDisconnectReason_WORKER_SHUTDOWN.String())
	}()

	// Send buffered but unsent messages if connection was re-established
	if len(h.messageBuffer) > 0 {
		h.logger.Debug("sending buffered messages", "count", len(h.messageBuffer))
		err := h.sendBufferedMessages(ws)
		if err != nil {
			return reconnectError{fmt.Errorf("could not send buffered messages: %w", err)}
		}
	}

	go func() {
		heartbeatTicker := time.NewTicker(WorkerHeartbeatInterval)
		defer heartbeatTicker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-heartbeatTicker.C:
				err := wsproto.Write(ctx, ws, &connectproto.ConnectMessage{
					Kind: connectproto.GatewayMessageType_WORKER_HEARTBEAT,
				})
				if err != nil {
					h.logger.Error("failed to send worker heartbeat", "err", err)
				}
			}

		}
	}()

	eg := errgroup.Group{}
	eg.Go(func() error {
		for {
			var msg connectproto.ConnectMessage
			err := wsproto.Read(context.Background(), ws, &msg)
			if err != nil {
				h.logger.Error("failed to read message", "err", err)

				// The connection may still be active, but for some reason we couldn't read the message
				return err
			}

			h.logger.Debug("received gateway request", "msg", &msg)

			switch msg.Kind {
			case connectproto.GatewayMessageType_GATEWAY_CLOSING:
				// Stop the read loop: We will not receive any further messages and should establish a new connection
				// We can still use the old connection to send replies to the gateway
				return errGatewayDraining
			case connectproto.GatewayMessageType_GATEWAY_EXECUTOR_REQUEST:
				// Handle invoke in a non-blocking way to allow for other messages to be processed
				h.workerPool.Add(workerPoolMsg{
					msg: &msg,
					ws:  ws,
				})
			default:
				h.logger.Error("got unknown gateway request", "err", err)
				continue
			}
		}
	})

	h.logger.Debug("waiting for read loop to end")

	// If read loop ends, this can be for two reasons
	// - Connection loss (io.EOF), read loop terminated intentionally (CloseError), other error (unexpected)
	// - Worker shutdown, parent context got cancelled
	if err := eg.Wait(); err != nil && ctx.Err() == nil {
		if errors.Is(err, errGatewayDraining) {
			h.hostsManager.markDrainingGateway(gatewayHost)

			// Gateway is draining and will not accept new connections.
			// We must reconnect to a different gateway, only then can we close the old connection.
			waitUntilConnected, doneWaiting := context.WithCancel(context.Background())
			defer doneWaiting()

			// Intercept connected signal and pass it to the main goroutine
			notifyConnectedInterceptChan := make(chan struct{})
			go func() {
				<-h.notifyConnectedChan
				notifyConnectedInterceptChan <- struct{}{}
				doneWaiting()
			}()

			// Establish new connection and pass close reports back to the main goroutine
			go h.connect(context.Background(), data)

			cancel()

			// Wait until the new connection is established before closing the old one
			select {
			case <-waitUntilConnected.Done():
			case <-time.After(10 * time.Second):
				h.logger.Error("timed out waiting for new connection to be established")
			}

			// By returning, we will close the old connection
			return errGatewayDraining
		}

		h.logger.Debug("read loop ended with error", "err", err)

		// In case the gateway intentionally closed the connection, we'll receive a close error
		cerr := websocket.CloseError{}
		if errors.As(err, &cerr) {
			h.logger.Error("connection closed with reason", "reason", cerr.Reason)

			// Reconnect!
			return reconnectError{fmt.Errorf("connection closed with reason %q: %w", cerr.Reason, cerr)}
		}

		// connection closed without reason
		if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
			h.logger.Error("failed to read message from gateway, lost connection unexpectedly", "err", err)
			return reconnectError{fmt.Errorf("connection closed unexpectedly: %w", cerr)}
		}

		// If this is not a worker shutdown, we should reconnect
		return reconnectError{fmt.Errorf("connection closed unexpectedly: %w", ctx.Err())}
	}

	// Perform graceful shutdown routine (context was cancelled)

	// Signal gateway that we won't process additional messages!
	{
		h.logger.Debug("sending worker pause message")
		err := wsproto.Write(context.Background(), ws, &connectproto.ConnectMessage{
			Kind: connectproto.GatewayMessageType_WORKER_PAUSE,
		})
		if err != nil {
			// We should not exit here, as we're already in the shutdown routine
			h.logger.Error("failed to serialize worker pause msg", "err", err)
		}
	}

	h.logger.Debug("waiting for in-progress requests to finish")

	// Wait until all in-progress requests are completed
	h.workerPool.Wait()

	// Attempt to shut down connection if not already done
	_ = ws.Close(websocket.StatusNormalClosure, connectproto.WorkerDisconnectReason_WORKER_SHUTDOWN.String())

	return nil
}

func (h *connectHandler) withTemporaryConnection(data connectionEstablishData, handler func(ws *websocket.Conn) error) error {
	// Prevent this connection from receiving work
	data.manualReadinessAck = true

	maxAttempts := 4

	var conn *websocket.Conn
	var attempts int
	for {
		if attempts == maxAttempts {
			return fmt.Errorf("could not establish connection after %d attempts", maxAttempts)
		}

		ws, err := h.prepareConnection(context.Background(), data)
		if err != nil {
			attempts++
			continue
		}

		conn = ws.ws
		break
	}

	defer func() {
		_ = conn.Close(websocket.StatusNormalClosure, connectproto.WorkerDisconnectReason_WORKER_SHUTDOWN.String())
	}()

	err := handler(conn)
	if err != nil {
		return err
	}

	return nil
}
