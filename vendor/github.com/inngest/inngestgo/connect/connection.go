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
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"net/url"
	"time"
)

type connectReport struct {
	reconnect bool
	err       error
}

type connectOpt func(opts *connectOpts)
type connectOpts struct {
	notifyConnectedChan chan struct{}
	excludeGateways     []string
}

func withNotifyConnectedChan(ch chan struct{}) connectOpt {
	return func(opts *connectOpts) {
		opts.notifyConnectedChan = ch
	}
}

func withExcludeGateways(exclude ...string) connectOpt {
	return func(opts *connectOpts) {
		opts.excludeGateways = exclude
	}
}

func (h *connectHandler) connect(ctx context.Context, data connectionEstablishData, opts ...connectOpt) {
	o := connectOpts{}
	for _, opt := range opts {
		opt(&o)
	}

	// Set up connection (including connect handshake protocol)
	preparedConn, err := h.prepareConnection(ctx, data, o.excludeGateways)
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

	// If an additional notification channel was provided, notify it as well
	if o.notifyConnectedChan != nil {
		o.notifyConnectedChan <- struct{}{}
		close(o.notifyConnectedChan)
	}

	// Set up connection lifecycle logic (receiving messages, handling requests, etc.)
	err = h.handleConnection(h.workerCtx, data, preparedConn.ws, preparedConn.gatewayGroupName)
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
	ws               *websocket.Conn
	gatewayGroupName string
	connectionId     string
}

func (h *connectHandler) prepareConnection(ctx context.Context, data connectionEstablishData, excludeGateways []string) (connection, error) {
	connectTimeout, cancelConnectTimeout := context.WithTimeout(ctx, 10*time.Second)
	defer cancelConnectTimeout()

	startTime := time.Now()

	startRes, err := h.apiClient.start(ctx, data.hashedSigningKey, &connectproto.StartRequest{
		ExcludeGateways: excludeGateways,
	}, h.logger)
	if err != nil {
		return connection{}, newReconnectErr(fmt.Errorf("could not start connection: %w", err))
	}

	h.logger.Debug("handshake successful", "gateway_endpoint", startRes.GetGatewayEndpoint(), "gateway_group", startRes.GetGatewayGroup())

	gatewayHost, err := url.Parse(startRes.GetGatewayEndpoint())
	if err != nil {
		return connection{}, newReconnectErr(fmt.Errorf("received invalid start gateway host: %w", err))
	}

	if h.opts.RewriteGatewayEndpoint != nil {
		newGatewayHost, err := h.opts.RewriteGatewayEndpoint(*gatewayHost)
		if err != nil {
			return connection{}, newReconnectErr(fmt.Errorf("rewriting gateway host failed: %w", err))
		}
		gatewayHost = &newGatewayHost
	}

	// Establish WebSocket connection to one of the gateways
	ws, _, err := websocket.Dial(connectTimeout, gatewayHost.String(), &websocket.DialOptions{
		Subprotocols: []string{
			types.GatewaySubProtocol,
		},
	})
	if err != nil {
		return connection{}, newReconnectErr(fmt.Errorf("could not connect to gateway: %w", err))
	}

	// Connection ID is unique per connection, reconnections should get a new ID
	connectionId := ulid.MustNew(ulid.Timestamp(startTime), rand.Reader)

	h.logger.Debug("websocket connection established", "gateway_host", gatewayHost)

	err = h.performConnectHandshake(ctx, connectionId.String(), ws, startRes, data, startTime)
	if err != nil {
		return connection{}, newReconnectErr(fmt.Errorf("could not perform connect handshake: %w", err))
	}

	return connection{
		ws:               ws,
		gatewayGroupName: startRes.GetGatewayGroup(),
		connectionId:     connectionId.String(),
	}, nil
}

func (h *connectHandler) handleConnection(ctx context.Context, data connectionEstablishData, ws *websocket.Conn, gatewayGroupName string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	defer func() {
		// This is a fallback safeguard to always close the WebSocket connection at the end of the function
		// Usually, we provide a specific reason, so this is only necessary for unhandled errors
		_ = ws.CloseNow()
	}()

	// Send buffered but unsent messages if connection was re-established
	if h.messageBuffer.hasMessages() {
		err := h.messageBuffer.flush(data.hashedSigningKey)
		if err != nil {
			return newReconnectErr(fmt.Errorf("could not send buffered messages: %w", err))
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
				err := wsproto.Write(context.Background(), ws, &connectproto.ConnectMessage{
					Kind: connectproto.GatewayMessageType_WORKER_HEARTBEAT,
				})
				if err != nil {
					h.logger.Error("failed to send worker heartbeat", "err", err)
				}
				h.logger.Debug("sent worker heartbeat")
			}

		}
	}()

	readerLifetimeContext, cancelReaderLifetimeContext := context.WithCancel(ctx)
	defer cancelReaderLifetimeContext()

	var lastGatewayHeartbeatReceived time.Time
	go func() {
		// Wait until initial heartbeat was sent out
		<-time.After(WorkerHeartbeatInterval)

		heartbeatReplyTicker := time.NewTicker(WorkerHeartbeatInterval)
		defer heartbeatReplyTicker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-heartbeatReplyTicker.C:
				gracePeriod := 2 * WorkerHeartbeatInterval
				if lastGatewayHeartbeatReceived.Before(time.Now().Add(-gracePeriod)) {
					// No heartbeat received in time!
					h.logger.Error("did not receive gateway heartbeat in time")
					cancelReaderLifetimeContext()
				}
			}
		}
	}()

	eg := errgroup.Group{}
	eg.Go(func() error {
		for {
			var msg connectproto.ConnectMessage

			// The context will be canceled for two reasons only:
			// - Parent context was canceled (user requested graceful shutdown)
			// - Gateway heartbeat was missed (unexpected connection loss)
			err := wsproto.Read(readerLifetimeContext, ws, &msg)
			if err != nil {
				h.logger.Error("failed to read message", "err", err)

				// The connection may still be active, but for some reason we couldn't read the message
				return err
			}

			h.logger.Debug("received gateway request", "kind", msg.Kind.String())

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
			case connectproto.GatewayMessageType_GATEWAY_HEARTBEAT:
				lastGatewayHeartbeatReceived = time.Now()
			case connectproto.GatewayMessageType_WORKER_REPLY_ACK:
				if err := h.handleMessageReplyAck(&msg); err != nil {
					h.logger.Error("could not handle message reply ack", "err", err)
					continue
				}
			default:
				h.logger.Error("got unknown gateway request", "err", err)
				continue
			}
		}
	})

	h.logger.Debug("waiting for read loop to end")

	// If read loop ends, this can be for two reasons
	// - Connection loss (io.EOF), read loop terminated intentionally (CloseError), other error (unexpected), missed heartbeat (readerLifetimeContext canceled)
	// - Worker shutdown, parent context got cancelled
	if err := eg.Wait(); err != nil && ctx.Err() == nil {
		if errors.Is(err, errGatewayDraining) {
			// Gateway is draining and will not accept new connections.
			// We must reconnect to a different gateway, only then can we close the old connection.
			waitUntilConnected, doneWaiting := context.WithTimeout(context.Background(), 10*time.Second)
			defer doneWaiting()

			// Set up local notification listener
			notifyConnectedChan := make(chan struct{})
			go func() {
				<-notifyConnectedChan
				doneWaiting()
			}()

			// Establish new connection, notify the routine above when the new connection is established
			go h.connect(context.Background(), data, withNotifyConnectedChan(notifyConnectedChan), withExcludeGateways(gatewayGroupName))

			// Wait until the new connection is established before closing the old one
			<-waitUntilConnected.Done()
			if errors.Is(waitUntilConnected.Err(), context.DeadlineExceeded) {
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
			return newReconnectErr(fmt.Errorf("connection closed with reason %q: %w", cerr.Reason, cerr))
		}

		// connection closed without reason
		if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
			h.logger.Error("failed to read message from gateway, lost connection unexpectedly", "err", err)
			return newReconnectErr(fmt.Errorf("connection closed unexpectedly: %w", cerr))
		}

		// gateway heartbeat missed, we should reconnect
		if readerLifetimeContext.Err() != nil {
			return newReconnectErr(fmt.Errorf("connection closed unexpectedly due to missed heartbeat"))
		}

		// If this is not a worker shutdown, we should reconnect
		return newReconnectErr(fmt.Errorf("connection closed unexpectedly: %w", ctx.Err()))
	}

	// Perform graceful shutdown routine (parent context was cancelled)

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

	// Attempt to flush leftover messages before closing
	if h.messageBuffer.hasMessages() {
		err := h.messageBuffer.flush(data.hashedSigningKey)
		if err != nil {
			h.logger.Error("could not send buffered messages", "err", err)
		}
	}

	h.logger.Debug("connection done")

	return nil
}

func (h *connectHandler) handleMessageReplyAck(msg *connectproto.ConnectMessage) error {
	var payload connectproto.WorkerReplyAckData
	if err := proto.Unmarshal(msg.Payload, msg); err != nil {
		return fmt.Errorf("could not unmarshal reply ack data: %w", err)
	}

	h.messageBuffer.acknowledge(payload.RequestId)

	return nil
}
