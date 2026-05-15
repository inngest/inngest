package connect

import (
	"context"
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
	"sync/atomic"
	"time"
)

var (
	defaultWSReadLimit int64 = 10 * 1024 * 1024 // 10MB
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
		h.logger.Error("could not establish connection", "err", err, "reconnect", shouldReconnect(err))

		h.notifyConnectDoneChan <- connectReport{
			reconnect: shouldReconnect(err),
			err:       fmt.Errorf("could not establish connection: %w", err),
		}
		return
	}

	l := h.logger.With(preparedConn.logAttrs()...)
	l.Debug("connection established")

	// Notify that the connection was established
	h.notifyConnectedChan <- struct{}{}

	// If an additional notification channel was provided, notify it as well
	if o.notifyConnectedChan != nil {
		o.notifyConnectedChan <- struct{}{}
		close(o.notifyConnectedChan)
	}

	// Set up connection lifecycle logic (receiving messages, handling requests, etc.)
	err = h.handleConnection(h.workerCtx, data, preparedConn)
	if err != nil {
		l.Error("could not handle connection", "err", err, "reconnect", shouldReconnect(err))

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
	marshaledCapabilities []byte
	manualReadinessAck    bool
	apps                  []*connectproto.AppConfiguration
}

type connection struct {
	ws               *websocket.Conn
	gatewayGroupName string
	gatewayEndpoint  string
	connectionId     string

	heartbeatInterval   time.Duration
	extendLeaseInterval time.Duration

	retired atomic.Bool
}

func (c *connection) logAttrs() []any {
	return []any{
		"connection_id", c.connectionId,
		"gateway_group", c.gatewayGroupName,
		"gateway_endpoint", c.gatewayEndpoint,
	}
}

func (c *connection) retire() {
	c.retired.Store(true)
}

func (c *connection) isRetired() bool {
	return c.retired.Load()
}

func (h *connectHandler) prepareConnection(ctx context.Context, data connectionEstablishData, excludeGateways []string) (*connection, error) {
	connectTimeout, cancelConnectTimeout := context.WithTimeout(ctx, 10*time.Second)
	defer cancelConnectTimeout()

	startTime := time.Now()

	startRes, err := h.apiClient.start(ctx, data.hashedSigningKey, &connectproto.StartRequest{
		ExcludeGateways: excludeGateways,
	}, h.logger)
	if err != nil {
		return nil, newReconnectErr(fmt.Errorf("could not start connection: %w", err))
	}

	h.logger.Debug("handshake successful", "gateway_endpoint", startRes.GetGatewayEndpoint(), "gateway_group", startRes.GetGatewayGroup())

	gatewayHost, err := url.Parse(startRes.GetGatewayEndpoint())
	if err != nil {
		return nil, newReconnectErr(fmt.Errorf("received invalid start gateway host: %w", err))
	}

	if h.opts.RewriteGatewayEndpoint != nil {
		newGatewayHost, err := h.opts.RewriteGatewayEndpoint(*gatewayHost)
		if err != nil {
			return nil, newReconnectErr(fmt.Errorf("rewriting gateway host failed: %w", err))
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
		return nil, newReconnectErr(fmt.Errorf("could not connect to gateway: %w", err))
	}

	// Set message read limit if configured (skip if nil or 0 to use default)
	readLimit := defaultWSReadLimit
	if h.opts.MessageReadLimit != nil && *h.opts.MessageReadLimit != 0 {
		readLimit = *h.opts.MessageReadLimit
	}
	ws.SetReadLimit(readLimit)

	connectionId, err := ulid.Parse(startRes.GetConnectionId())
	if err != nil {
		return nil, newReconnectErr(fmt.Errorf("could not parse connection ID: %w", err))
	}

	h.logger.Debug("websocket connection established", "gateway_host", gatewayHost)

	readyPayload, err := h.performConnectHandshake(ctx, connectionId.String(), ws, startRes, data, startTime)
	if err != nil {
		return nil, newReconnectErr(fmt.Errorf("could not perform connect handshake: %w", err))
	}

	heartbeatInterval, err := time.ParseDuration(readyPayload.GetHeartbeatInterval())
	if err != nil {
		return nil, newReconnectErr(fmt.Errorf("could not parse heartbeat interval: %w", err))
	}

	extendLeaseInterval, err := time.ParseDuration(readyPayload.GetExtendLeaseInterval())
	if err != nil {
		return nil, newReconnectErr(fmt.Errorf("could not parse extend lease interval: %w", err))
	}

	return &connection{
		ws:                  ws,
		gatewayGroupName:    startRes.GetGatewayGroup(),
		gatewayEndpoint:     gatewayHost.String(),
		connectionId:        connectionId.String(),
		heartbeatInterval:   heartbeatInterval,
		extendLeaseInterval: extendLeaseInterval,
	}, nil
}

func (h *connectHandler) handleConnection(ctx context.Context, data connectionEstablishData, preparedConn *connection) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	l := h.logger.With(preparedConn.logAttrs()...)

	defer func() {
		// This is a fallback safeguard to always close the WebSocket connection at the end of the function
		// Usually, we provide a specific reason, so this is only necessary for unhandled errors
		_ = preparedConn.ws.CloseNow()
	}()

	// Send buffered but unsent messages if connection was re-established
	if h.messageBuffer.hasMessages() {
		err := h.messageBuffer.flush(data.hashedSigningKey)
		if err != nil {
			return newReconnectErr(fmt.Errorf("could not send buffered messages: %w", err))
		}
	}

	go func() {
		heartbeatTicker := time.NewTicker(preparedConn.heartbeatInterval)
		defer heartbeatTicker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-heartbeatTicker.C:
				err := wsproto.Write(context.Background(), preparedConn.ws, &connectproto.ConnectMessage{
					Kind: connectproto.GatewayMessageType_WORKER_HEARTBEAT,
				})
				if err != nil {
					l.Error("failed to send worker heartbeat", "err", err, "heartbeat_interval", preparedConn.heartbeatInterval.String())
				}
				l.Debug("sent worker heartbeat", "heartbeat_interval", preparedConn.heartbeatInterval.String())
			}

		}
	}()

	readerLifetimeContext, cancelReaderLifetimeContext := context.WithCancel(ctx)
	defer cancelReaderLifetimeContext()

	heartbeatReceived := make(chan struct{}, 1)
	go func() {
		// Wait until initial heartbeat was sent out
		select {
		case <-ctx.Done():
			return
		case <-time.After(preparedConn.heartbeatInterval):
		}

		heartbeatTimeout := 2 * preparedConn.heartbeatInterval
		heartbeatReplyTimer := time.NewTimer(heartbeatTimeout)
		defer heartbeatReplyTimer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-heartbeatReceived:
				if !heartbeatReplyTimer.Stop() {
					select {
					case <-heartbeatReplyTimer.C:
					default:
					}
				}
				heartbeatReplyTimer.Reset(heartbeatTimeout)
			case <-heartbeatReplyTimer.C:
				// No heartbeat received in time!
				l.Error("did not receive gateway heartbeat in time", "heartbeat_interval", preparedConn.heartbeatInterval.String(), "heartbeat_timeout", heartbeatTimeout.String())
				cancelReaderLifetimeContext()
				return
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
			err := wsproto.Read(readerLifetimeContext, preparedConn.ws, &msg)
			if err != nil {
				l.Error("failed to read message", "err", err, "reader_context_error", readerLifetimeContext.Err())

				// The connection may still be active, but for some reason we couldn't read the message
				return err
			}

			l.Debug("received gateway request", "kind", msg.Kind.String())

			switch msg.Kind {
			case connectproto.GatewayMessageType_GATEWAY_CLOSING:
				// Stop the read loop: We will not receive any further messages and should establish a new connection
				// We can still use the old connection to send replies to the gateway
				l.Info("gateway requested connection drain")
				return errGatewayDraining
			case connectproto.GatewayMessageType_GATEWAY_EXECUTOR_REQUEST:
				// Handle invoke in a non-blocking way to allow for other messages to be processed
				msgCopy := connectproto.ConnectMessage{
					Kind: msg.Kind,
				}
				if len(msg.Payload) > 0 {
					msgCopy.Payload = append([]byte(nil), msg.Payload...)
				}
				h.workerPool.Add(workerPoolMsg{
					msg:          &msgCopy,
					preparedConn: preparedConn,
				})
			case connectproto.GatewayMessageType_GATEWAY_HEARTBEAT:
				select {
				case heartbeatReceived <- struct{}{}:
				default:
				}
			case connectproto.GatewayMessageType_WORKER_REPLY_ACK:
				if err := h.handleMessageReplyAck(&msg); err != nil {
					l.Error("could not handle message reply ack", "err", err)
					continue
				}
			case connectproto.GatewayMessageType_WORKER_REQUEST_EXTEND_LEASE_ACK:
				if err := h.handleWorkerRequestExtendLeaseAck(&msg); err != nil {
					l.Error("could not handle extend lease ack", "err", err)
					continue
				}
			default:
				l.Debug("got unknown gateway request", "err", err)
				continue
			}
		}
	})

	l.Debug("waiting for read loop to end")

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
			go h.connect(context.Background(), data, withNotifyConnectedChan(notifyConnectedChan), withExcludeGateways(preparedConn.gatewayGroupName))

			// Wait until the new connection is established before closing the old one
			<-waitUntilConnected.Done()
			if errors.Is(waitUntilConnected.Err(), context.DeadlineExceeded) {
				l.Error("timed out waiting for new connection to be established")
			}

			// Send a proper close frame
			preparedConn.retire()
			_ = preparedConn.ws.Close(websocket.StatusNormalClosure, connectproto.WorkerDisconnectReason_WORKER_SHUTDOWN.String())

			return errGatewayDraining
		}

		l.Debug("read loop ended with error", "err", err)
		preparedConn.retire()

		// In case the gateway intentionally closed the connection, we'll receive a close error
		cerr := websocket.CloseError{}
		if errors.As(err, &cerr) {
			l.Error("connection closed with reason", "reason", cerr.Reason)

			// Reconnect!
			return newReconnectErr(fmt.Errorf("connection closed with reason %q: %w", cerr.Reason, cerr))
		}

		// connection closed without reason
		if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
			l.Error("failed to read message from gateway, lost connection unexpectedly", "err", err)
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
		l.Debug("sending worker pause message")
		err := wsproto.Write(context.Background(), preparedConn.ws, &connectproto.ConnectMessage{
			Kind: connectproto.GatewayMessageType_WORKER_PAUSE,
		})
		if err != nil {
			// We should not exit here, as we're already in the shutdown routine
			l.Error("failed to serialize worker pause msg", "err", err)
		}
	}

	l.Debug("waiting for in-progress requests to finish")

	// Wait until all in-progress requests are completed
	h.workerPool.Wait()

	// Attempt to shut down connection if not already done
	preparedConn.retire()
	_ = preparedConn.ws.Close(websocket.StatusNormalClosure, connectproto.WorkerDisconnectReason_WORKER_SHUTDOWN.String())

	// Attempt to flush leftover messages before closing
	if h.messageBuffer.hasMessages() {
		err := h.messageBuffer.flush(data.hashedSigningKey)
		if err != nil {
			l.Error("could not send buffered messages", "err", err)
		}
	}

	l.Debug("connection done")

	return nil
}

func (h *connectHandler) handleMessageReplyAck(msg *connectproto.ConnectMessage) error {
	var payload connectproto.WorkerReplyAckData
	if err := proto.Unmarshal(msg.Payload, &payload); err != nil {
		return fmt.Errorf("could not unmarshal reply ack data: %w", err)
	}

	h.messageBuffer.acknowledge(payload.RequestId)

	return nil
}

func (h *connectHandler) handleWorkerRequestExtendLeaseAck(msg *connectproto.ConnectMessage) error {
	var payload connectproto.WorkerRequestExtendLeaseAckData
	if err := proto.Unmarshal(msg.Payload, &payload); err != nil {
		return fmt.Errorf("could not unmarshal extend lease ack data: %w", err)
	}

	h.workerPool.inProgressLeasesLock.Lock()
	defer h.workerPool.inProgressLeasesLock.Unlock()

	if payload.NewLeaseId != nil {
		if _, ok := h.workerPool.inProgressLeases[payload.RequestId]; !ok {
			return nil
		}
		h.workerPool.inProgressLeases[payload.RequestId] = *payload.NewLeaseId
	} else {
		// remove local request lease to stop extending
		delete(h.workerPool.inProgressLeases, payload.RequestId)
	}

	return nil
}
