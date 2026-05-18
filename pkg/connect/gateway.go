package connect

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/connect/auth"
	connecterrors "github.com/inngest/inngest/pkg/connect/errors"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/connect/types"
	"github.com/inngest/inngest/pkg/connect/wsproto"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/syscode"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/util"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
)

const (
	pkgName = "connect.gateway"
)

const (
	DefaultAppsPerConnection = 10
	MaxAppsPerConnection     = 100

	// wsWriteTimeout is the maximum time allowed for a WebSocket write to complete.
	// Writes use context.Background() because the original request context may already
	// be canceled (e.g. during drain). This timeout prevents blocking indefinitely
	// on a dead TCP connection (network partition).
	wsWriteTimeout = 5 * time.Second
)

func isConnectionClosedErr(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
		return true
	}

	closeErr := websocket.CloseError{}
	return errors.As(err, &closeErr)
}

func isWebSocketReadLimitErr(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), "read limited at")
}

func (c *connectGatewaySvc) closeWithConnectError(ws *websocket.Conn, serr *connecterrors.SocketError) {
	// reason must be limited to 125 bytes and should not be dynamic,
	// so we restrict it to the known syscodes to prevent unintentional overflows
	err := ws.Close(serr.StatusCode, serr.SysCode)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return
		}

		if isConnectionClosedErr(err) {
			return
		}

		c.logger.Debug("could not close WebSocket connection", "err", err, "serr", serr)
	}
}

// connectionHandler holds the WebSocket and current connection, if the connection was established.
type connectionHandler struct {
	svc  *connectGatewaySvc
	conn *state.Connection
	ws   *websocket.Conn

	updateLock sync.Mutex
	log        logger.Logger

	remoteAddr string

	// draining is set to true when a WORKER_PAUSE message is received.
	// Once set, heartbeats must not reset the connection status to READY.
	draining atomic.Bool

	// messageChan receives forwarded requests from the router.
	messageChan chan forwardMessage

	// stopForwarding is closed after the connection is marked DRAINING in
	// Redis, signalling receiveRouterMessagesFromGRPC to exit and remove
	// the connection from the in-memory map.
	stopForwarding     chan struct{}
	stopForwardingOnce sync.Once

	// pendingAcks tracks request IDs waiting for WORKER_REQUEST_ACK.
	// Forward blocks until the ACK arrives or times out.
	pendingAcks sync.Map // requestID -> chan struct{}

	consecutiveConnStatusUpdateFailures atomic.Int64

	lastHeartbeatLock       sync.Mutex
	lastHeartbeatReceivedAt time.Time

	lastStatusLock       sync.Mutex
	lastStatusReceivedAt time.Time

	connectionStatus   connectpb.ConnectionStatus
	connectionStatusOK bool
}

func (c *connectionHandler) setLastHeartbeat(time time.Time) {
	c.lastHeartbeatLock.Lock()
	defer c.lastHeartbeatLock.Unlock()
	c.lastHeartbeatReceivedAt = time
}

func (c *connectionHandler) getLastHeartbeat() time.Time {
	c.lastHeartbeatLock.Lock()
	defer c.lastHeartbeatLock.Unlock()
	return c.lastHeartbeatReceivedAt
}

func (c *connectionHandler) setLastStatus(t time.Time) {
	c.lastStatusLock.Lock()
	defer c.lastStatusLock.Unlock()
	c.lastStatusReceivedAt = t
}

func (c *connectionHandler) getLastStatus() time.Time {
	c.lastStatusLock.Lock()
	defer c.lastStatusLock.Unlock()
	return c.lastStatusReceivedAt
}

func (c *connectionHandler) bindConnectionLogger(conn *state.Connection) {
	c.log = c.log.With(connectionLogAttrs(conn)...)
}

func connectionLogAttrs(conn *state.Connection) []any {
	if conn == nil {
		return nil
	}

	attrs := []any{
		"account_id", conn.AccountID.String(),
		"env_id", conn.EnvID.String(),
		"connection_id", conn.ConnectionId.String(),
		"worker_ip", util.SanitizeLogField(conn.WorkerIP),
	}

	if conn.Data != nil {
		attrs = append(attrs,
			"instance_id", util.SanitizeLogField(conn.Data.GetInstanceId()),
			"sdk_language", conn.Data.GetSdkLanguage(),
			"sdk_version", conn.Data.GetSdkVersion(),
			"sdk_platform", conn.Data.GetPlatform(),
			"sdk_framework", conn.Data.GetFramework(),
			"worker_environment", conn.Data.GetEnvironment(),
			"worker_manual_readiness_ack", conn.Data.GetWorkerManualReadinessAck(),
			"max_worker_concurrency", conn.Data.GetMaxWorkerConcurrency(),
			"app_names", conn.AppNames(),
			"app_count", len(conn.Data.GetApps()),
		)

		system := conn.Data.GetSystemAttributes()
		attrs = append(attrs,
			"worker_cpu_cores", system.GetCpuCores(),
			"worker_mem_bytes", system.GetMemBytes(),
			"worker_os", system.GetOs(),
		)
	}

	groupHashes := make([]string, 0, len(conn.Groups))
	for groupHash := range conn.Groups {
		groupHashes = append(groupHashes, groupHash)
	}
	sort.Strings(groupHashes)

	attrs = append(attrs, "worker_group_hashes", groupHashes)

	return attrs
}

func (c *connectionHandler) logConnStatus(status connectpb.ConnectionStatus, reason string, attrs ...any) {
	c.updateLock.Lock()
	defer c.updateLock.Unlock()

	c.logConnStatusLocked(status, reason, attrs...)
}

func (c *connectionHandler) logConnStatusLocked(status connectpb.ConnectionStatus, reason string, attrs ...any) {
	previous := c.connectionStatus
	previousOK := c.connectionStatusOK
	c.connectionStatus = status
	c.connectionStatusOK = true

	from := ""
	if previousOK {
		from = previous.String()
	}

	logAttrs := []any{
		"from", from,
		"to", status.String(),
		"reason", reason,
	}
	logAttrs = append(logAttrs, attrs...)

	if !previousOK || previous != status {
		c.log.Debug("worker connection status transition", logAttrs...)
		return
	}

	c.log.Trace("worker connection status refreshed", logAttrs...)
}

var ErrDraining = connecterrors.SocketError{
	SysCode:    syscode.CodeConnectGatewayClosing,
	StatusCode: websocket.StatusGoingAway,
	Msg:        "Gateway is draining, reconnect to another gateway",
}

func (c *connectGatewaySvc) closeDraining(ws *websocket.Conn) {
	c.closeWithConnectError(ws, &ErrDraining)
}

func (c *connectGatewaySvc) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This context is canceled when the gateway is shutting down. There's no other deadline.
		ctx, cancel := context.WithCancel(c.runCtx)
		defer cancel()

		// When the gateway starts draining, cancel the connection context
		unsub := c.drainListener.OnDrain(cancel)
		defer unsub()

		ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			Subprotocols: []string{
				types.GatewaySubProtocol,
			},
		})
		if err != nil {
			return
		}

		// Adjust read limit to accommodate large step output responses
		// The default imposed by the websockets library is 32,678 bytes
		ws.SetReadLimit(consts.MaxSDKResponseBodySize)

		additionalMetricsTags := c.metricsTags()

		metrics.IncrConnectGatewayReceiveConnectionAttemptCounter(ctx, 1, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    additionalMetricsTags,
		})

		// Do not accept new connections if the gateway is draining
		if c.isDraining.Load() {
			c.closeDraining(ws)
			return
		}

		var closed bool

		remoteAddr := r.RemoteAddr
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			remoteAddr = xff
		}
		if xff := r.Header.Get("X-Real-IP"); xff != "" {
			remoteAddr = xff
		}

		ch := &connectionHandler{
			svc:            c,
			log:            c.logger,
			ws:             ws,
			updateLock:     sync.Mutex{},
			remoteAddr:     remoteAddr,
			stopForwarding: make(chan struct{}),
		}

		closeReason := connectpb.WorkerDisconnectReason_UNEXPECTED.String()
		var closeReasonPtr atomic.Pointer[string]
		closeReasonPtr.Store(&closeReason)

		setCloseReason := func(reason string) {
			if reason != connectpb.WorkerDisconnectReason_UNEXPECTED.String() {
				closeReasonPtr.Store(&reason)
			}
		}

		c.connectionCount.Add()
		defer func() {
			// This is deferred so we always update the semaphore
			defer c.connectionCount.Done()
			ch.log.Debug("Closing WebSocket connection", "reason", *closeReasonPtr.Load())
			c.logger.Trace("worker disconnected")

			closed = true

			if c.isDraining.Load() {
				c.closeDraining(ws)
				return
			}

			_ = ws.CloseNow()
		}()

		ch.log.Trace("WebSocket connection established, sending hello")

		{
			err = wsproto.Write(ctx, ws, &connectpb.ConnectMessage{
				Kind: connectpb.GatewayMessageType_GATEWAY_HELLO,
			})
			if err != nil {
				if ctx.Err() != nil {
					c.closeDraining(ws)
					return
				}

				if errors.Is(err, context.DeadlineExceeded) {
					return
				}

				if isConnectionClosedErr(err) {
					return
				}

				ch.log.ReportError(err, "could not send hello")
				c.closeWithConnectError(ws, &connecterrors.SocketError{
					SysCode:    syscode.CodeConnectInternal,
					StatusCode: websocket.StatusInternalError,
					Msg:        "could not send gateway hello",
				})

				return
			}
		}

		conn, serr := ch.establishConnection(ctx)
		if serr != nil {
			ch.log.ReportError(serr, "error establishing connection")
			c.closeWithConnectError(ws, serr)

			return
		}

		// Connection was closed during the handshake process
		if conn == nil && serr == nil {
			return
		}

		workerDrainedCtx, notifyWorkerDrained := context.WithCancel(context.Background())
		defer notifyWorkerDrained()

		go func() {
			// This is called in two cases
			// - Connection was closed by the worker, and we already ran all defer actions
			// - Gateway is draining/shutting down and the parent context was canceled
			<-ctx.Done()

			// If the connection is already closed, we don't have to drain
			if closed {
				return
			}

			// If gateway is shutting down, we must immediately start the draining process
			ch.log.Debug("context done, starting draining process", "active_connections", c.connectionCount.Count())

			// Mark connection as draining in-memory to prevent subsequent
			// heartbeats from marking it as ready, but do NOT update Redis
			// yet, the connection stays READY for routing to give the worker
			// enough time to reconnect to a different gateway.
			ch.draining.Store(true)

			setCloseReason(connectpb.WorkerDisconnectReason_GATEWAY_DRAINING.String())

			// Close WS connection once worker established another connection
			defer func() {
				_ = ws.CloseNow()
			}()

			// Signal the client that we're going away so it reconnects.
			drainStart := time.Now()
			closingWriteCtx, closingWriteCancel := context.WithTimeout(context.Background(), wsWriteTimeout)
			defer closingWriteCancel()
			err := wsproto.Write(closingWriteCtx, ws, &connectpb.ConnectMessage{
				Kind: connectpb.GatewayMessageType_GATEWAY_CLOSING,
			})
			if err != nil {
				// Can't tell the worker so we mark as DRAINING immediately.
				if statusErr := ch.updateConnStatus(connectpb.ConnectionStatus_DRAINING, "gateway drain started after failed closing write", "err", err); statusErr != nil {
					ch.log.ReportError(statusErr, "could not update connection status after context done")
				}
				ch.stopForwardingOnce.Do(func() { close(ch.stopForwarding) })
				for _, l := range c.lifecycles {
					go l.OnStartDraining(context.Background(), conn)
				}
				return
			}

			// Wait for the worker to close the connection, worker should make sure that it established
			// a new connection before closing the current one.
			select {
			case <-workerDrainedCtx.Done():
				ch.log.Debug("worker closed connection")
			case <-time.After(c.drainAckTimeout):
				ch.log.Debug("timed out waiting for drain ack, marking connection as draining")
			}

			if statusErr := ch.updateConnStatus(connectpb.ConnectionStatus_DRAINING, "gateway drain completed or timed out", "drain_ack_wait_ms", time.Since(drainStart).Milliseconds()); statusErr != nil {
				ch.log.ReportError(statusErr, "could not update connection status after drain ack")
			}
			ch.stopForwardingOnce.Do(func() { close(ch.stopForwarding) })

			for _, l := range c.lifecycles {
				go l.OnStartDraining(context.Background(), conn)
			}

			metrics.HistogramConnectGatewayDrainDuration(context.Background(), time.Since(drainStart).Milliseconds(), metrics.HistogramOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"workspace_id": conn.EnvID.String(),
				},
			})

			c.closeDraining(ws)
		}()

		// Once a connection is established, we must make sure to update the state on any disconnect,
		// regardless of whether it's permanent or temporary
		defer func() {
			// Ensure receiveRouterMessagesFromGRPC exits on any disconnect.
			ch.stopForwardingOnce.Do(func() { close(ch.stopForwarding) })
			ch.logConnStatus(connectpb.ConnectionStatus_DISCONNECTED, "connection cleanup", "close_reason", *closeReasonPtr.Load())

			// This is a transactional operation, it should always complete regardless of context cancellation
			err := c.stateManager.DeleteConnection(context.Background(), conn.EnvID, conn.ConnectionId)
			if err != nil {
				ch.log.ReportError(err, "error deleting connection from state")
			}

			for _, lifecycle := range c.lifecycles {
				lifecycle.OnDisconnected(context.Background(), conn, *closeReasonPtr.Load())
			}

			ch.log.Debug("cleaned up connection in metadata store")
		}()

		{
			eg := errgroup.Group{}
			eg.SetLimit(10) // Limit concurrent syncs
			for _, group := range conn.Groups {
				group := group
				eg.Go(func() error {
					err := group.Sync(ctx, c.stateManager, c.apiBaseUrl, conn.Data, c.dev)
					if err != nil {
						return fmt.Errorf("could not sync app %q: %w", group.AppName, err)
					}
					return nil
				})
			}

			if err := eg.Wait(); err != nil {
				if ctx.Err() != nil {
					c.closeDraining(ws)
					return
				}

				ch.log.ReportError(err, "error handling sync")

				// Allow returning user-facing errors to hint about invalid config, etc.
				serr := connecterrors.SocketError{}
				if errors.As(err, &serr) {
					sysErr, err := proto.Marshal(&connectpb.SystemError{
						Code:    serr.SysCode,
						Message: serr.Msg,
					})
					if err == nil {
						err := wsproto.Write(ctx, ws, &connectpb.ConnectMessage{
							Kind:    connectpb.GatewayMessageType_SYNC_FAILED,
							Payload: sysErr,
						})
						if err != nil {
							ch.log.Warn("failed to send sync err", "err", err, "sync_err", serr)
						}
					}

					c.closeWithConnectError(ws, &serr)
					return
				}

				c.closeWithConnectError(ws, &connecterrors.SocketError{
					SysCode:    syscode.CodeConnectInternal,
					StatusCode: websocket.StatusInternalError,
					Msg:        "Internal error while syncing",
				})

				return
			}

			// upsert connection to update WorkerGroups map
			if err := ch.updateConnStatus(connectpb.ConnectionStatus_CONNECTED, "worker groups synced"); err != nil {
				ch.log.ReportError(err, "updating connection state after sync failed")
				c.closeWithConnectError(ws, &connecterrors.SocketError{
					SysCode:    syscode.CodeConnectInternal,
					StatusCode: websocket.StatusInternalError,
					Msg:        "connection not stored",
				})

				return
			}

			for _, l := range c.lifecycles {
				go l.OnSynced(context.Background(), conn)
			}

			appNames := make([]string, 0, len(conn.Groups))
			appIds := make([]uuid.UUID, 0, len(conn.Groups))
			syncIds := make([]uuid.UUID, 0, len(conn.Groups))

			for _, group := range conn.Groups {
				appNames = append(appNames, group.AppName)
				if group.AppID != nil {
					appIds = append(appIds, *group.AppID)
				}
				if group.SyncID != nil {
					syncIds = append(syncIds, *group.SyncID)
				}

			}
			ch.log.Debug("synced apps", "app_ids", appIds, "sync_ids", syncIds, "app_names", appNames)
		}

		{
			onSubscribedGRPC := make(chan struct{})

			// Wait for relevant messages and forward them over the WebSocket connection
			go ch.receiveRouterMessagesFromGRPC(ctx, onSubscribedGRPC)
			<-onSubscribedGRPC
		}

		// Run loop
		runLoopCtx, cancelRunLoopContext := context.WithCancel(context.Background())
		defer cancelRunLoopContext()

		eg := errgroup.Group{}
		eg.Go(func() error {
			for {
				// We already handle context cancellations in a goroutine above.
				// If we timed out the read loop, the connection would be closed. This is bad because
				// when draining, we still want to send a close frame to the client.
				var msg connectpb.ConnectMessage
				err := wsproto.Read(runLoopCtx, ws, &msg)
				if err != nil {
					// immediately stop routing messages to this connection
					if err := ch.updateConnStatus(connectpb.ConnectionStatus_DISCONNECTING, "websocket read loop ended", "err", err); err != nil {
						ch.log.ReportError(err, "could not update connection status after read error")
					}

					for _, l := range c.lifecycles {
						go l.OnStartDisconnecting(context.Background(), conn)
					}

					// If the run loop was canceled (e.g. missing consecutive heartbeats), just return
					if errors.Is(err, context.Canceled) && runLoopCtx.Err() != nil {
						return nil
					}

					closeErr := websocket.CloseError{}
					if errors.As(err, &closeErr) {
						// Empty reason (unexpected)
						if closeErr.Reason == "" {
							return nil
						}

						// Force-closed during draining after timeout
						if closeErr.Code == ErrDraining.StatusCode && closeErr.Reason == ErrDraining.SysCode {
							setCloseReason(connectpb.WorkerDisconnectReason_GATEWAY_DRAINING.String())
							return nil
						}

						// Expected worker shutdown
						if closeErr.Code == websocket.StatusNormalClosure && closeErr.Reason == connectpb.WorkerDisconnectReason_WORKER_SHUTDOWN.String() {
							setCloseReason(connectpb.WorkerDisconnectReason_WORKER_SHUTDOWN.String())
							return nil
						}

						ch.log.Debug("connection closed with code and reason", "code", closeErr.Code.String(), "reason", closeErr.Reason)

						// If client connection closed unexpectedly, we should store the reason, if set.
						// If the reason is set, it may have been an intentional close, so the connection
						// may not be re-established.
						// Workers should always close with code: 1000 and reason: WORKER_SHUTDOWN.
						if closeErr.Reason != "" {
							setCloseReason(closeErr.Reason)
						}

						// Do not return an error. We already capture the close reason above.
						return nil
					}

					// connection was closed (this may not be expected but should not be logged as an error)
					// this is expected when the gateway is draining
					if isConnectionClosedErr(err) {
						return nil
					}

					// Unfortunately, the websocket library does not expose a proper error when the size limit is reached,
					// so we have to check the error message instead. This should rarely happen.
					if isWebSocketReadLimitErr(err) {
						setCloseReason(connectpb.WorkerDisconnectReason_MESSAGE_TOO_LARGE.String())
						ch.log.Warn("worker WebSocket message exceeded read limit", "max_bytes", consts.MaxSDKResponseBodySize, "err", err)
						return nil
					}

					ch.log.ReportError(err, "failed to read websocket message")

					// If we failed to read the message for another reason, we should probably reconnect as well.
					return nil
				}

				tags := map[string]any{
					"kind": msg.Kind.String(),
				}
				for k, v := range additionalMetricsTags {
					tags[k] = v
				}

				metrics.IncrConnectGatewayReceivedWorkerMessageCounter(ctx, 1, metrics.CounterOpt{
					PkgName: pkgName,
					Tags:    tags,
				})

				// Handle message in a goroutine to ensure we don't process
				// messages sequentially.
				go func() {
					defer func() {
						if r := recover(); r != nil {
							ch.log.Error("panic handling incoming websocket message",
								"panic", r,
								"stack", string(debug.Stack()),
								"kind", msg.Kind.String(),
							)
							ch.svc.closeWithConnectError(ch.ws, &connecterrors.SocketError{
								SysCode:    syscode.CodeConnectInternal,
								StatusCode: websocket.StatusInternalError,
								Msg:        "internal panic",
							})
						}
					}()

					if serr := ch.handleIncomingWebSocketMessage(&msg); serr != nil {
						ch.svc.closeWithConnectError(ch.ws, serr)
					}
				}()
			}
		})

		// Let the worker know we're ready to receive messages
		{
			statusInterval := c.workerStatusInterval(ctx, conn.AccountID, conn.EnvID)
			readyPayload, err := proto.Marshal(&connectpb.GatewayConnectionReadyData{
				HeartbeatInterval:   c.workerHeartbeatInterval.String(),
				ExtendLeaseInterval: c.workerRequestExtendLeaseInterval.String(),
				StatusInterval:      statusInterval.String(),
			})
			if err != nil {
				ch.log.ReportError(err, "could not marshal connection ready")
				c.closeWithConnectError(ws, &connecterrors.SocketError{
					SysCode:    syscode.CodeConnectInternal,
					StatusCode: websocket.StatusInternalError,
					Msg:        "could not prepare gateway connection ready",
				})

				return
			}

			err = wsproto.Write(ctx, ws, &connectpb.ConnectMessage{
				Kind:    connectpb.GatewayMessageType_GATEWAY_CONNECTION_READY,
				Payload: readyPayload,
			})
			if err != nil {
				if ctx.Err() != nil {
					c.closeDraining(ws)
					return
				}

				if isConnectionClosedErr(err) {
					return
				}

				ch.log.ReportError(err, "could not send connection ready")
				c.closeWithConnectError(ws, &connecterrors.SocketError{
					SysCode:    syscode.CodeConnectInternal,
					StatusCode: websocket.StatusInternalError,
					Msg:        "could not send gateway connection ready",
				})

				return
			}
		}

		if !conn.Data.WorkerManualReadinessAck {
			if ctx.Err() != nil {
				c.closeDraining(ws)
				return
			}

			// Mark connection as ready to receive traffic unless we require manual client ready signal (optional)
			err = ch.updateConnStatus(connectpb.ConnectionStatus_READY, "automatic readiness")
			if err != nil {
				if ctx.Err() != nil {
					c.closeDraining(ws)
					return
				}

				ch.log.ReportError(err, "could not update connection status")
				c.closeWithConnectError(ws, &connecterrors.SocketError{
					SysCode:    syscode.CodeConnectInternal,
					StatusCode: websocket.StatusInternalError,
					Msg:        "could not update connection status",
				})

				return
			}

			for _, l := range c.lifecycles {
				go l.OnReady(context.Background(), conn)
			}
		}

		{
			ch.log.Debug("connection is ready")
			c.logger.Trace("worker connected", "app_names", conn.AppNames())

			successTags := map[string]any{
				"success": true,
			}
			for k, v := range additionalMetricsTags {
				successTags[k] = v
			}

			metrics.IncrConnectGatewayReceiveConnectionAttemptCounter(ctx, 1, metrics.CounterOpt{
				PkgName: pkgName,
				Tags:    successTags,
			})

			metrics.HistogramConnectSetupDuration(ctx, time.Since(conn.Data.StartedAt.AsTime()).Milliseconds(), metrics.HistogramOpt{
				PkgName: pkgName,
				Tags:    additionalMetricsTags,
			})
		}

		// Safeguard: Clean up connections that haven't sent n consecutive heartbeats.
		ch.setLastHeartbeat(time.Now()) // set initial value
		go func() {
			for {
				select {
				case <-runLoopCtx.Done():
					return
				case <-time.After(c.workerHeartbeatInterval):
				}

				if time.Since(ch.getLastHeartbeat()) > time.Duration(c.consecutiveWorkerHeartbeatMissesBeforeConnectionClose)*c.workerHeartbeatInterval {
					setCloseReason(connectpb.WorkerDisconnectReason_CONSECUTIVE_HEARTBEATS_MISSED.String())

					ch.log.Debug("missed consecutive heartbeats, closing connection")

					cancelRunLoopContext()
					return
				}
			}
		}()

		// Connection was drained once it's closed by the worker (even if
		// the connection broke unintentionally, we can stop waiting)
		defer notifyWorkerDrained()

		// The error group returns once the connection is closed
		if err := eg.Wait(); err != nil {
			if err != ErrDraining {
				ch.log.ReportError(err, "error in run loop")
			} else {
				ch.log.Warn("error in run loop", "err", err)
			}
			return
		}
	})
}

func (c *connectionHandler) receiveRouterMessagesFromGRPC(ctx context.Context, onSubscribed chan struct{}) {
	additionalMetricsTags := c.svc.metricsTags()

	messageChan := make(chan forwardMessage)

	connectionID := c.conn.ConnectionId.String()
	c.messageChan = messageChan
	c.svc.wsConnections.Store(connectionID, c)

	// Ensure cleanup when function exits
	defer func() {
		c.svc.wsConnections.Delete(connectionID)
		// NOTE: To avoid panics due to sending on a closed channel, we do not close the message channel
		// and instead let the gc reclaim it once no more goroutine is sending to it
	}()

	close(onSubscribed)

	// Receive execution-related messages for the app through gRPC, forwarded by the router.
	// The router selects only one gateway to handle a request from a pool of one or more workers (and thus WebSockets)
	// running for each app.
	for {
		select {
		case <-c.stopForwarding:
			c.log.Debug("connection marked as draining, stopping forwarding")
			return

		case msg, ok := <-messageChan:
			if !ok {
				c.log.ReportError(fmt.Errorf("close gRPC channel"), "BUG: message channel was closed unexpectedly - this should never happen")
				return
			}

			data := msg.Data

			rawBytes, err := proto.Marshal(data)
			if err != nil {
				// TODO This should never happen, we should likely push the message into a dead-letter queue.
				c.log.ReportError(err, "invalid protobuf received by grpc",
					logger.WithErrorReportTags(map[string]string{
						"gateway_id": c.conn.GatewayId.String(),
						"conn_id":    c.conn.ConnectionId.String(),
					}))
				msg.Result <- err
				continue
			}

			log := c.log.With(
				"app_id", data.AppId,
				"app_name", data.AppName,
				"req_id", data.RequestId,
				"fn_slug", data.FunctionSlug,
				"step_id", data.StepId,
				"run_id", data.RunId,
				"transport", "grpc",
			)

			// Block forwards while draining — wait for the new connection
			// to be READY before failing so the proxy re-route succeeds.
			if c.draining.Load() {
				<-c.stopForwarding
				log.Debug("rejecting forward, connection finished draining")
				msg.Result <- fmt.Errorf("connection is draining")
				continue
			}

			log.Trace("gateway received grpc message")
			grpcTags := map[string]any{
				"transport": "grpc",
			}
			for k, v := range additionalMetricsTags {
				grpcTags[k] = v
			}
			metrics.IncrConnectGatewayReceivedRouterGRPCMessageCounter(ctx, 1, metrics.CounterOpt{
				PkgName: pkgName,
				Tags:    grpcTags,
			})

			// Wait for WORKER_REQUEST_ACK in a goroutine so we don't
			// block the message loop from processing other requests.
			// IMPORTANT: Store before writing to the WebSocket to avoid a race
			// condition.
			ackCh := make(chan struct{})
			c.pendingAcks.Store(data.RequestId, ackCh)

			// Use a fresh context instead of the connection ctx. During a
			// Gateway drain, ctx is cancelled, which would fail this write even
			// though we already consumed the message from the channel. The 5s
			// timeout prevents goroutine leaks if the write hangs. `Forward()`
			// is blocked waiting on `msg.Err`, so a failure here correctly
			// propagates back to the executor.
			writeCtx, writeCancel := context.WithTimeout(
				context.Background(),
				5*time.Second,
			)
			err = wsproto.Write(writeCtx, c.ws, &connectpb.ConnectMessage{
				Kind:    connectpb.GatewayMessageType_GATEWAY_EXECUTOR_REQUEST,
				Payload: rawBytes,
			})
			writeCancel()
			if err != nil {
				c.pendingAcks.Delete(data.RequestId)
				msg.Result <- err
				if isConnectionClosedErr(err) {
					return
				}
				log.ReportError(err, "failed to forward message to worker")
				continue
			}

			go func() {
				select {
				case <-ackCh:
					msg.Result <- nil
				case <-time.After(consts.ConnectWorkerRequestLeaseDuration + consts.ConnectWorkerRequestGracePeriod):
					c.pendingAcks.Delete(data.RequestId)
					msg.Result <- fmt.Errorf("worker did not ACK request %s", data.RequestId)
					log.Warn("worker did not ACK request in time", "req_id", data.RequestId)
				}
			}()
		}
	}
}

func (c *connectionHandler) establishConnection(ctx context.Context) (*state.Connection, *connecterrors.SocketError) {
	var (
		initialMessageData connectpb.WorkerConnectRequestData
		initialMessage     connectpb.ConnectMessage
	)

	shorterContext, cancelShorter := context.WithTimeout(ctx, 5*time.Second)
	defer cancelShorter()

	err := wsproto.Read(shorterContext, c.ws, &initialMessage)
	if err != nil {
		if ctx.Err() != nil {
			return nil, &ErrDraining
		}

		if isConnectionClosedErr(err) {
			c.log.Warn("connection was closed during handshake")
			return nil, nil
		}

		code := syscode.CodeConnectInternal
		statusCode := websocket.StatusInternalError
		msg := err.Error()

		if errors.Is(err, context.DeadlineExceeded) {
			code = syscode.CodeConnectWorkerHelloTimeout
			statusCode = websocket.StatusPolicyViolation
			msg = "Timeout waiting for worker SDK connect message"

			c.log.Debug("Timeout waiting for worker SDK connect message")
		}

		return nil, &connecterrors.SocketError{
			SysCode:    code,
			StatusCode: statusCode,
			Msg:        msg,
		}
	}

	if initialMessage.Kind != connectpb.GatewayMessageType_WORKER_CONNECT {
		c.log.Debug("initial worker SDK message was not connect")

		return nil, &connecterrors.SocketError{
			SysCode:    syscode.CodeConnectWorkerHelloInvalidMsg,
			StatusCode: websocket.StatusPolicyViolation,
			Msg:        "Invalid first message, expected sdk-connect",
		}
	}

	if err := proto.Unmarshal(initialMessage.Payload, &initialMessageData); err != nil {
		c.log.Debug("initial SDK message contained invalid Protobuf")

		return nil, &connecterrors.SocketError{
			SysCode:    syscode.CodeConnectWorkerHelloInvalidPayload,
			StatusCode: websocket.StatusPolicyViolation,
			Msg:        "Invalid Protobuf in SDK connect message",
		}
	}

	// Ensure connection ID is valid ULID
	var connectionId ulid.ULID
	{
		if connectionId, err = ulid.Parse(initialMessageData.ConnectionId); err != nil {
			c.log.Debug("initial SDK message contained invalid connection ID")

			return nil, &connecterrors.SocketError{
				SysCode:    syscode.CodeConnectWorkerHelloInvalidPayload,
				StatusCode: websocket.StatusPolicyViolation,
				Msg:        "Invalid connection ID in SDK connect message",
			}
		}
	}

	// Ensure Instance ID is provided
	if initialMessageData.InstanceId == "" {
		c.log.Debug("initial SDK message missing instance ID")

		return nil, &connecterrors.SocketError{
			SysCode:    syscode.CodeConnectWorkerHelloInvalidPayload,
			StatusCode: websocket.StatusPolicyViolation,
			Msg:        "Missing instanceId in SDK connect message",
		}
	}

	var authResp *auth.Response
	{
		// Run auth, add to distributed state
		authResp, err = c.svc.auther(ctx, &initialMessageData)
		if err != nil {
			if ctx.Err() != nil {
				return nil, &ErrDraining
			}

			c.log.ReportError(err, "connect auth failed")
			return nil, &connecterrors.SocketError{
				SysCode:    syscode.CodeConnectInternal,
				StatusCode: websocket.StatusInternalError,
				Msg:        "Internal error",
			}
		}

		if authResp == nil {
			c.log.Debug("Auth failed")
			return nil, &connecterrors.SocketError{
				SysCode:    syscode.CodeConnectAuthFailed,
				StatusCode: websocket.StatusPolicyViolation,
				Msg:        "Authentication failed",
			}
		}

		c.log.Debug("SDK successfully authenticated", "authResp", authResp)
	}

	{
		limit := DefaultAppsPerConnection
		if authResp.Entitlements.AppsPerConnection != 0 {
			limit = authResp.Entitlements.AppsPerConnection
		}

		if len(initialMessageData.Apps) == 0 {
			return nil, &connecterrors.SocketError{
				SysCode:    syscode.CodeConnectWorkerHelloInvalidPayload,
				StatusCode: websocket.StatusPolicyViolation,
				Msg:        "Missing apps in SDK connect message",
			}
		}

		if len(initialMessageData.Apps) > limit {
			return nil, &connecterrors.SocketError{
				SysCode:    syscode.CodeConnectTooManyAppsPerConnection,
				StatusCode: websocket.StatusPolicyViolation,
				Msg:        fmt.Sprintf("You exceeded the max. number of allowed apps per connection (%d)", limit),
			}
		}
	}

	log := c.log.With(
		"account_id", authResp.AccountID,
		"env_id", authResp.EnvID,
		"connection_id", connectionId.String(),
		"instance_id", util.SanitizeLogField(initialMessageData.InstanceId),
		"worker_ip", util.SanitizeLogField(c.remoteAddr))

	workerGroups := make(map[string]*state.WorkerGroup)
	{

		eg := errgroup.Group{}

		lock := sync.Mutex{}

		for _, app := range initialMessageData.Apps {
			app := app
			eg.Go(func() error {
				workerGroup, err := state.NewWorkerGroupFromConnRequest(ctx, &initialMessageData, authResp, app)
				if err != nil {
					log.ReportError(err, "could not create worker group for request")
					return err
				}

				lock.Lock()
				workerGroups[workerGroup.Hash] = workerGroup
				lock.Unlock()

				return nil
			})
		}

		err := eg.Wait()
		if err != nil {
			if ctx.Err() != nil {
				return nil, &ErrDraining
			}

			return nil, &connecterrors.SocketError{
				SysCode:    syscode.CodeConnectInternal,
				StatusCode: websocket.StatusInternalError,
				Msg:        "Internal error",
			}
		}
	}

	conn := state.Connection{
		AccountID:    authResp.AccountID,
		EnvID:        authResp.EnvID,
		ConnectionId: connectionId,

		WorkerIP: c.remoteAddr,

		Data:   &initialMessageData,
		Groups: workerGroups,

		// Used for routing messages to the correct gateway
		GatewayId: c.svc.gatewayId,
	}
	c.conn = &conn
	c.bindConnectionLogger(&conn)

	{
		// This is a transactional operation, it should always complete regardless of context cancellation

		// Connection should always be upserted, we don't want inconsistent state
		if err := c.updateConnStatus(connectpb.ConnectionStatus_CONNECTED, "worker authenticated"); err != nil {
			c.log.ReportError(err, "adding connection state failed")
			return nil, &connecterrors.SocketError{
				SysCode:    syscode.CodeConnectInternal,
				StatusCode: websocket.StatusInternalError,
				Msg:        "connection not stored",
			}
		}

		// if the instance ID is not set, we return an error
		if initialMessageData.InstanceId == "" {
			return nil, &connecterrors.SocketError{
				SysCode:    syscode.CodeConnectInternal,
				StatusCode: websocket.StatusInternalError,
				Msg:        "instance ID is required",
			}
		}

		// Set worker capacity limit
		// If MaxWorkerConcurrency is 0, this will clear any existing capacity limit
		maxConcurrency := int64(0)
		if initialMessageData.MaxWorkerConcurrency != nil {
			maxConcurrency = *initialMessageData.MaxWorkerConcurrency
		}
		if err := c.svc.stateManager.SetWorkerTotalCapacity(context.Background(), authResp.EnvID, initialMessageData.InstanceId, maxConcurrency); err != nil {
			c.log.ReportError(err, "failed to set worker capacity")
			return nil, &connecterrors.SocketError{
				SysCode:    syscode.CodeConnectInternal,
				StatusCode: websocket.StatusInternalError,
				Msg:        "worker capacity not enforced",
			}
		}

		c.log.Trace("worker capacity set", "max_concurrency", maxConcurrency)

		// TODO Connection should not be marked as ready to receive traffic until the read loop is set up, sync is handled, and the client optionally sent a ready signal
		for _, l := range c.svc.lifecycles {
			go l.OnConnected(context.Background(), &conn)
		}
	}

	return &conn, nil
}

func (c *connectionHandler) handleSdkReply(ctx context.Context, msg *connectpb.ConnectMessage) error {
	data := &connectpb.SDKResponse{}
	if err := proto.Unmarshal(msg.Payload, data); err != nil {
		return fmt.Errorf("invalid response type: %w", err)
	}

	l := c.log.With(
		"status", data.Status.String(),
		"no_retry", data.NoRetry,
		"retry_after", data.RetryAfter,
		"app_id", data.AppId,
		"req_id", data.RequestId,
		"run_id", data.RunId,
	)

	l.Trace("saving response and notifying executor")

	// Persist response in buffer, which is polled by executor.
	err := c.svc.stateManager.SaveResponse(ctx, c.conn.EnvID, data.RequestId, data)
	if err != nil && !errors.Is(err, state.ErrResponseAlreadyBuffered) {
		return fmt.Errorf("could not save response: %w", err)
	}

	{
		// Send a best-effort gRPC message to fast-track the response,
		// this should be reliable enough but we still combine it with a reliable store like the buffer above.
		grpcClient, err := c.svc.getOrCreateGRPCClient(ctx, c.conn.EnvID, data.RequestId)

		switch {
		case err == nil:
			if _, err := grpcClient.Reply(ctx, &connectpb.ReplyRequest{Data: data}); err != nil {
				l.Warn("could not fast-track response through grpc, executor will poll buffered response", "err", err)
			}
		case errors.Is(err, state.ErrExecutorNotFound):
			l.Debug("executor not found in lease, reply was likely picked up by polling")
		default:
			l.Warn("could not create grpc client for sdk reply, executor will poll buffered response", "err", err)
		}
	}

	replyAck, err := proto.Marshal(&connectpb.WorkerReplyAckData{
		RequestId: data.RequestId,
	})
	if err != nil {
		return fmt.Errorf("could not marshal reply ack: %w", err)
	}

	writeCtx, writeCancel := context.WithTimeout(context.Background(), wsWriteTimeout)
	defer writeCancel()
	if err := wsproto.Write(writeCtx, c.ws, &connectpb.ConnectMessage{
		Kind:    connectpb.GatewayMessageType_WORKER_REPLY_ACK,
		Payload: replyAck,
	}); err != nil {
		l.Warn("could not send worker reply ack after response was buffered", "err", err)
	}

	return nil
}

func (c *connectionHandler) updateConnStatus(status connectpb.ConnectionStatus, reason string, attrs ...any) error {
	c.updateLock.Lock()
	defer c.updateLock.Unlock()

	// Always update the connection status, do not use context cancellation
	err := c.svc.stateManager.UpsertConnection(context.Background(), c.conn, status, time.Now())
	if err != nil {
		return err
	}

	c.logConnStatusLocked(status, reason, attrs...)
	return nil
}
