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
	"time"
)

// defaultMissedGatewayHeartbeatTolerance allows transient gateway heartbeat
// gaps without keeping an unhealthy websocket open indefinitely.
const defaultMissedGatewayHeartbeatTolerance = 3

var (
	defaultWSReadLimit int64 = 10 * 1024 * 1024 // 10MB

	gatewayDrainReplacementTimeout = 10 * time.Second
)

func (h *connectHandler) missedGatewayHeartbeatTolerance() int {
	if h.opts.MissedGatewayHeartbeatTolerance == nil || *h.opts.MissedGatewayHeartbeatTolerance < 0 {
		return defaultMissedGatewayHeartbeatTolerance
	}

	return *h.opts.MissedGatewayHeartbeatTolerance
}

func gatewayHeartbeatTimeout(heartbeatInterval time.Duration, missedTolerance int) time.Duration {
	// The first interval covers the next expected gateway heartbeat. Each
	// additional interval is one tolerated miss before disconnecting.
	return time.Duration(missedTolerance+1) * heartbeatInterval
}

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
			// Gateway drain owns its replacement and old-generation close work
			// inside handleConnection; the manager loop remains ACTIVE.
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

	lifecycle connLifecycle
}

func (c *connection) logAttrs() []any {
	return []any{
		"connection_id", c.connectionId,
		"gateway_group", c.gatewayGroupName,
		"gateway_endpoint", c.gatewayEndpoint,
	}
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

	preparedConn := &connection{
		ws:               ws,
		gatewayGroupName: startRes.GetGatewayGroup(),
		gatewayEndpoint:  gatewayHost.String(),
		connectionId:     connectionId.String(),
	}
	preparedConn.initLifecycle(h.logger, h.notifyFlushChan)
	_ = preparedConn.transition(connPhaseHandshaking, "websocket dialed")

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

	preparedConn.heartbeatInterval = heartbeatInterval
	preparedConn.extendLeaseInterval = extendLeaseInterval
	if err := preparedConn.markActive("gateway connection ready"); err != nil {
		return nil, newReconnectErr(fmt.Errorf("could not mark connection active: %w", err))
	}

	return preparedConn, nil
}

func (h *connectHandler) handleConnection(ctx context.Context, data connectionEstablishData, preparedConn *connection) error {
	backgroundCtx, stopBackground := context.WithCancel(ctx)
	defer stopBackground()

	l := h.logger.With(preparedConn.logAttrs()...)

	defer func() {
		// Fallback safeguard for exits that do not reach the explicit drain,
		// shutdown, or read-error close paths. closeNow is idempotent, so this
		// does not duplicate side effects after a lifecycle helper already
		// closed the transport.
		preparedConn.closeNow("handle connection ended")
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
			case <-backgroundCtx.Done():
				return
			case <-heartbeatTicker.C:
				if !preparedConn.canWriteHeartbeat() {
					l.Debug("skipping worker heartbeat because connection phase does not allow heartbeat write", "phase", preparedConn.phase())
					return
				}
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
		// Start checking only after the worker has had a chance to send its
		// first heartbeat. The gateway heartbeat deadline is then reset every
		// time a gateway heartbeat is read below.
		select {
		case <-backgroundCtx.Done():
			return
		case <-time.After(preparedConn.heartbeatInterval):
		}

		missedTolerance := h.missedGatewayHeartbeatTolerance()
		heartbeatTimeout := gatewayHeartbeatTimeout(preparedConn.heartbeatInterval, missedTolerance)
		heartbeatReplyTimer := time.NewTimer(heartbeatTimeout)
		defer heartbeatReplyTimer.Stop()
		for {
			select {
			case <-backgroundCtx.Done():
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
				l.Error("did not receive gateway heartbeat in time", "heartbeat_interval", preparedConn.heartbeatInterval.String(), "heartbeat_timeout", heartbeatTimeout.String(), "missed_heartbeat_tolerance", missedTolerance)
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
				// Gateway drain is gateway-initiated replacement, not local
				// worker shutdown. The old generation stops new request ACKs
				// and heartbeats immediately, but remains open for explicitly
				// allowed writes from already-ACKed in-flight work until a
				// replacement connects or the replacement wait times out.
				l.Info("gateway requested connection drain")
				if err := preparedConn.beginDrain("gateway closing"); err != nil {
					l.Error("could not mark connection draining", "err", err)
				}
				stopBackground()
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

	// When the read loop exits before local shutdown, the websocket generation
	// is no longer a reliable writer. Gateway drain is the one exception: it
	// keeps this generation in Draining while a replacement connects.
	if err := eg.Wait(); err != nil && ctx.Err() == nil {
		if errors.Is(err, errGatewayDraining) {
			// Gateway is draining this generation. Establish a replacement on
			// a different gateway before retiring and closing the old transport
			// so already-ACKed work has a bounded window to reply on the socket
			// that still owns it.
			notifyConnectedChan := make(chan struct{}, 1)
			go h.startConnectionFunc()(context.Background(), data, withNotifyConnectedChan(notifyConnectedChan), withExcludeGateways(preparedConn.gatewayGroupName))

			select {
			case <-notifyConnectedChan:
			case <-time.After(gatewayDrainReplacementTimeout):
				l.Error("timed out waiting for new connection to be established")
			}

			preparedConn.retire("gateway drain complete")
			preparedConn.closeNormal(connectproto.WorkerDisconnectReason_WORKER_SHUTDOWN.String(), "reason", "gateway drain complete")

			return errGatewayDraining
		}

		l.Debug("read loop ended with error", "err", err)
		// Unexpected read termination retires the generation before the manager
		// decides whether to reconnect. That prevents queued request work from
		// attempting stale ACKs on this websocket.
		preparedConn.retire("read loop ended with error", "err", err)

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

	// Graceful shutdown is local worker shutdown, not gateway-initiated
	// replacement. Enter Closing to stop new request ACKs, notify the gateway
	// with WORKER_PAUSE, and keep explicitly allowed writes for already-ACKed
	// in-flight work until the worker pool drains.
	if err := preparedConn.beginClose("worker context canceled"); err != nil {
		l.Error("could not mark connection closing", "err", err)
	}

	// Signal gateway that we won't process additional messages!
	{
		if preparedConn.canWritePause() {
			l.Debug("sending worker pause message")
			err := wsproto.Write(context.Background(), preparedConn.ws, &connectproto.ConnectMessage{
				Kind: connectproto.GatewayMessageType_WORKER_PAUSE,
			})
			if err != nil {
				// We should not exit here, as we're already in the shutdown routine
				l.Error("failed to serialize worker pause msg", "err", err)
			}
		} else {
			l.Debug("skipping worker pause because connection phase does not allow pause write", "phase", preparedConn.phase())
		}
	}

	l.Debug("waiting for in-progress requests to finish")

	// Wait until all in-progress requests are completed
	h.workerPool.Wait()

	preparedConn.retire("worker pool drained")

	// Close through the lifecycle helper after the worker pool drains. The
	// deferred closeNow remains only as a fallback for unhandled exits.
	preparedConn.closeNormal(connectproto.WorkerDisconnectReason_WORKER_SHUTDOWN.String(), "reason", "worker shutdown")

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

// startConnectionFunc returns the connection starter used by paths that need
// to create a replacement generation. Production uses h.connect directly; tests
// can set h.startConnection to control replacement timing without a real dial.
func (h *connectHandler) startConnectionFunc() func(context.Context, connectionEstablishData, ...connectOpt) {
	if h.startConnection != nil {
		return h.startConnection
	}
	return h.connect
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
