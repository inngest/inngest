package connect

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	connecterrors "github.com/inngest/inngest/pkg/connect/errors"
	"github.com/inngest/inngest/pkg/consts"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/inngest/inngest/pkg/connect/auth"
	"github.com/inngest/inngest/pkg/connect/pubsub"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/connect/types"
	"github.com/inngest/inngest/pkg/connect/wsproto"
	"github.com/inngest/inngest/pkg/syscode"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
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

func (c *connectGatewaySvc) closeWithConnectError(ws *websocket.Conn, serr *connecterrors.SocketError) {
	// reason must be limited to 125 bytes and should not be dynamic,
	// so we restrict it to the known syscodes to prevent unintentional overflows
	err := ws.Close(serr.StatusCode, serr.SysCode)
	if errors.Is(err, context.DeadlineExceeded) {
		return
	}

	if isConnectionClosedErr(err) {
		c.logger.Error("could not close WebSocket connection", "err", err, "serr", serr)
	}
}

// connectionHandler holds the WebSocket and current connection, if the connection was established.
type connectionHandler struct {
	svc  *connectGatewaySvc
	conn *state.Connection
	ws   *websocket.Conn

	updateLock sync.Mutex
	log        *slog.Logger

	remoteAddr string

	lastHeartbeatLock       sync.Mutex
	lastHeartbeatReceivedAt time.Time
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
		if c.isDraining {
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
			svc:        c,
			log:        c.logger,
			ws:         ws,
			updateLock: sync.Mutex{},
			remoteAddr: remoteAddr,
		}

		closeReason := connectpb.WorkerDisconnectReason_UNEXPECTED.String()
		var closeReasonLock sync.Mutex
		setCloseReason := func(reason string) {
			closeReasonLock.Lock()
			defer closeReasonLock.Unlock()

			if reason != connectpb.WorkerDisconnectReason_UNEXPECTED.String() {
				closeReason = reason
			}
		}

		c.connectionCount.Add()
		defer func() {
			// This is deferred so we always update the semaphore
			defer c.connectionCount.Done()
			ch.log.Debug("Closing WebSocket connection", "reason", closeReason)
			if c.devlogger != nil {
				c.devlogger.Info().Msg("worker disconnected")
			}

			closed = true

			if c.isDraining {
				c.closeDraining(ws)
				return
			}

			_ = ws.CloseNow()
		}()

		ch.log.Debug("WebSocket connection established, sending hello")

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

				ch.log.Error("could not send hello", "err", err)
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
			ch.log.Error("error establishing connection", "error", serr)
			c.closeWithConnectError(ws, serr)

			return
		}

		// Connection was closed during the handshake process
		if conn == nil && serr == nil {
			return
		}

		ch.log = ch.log.With("account_id", conn.AccountID, "env_id", conn.EnvID, "conn_id", conn.ConnectionId)

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
			ch.log.Debug("context done, starting draining process")

			// Prevent routing any more messages to this connection
			err = ch.updateConnStatus(connectpb.ConnectionStatus_DRAINING)
			if err != nil {
				ch.log.Error("could not update connection status after context done", "err", err)
			}

			for _, l := range c.lifecycles {
				go l.OnStartDraining(context.Background(), conn)
			}

			setCloseReason(connectpb.WorkerDisconnectReason_GATEWAY_DRAINING.String())

			// Close WS connection once worker established another connection
			defer func() {
				_ = ws.CloseNow()
			}()

			// If the parent context timed out or got canceled, we should signal the client that we're going away,
			// and it should reconnect to another gateway.
			err := wsproto.Write(context.Background(), ws, &connectpb.ConnectMessage{
				Kind: connectpb.GatewayMessageType_GATEWAY_CLOSING,
			})
			if err != nil {
				return
			}

			select {
			case <-workerDrainedCtx.Done():
				ch.log.Debug("worker closed connection")
			case <-time.After(5 * time.Second):
				ch.log.Debug("reached timeout waiting for worker to close connection")
				// On timeout, the gateway forcefully closes the connection
				c.closeDraining(ws)
			}
		}()

		// Once a connection is established, we must make sure to update the state on any disconnect,
		// regardless of whether it's permanent or temporary
		defer func() {
			// This is a transactional operation, it should always complete regardless of context cancellation
			err := c.stateManager.DeleteConnection(context.Background(), conn.EnvID, conn.ConnectionId)
			if err != nil {
				ch.log.Error("error deleting connection from state", "error", err)
			}

			for _, lifecycle := range c.lifecycles {
				lifecycle.OnDisconnected(context.Background(), conn, closeReason)
			}
		}()

		{
			eg := errgroup.Group{}
			eg.SetLimit(10) // Limit concurrent syncs
			for _, group := range conn.Groups {
				group := group
				eg.Go(func() error {
					err := group.Sync(ctx, c.stateManager, c.apiBaseUrl, conn.Data)
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

				ch.log.Error("error handling sync", "error", err)

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
			if err := c.stateManager.UpsertConnection(context.Background(), conn, connectpb.ConnectionStatus_CONNECTED, time.Now()); err != nil {
				ch.log.Error("updating connection state after sync failed", "err", err)
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
			onSubscribed := make(chan struct{})
			// Wait for relevant messages and forward them over the WebSocket connection
			go ch.receiveRouterMessages(ctx, onSubscribed)

			<-onSubscribed
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
					if err := ch.updateConnStatus(connectpb.ConnectionStatus_DISCONNECTING); err != nil {
						ch.log.Error("could not update connection status after read error", "err", err)
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
						if closeErr.Code == websocket.StatusNoStatusRcvd && closeErr.Reason == "" {
							return nil
						}

						// Force-closed during draining after timeout
						if closeErr.Code == ErrDraining.StatusCode && closeErr.Reason == ErrDraining.SysCode {
							return nil
						}

						ch.log.Debug("connection closed with code and reason", "code", closeErr.Code.String(), "reason", closeErr.Reason)

						// Expected worker shutdown
						if closeErr.Code == websocket.StatusNormalClosure && closeErr.Reason == connectpb.WorkerDisconnectReason_WORKER_SHUTDOWN.String() {
							setCloseReason(connectpb.WorkerDisconnectReason_WORKER_SHUTDOWN.String())
							return nil
						}

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
					if strings.HasPrefix(err.Error(), "read limited at") {
						setCloseReason(connectpb.WorkerDisconnectReason_MESSAGE_TOO_LARGE.String())
						return nil
					}

					ch.log.Error("failed to read websocket message", "err", err)

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

				serr := ch.handleIncomingWebSocketMessage(ctx, &msg)
				if serr != nil {
					c.closeWithConnectError(ws, serr)
					return serr
				}
			}
		})

		// Let the worker know we're ready to receive messages
		{
			readyPayload, err := proto.Marshal(&connectpb.GatewayConnectionReadyData{
				HeartbeatInterval:   c.workerHeartbeatInterval.String(),
				ExtendLeaseInterval: c.workerRequestExtendLeaseInterval.String(),
			})
			if err != nil {
				ch.log.Error("could not marshal connection ready", "err", err)
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

				ch.log.Error("could not send connection ready", "err", err)
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
			err = c.stateManager.UpsertConnection(ctx, conn, connectpb.ConnectionStatus_READY, time.Now())
			if err != nil {
				if ctx.Err() != nil {
					c.closeDraining(ws)
					return
				}

				ch.log.Error("could not update connection status", "err", err)
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
			if c.devlogger != nil {
				c.devlogger.Info().Strs("app_names", conn.AppNames()).Msg("worker connected")
			}

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
			ch.log.Error("error in run loop", "err", err)
			return
		}
	})
}

func (c *connectionHandler) handleIncomingWebSocketMessage(ctx context.Context, msg *connectpb.ConnectMessage) *connecterrors.SocketError {
	c.log.Debug("received WebSocket message", "kind", msg.Kind.String())

	switch msg.Kind {
	case connectpb.GatewayMessageType_WORKER_READY:
		if c.svc.isDraining {
			return &ErrDraining
		}

		err := c.updateConnStatus(connectpb.ConnectionStatus_READY)
		if err != nil {
			// TODO Should we actually close the connection here?
			return &connecterrors.SocketError{
				SysCode:    syscode.CodeConnectInternal,
				StatusCode: websocket.StatusInternalError,
				Msg:        "could not update connection status",
			}
		}

		for _, l := range c.svc.lifecycles {
			go l.OnReady(context.Background(), c.conn)
		}

		return nil
	case connectpb.GatewayMessageType_WORKER_HEARTBEAT:
		if c.svc.isDraining {
			return &ErrDraining
		}

		err := c.updateConnStatus(connectpb.ConnectionStatus_READY)
		if err != nil {
			// TODO Should we actually close the connection here?
			return &connecterrors.SocketError{
				SysCode:    syscode.CodeConnectInternal,
				StatusCode: websocket.StatusInternalError,
				Msg:        "could not update connection status",
			}
		}

		for _, l := range c.svc.lifecycles {
			go l.OnHeartbeat(context.Background(), c.conn)
		}

		// Respond with gateway heartbeat
		if err := wsproto.Write(ctx, c.ws, &connectpb.ConnectMessage{
			Kind: connectpb.GatewayMessageType_GATEWAY_HEARTBEAT,
		}); err != nil {
			// The connection will fail to read and be closed in the read loop
			return nil
		}

		c.setLastHeartbeat(time.Now())

		return nil
	case connectpb.GatewayMessageType_WORKER_PAUSE:
		if c.svc.isDraining {
			return &ErrDraining
		}

		err := c.updateConnStatus(connectpb.ConnectionStatus_DRAINING)
		if err != nil {
			// TODO Should we actually close the connection here?
			return &connecterrors.SocketError{
				SysCode:    syscode.CodeConnectInternal,
				StatusCode: websocket.StatusInternalError,
				Msg:        "could not update connection status",
			}
		}

		for _, l := range c.svc.lifecycles {
			go l.OnStartDraining(context.Background(), c.conn)
		}

		return nil
	case connectpb.GatewayMessageType_WORKER_REQUEST_ACK:
		{
			var data connectpb.WorkerRequestAckData
			if err := proto.Unmarshal(msg.Payload, &data); err != nil {
				// This should never happen: Failing the ack means we will redeliver the same request even though
				// the worker already started processing it.
				return &connecterrors.SocketError{
					SysCode:    syscode.CodeConnectWorkerRequestAckInvalidPayload,
					StatusCode: websocket.StatusPolicyViolation,
					Msg:        "invalid payload in worker request ack",
				}
			}

			// This will be sent exactly once, as the router selected this gateway to handle the request
			// Even if the gateway is draining, we should ack the message, the SDK will buffer messages and use a new connection to report results
			err := c.svc.receiver.AckMessage(context.Background(), data.RequestId, pubsub.AckSourceWorker)
			if err != nil {
				// This should never happen: Failing the ack means we will redeliver the same request even though
				// the worker already started processing it.
				c.log.Error("failed to ack message", "err", err)
				return &connecterrors.SocketError{
					SysCode:    syscode.CodeConnectInternal,
					StatusCode: websocket.StatusInternalError,
					Msg:        "could not ack message",
				}
			}

			c.log.Debug("worker acked message",
				"req_id", data.RequestId,
				"run_id", data.RunId,
			)

			// TODO Should we send a reverse ack to the worker to start processing the request?

			return nil
		}
	case connectpb.GatewayMessageType_WORKER_REPLY:
		// Always handle SDK reply, even if gateway is draining
		err := c.handleSdkReply(context.Background(), msg)
		if err != nil {
			c.log.Error("could not handle sdk reply", "err", err)
			// TODO Should we actually close the connection here?
			return &connecterrors.SocketError{
				SysCode:    syscode.CodeConnectInternal,
				StatusCode: websocket.StatusInternalError,
				Msg:        "could not handle SDK reply",
			}
		}
	case connectpb.GatewayMessageType_WORKER_REQUEST_EXTEND_LEASE:
		{
			var data connectpb.WorkerRequestExtendLeaseData
			if err := proto.Unmarshal(msg.Payload, &data); err != nil {
				// This should never happen: Failing the ack means we will redeliver the same request even though
				// the worker already started processing it.
				return &connecterrors.SocketError{
					SysCode:    syscode.CodeConnectWorkerRequestExtendLeaseInvalidPayload,
					StatusCode: websocket.StatusPolicyViolation,
					Msg:        "invalid payload in worker request extend lease",
				}
			}

			leaseID, err := ulid.Parse(data.LeaseId)
			if err != nil {
				return &connecterrors.SocketError{
					SysCode:    syscode.CodeConnectWorkerRequestExtendLeaseInvalidPayload,
					StatusCode: websocket.StatusPolicyViolation,
					Msg:        "invalid lease ID in worker request extend lease payload",
				}
			}

			newLeaseID, err := c.svc.stateManager.ExtendRequestLease(ctx, c.conn.EnvID, data.RequestId, leaseID, consts.ConnectWorkerRequestLeaseDuration)
			if err != nil {
				if errors.Is(err, state.ErrRequestLeaseExpired) || errors.Is(err, state.ErrRequestLeased) {
					c.log.Error("lease was claimed by other worker or expired", "err", err, "req_id", data.RequestId, "lease_id", leaseID.String())

					// Respond with nack
					nackPayload, marshalErr := proto.Marshal(&connectpb.WorkerRequestExtendLeaseAckData{
						RequestId:    data.RequestId,
						AccountId:    data.AccountId,
						EnvId:        data.EnvId,
						AppId:        data.AppId,
						FunctionSlug: data.FunctionSlug,

						// No new lease ID
						NewLeaseId: nil,
					})
					if marshalErr != nil {
						// This should never happen
						return &connecterrors.SocketError{
							SysCode:    syscode.CodeConnectInternal,
							StatusCode: websocket.StatusInternalError,
							Msg:        "failed to marshal nack payload",
						}
					}

					if err := wsproto.Write(ctx, c.ws, &connectpb.ConnectMessage{
						Kind:    connectpb.GatewayMessageType_WORKER_REQUEST_EXTEND_LEASE_ACK,
						Payload: nackPayload,
					}); err != nil {
						// The connection will fail to read and be closed in the read loop
						return nil
					}

					return nil
				}

				c.log.Error("unexpected error extending lease", "err", err)

				// This should never happen
				return &connecterrors.SocketError{
					SysCode:    syscode.CodeConnectInternal,
					StatusCode: websocket.StatusInternalError,
					Msg:        "unexpected error extending lease",
				}
			}

			var newLeaseIDStr *string
			if newLeaseID != nil {
				newLeaseIDStr = proto.String(newLeaseID.String())
			}

			// Respond with ack
			ackPayload, marshalErr := proto.Marshal(&connectpb.WorkerRequestExtendLeaseAckData{
				RequestId:    data.RequestId,
				AccountId:    data.AccountId,
				EnvId:        data.EnvId,
				AppId:        data.AppId,
				FunctionSlug: data.FunctionSlug,
				NewLeaseId:   newLeaseIDStr,
			})
			if marshalErr != nil {
				// This should never happen
				return &connecterrors.SocketError{
					SysCode:    syscode.CodeConnectInternal,
					StatusCode: websocket.StatusInternalError,
					Msg:        "failed to marshal nack payload",
				}
			}

			if err := wsproto.Write(ctx, c.ws, &connectpb.ConnectMessage{
				Kind:    connectpb.GatewayMessageType_WORKER_REQUEST_EXTEND_LEASE_ACK,
				Payload: ackPayload,
			}); err != nil {
				// The connection will fail to read and be closed in the read loop
				return nil
			}

			c.log.Debug("extended lease for long-running request")

			// Extended lease, all good
			return nil
		}
	default:
		c.log.Warn("unexpected message kind received", "kind", msg.Kind.String())
		return nil
	}

	return nil
}

func (c *connectionHandler) receiveRouterMessages(ctx context.Context, onSubscribed chan struct{}) {
	additionalMetricsTags := c.svc.metricsTags()

	// Receive execution-related messages for the app, forwarded by the router.
	// The router selects only one gateway to handle a request from a pool of one or more workers (and thus WebSockets)
	// running for each app.
	err := c.svc.receiver.ReceiveRoutedRequest(ctx, c.svc.gatewayId, c.conn.ConnectionId, func(rawBytes []byte, data *connectpb.GatewayExecutorRequestData) {
		log := c.log.With(
			"app_id", data.AppId,
			"app_name", data.AppName,
			"req_id", data.RequestId,
			"fn_slug", data.FunctionSlug,
			"step_id", data.StepId,
			"run_id", data.RunId,
		)

		log.Debug("gateway received msg")

		metrics.IncrConnectGatewayReceivedRouterPubSubMessageCounter(ctx, 1, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    additionalMetricsTags,
		})

		// Do not forward messages if the connection is already draining
		if ctx.Err() != nil {
			log.Warn("connection is draining, not forwarding message")
			return
		}

		err := c.svc.receiver.AckMessage(ctx, data.RequestId, pubsub.AckSourceGateway)
		if err != nil {
			log.Error("failed to ack message", "err", err)
			// The executor will retry the message if it doesn't receive an ack
			return
		}

		// Do not forward messages if the connection is already draining
		if ctx.Err() != nil {
			log.Warn("acked message but connection is draining, not forwarding message")
			return
		}

		// Forward message to SDK!
		err = wsproto.Write(ctx, c.ws, &connectpb.ConnectMessage{
			Kind:    connectpb.GatewayMessageType_GATEWAY_EXECUTOR_REQUEST,
			Payload: rawBytes,
		})
		if err != nil {
			if isConnectionClosedErr(err) {
				return
			}

			log.Error("failed to forward message to worker", "err", err)

			// The connection cannot be used, the next read loop will run into the connection error and close the connection.
			// If the worker receives the message, it will send an ack through a new connection. Otherwise, the message will be redelivered.
			return
		}

		log.Debug("forwarded message to worker")
	}, onSubscribed)
	if err != nil {
		c.log.Error("failed to receive routed requests", "err", err)

		// TODO Log error, retry?
		return
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
		if isConnectionClosedErr(err) {
			c.log.Warn("connection was closed during handshake")
			return nil, nil
		}

		if ctx.Err() != nil {
			return nil, &ErrDraining
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

			c.log.Error("connect auth failed", "err", err)
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

		if len(initialMessageData.Apps) > limit {
			return nil, &connecterrors.SocketError{
				SysCode:    syscode.CodeConnectTooManyAppsPerConnection,
				StatusCode: websocket.StatusPolicyViolation,
				Msg:        fmt.Sprintf("You exceeded the max. number of allowed apps per connection (%d)", limit),
			}
		}
	}

	log := c.log.With("account_id", authResp.AccountID, "env_id", authResp.EnvID)

	workerGroups := make(map[string]*state.WorkerGroup)
	{

		eg := errgroup.Group{}

		lock := sync.Mutex{}

		for _, app := range initialMessageData.Apps {
			app := app
			eg.Go(func() error {
				workerGroup, err := state.NewWorkerGroupFromConnRequest(ctx, &initialMessageData, authResp, app)
				if err != nil {
					log.Error("could not create worker group for request", "err", err)
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

	{
		// This is a transactional operation, it should always complete regardless of context cancellation

		// Connection should always be upserted, we don't want inconsistent state
		if err := c.svc.stateManager.UpsertConnection(context.Background(), &conn, connectpb.ConnectionStatus_CONNECTED, time.Now()); err != nil {
			log.Error("adding connection state failed", "err", err)
			return nil, &connecterrors.SocketError{
				SysCode:    syscode.CodeConnectInternal,
				StatusCode: websocket.StatusInternalError,
				Msg:        "connection not stored",
			}
		}

		// TODO Connection should not be marked as ready to receive traffic until the read loop is set up, sync is handled, and the client optionally sent a ready signal
		for _, l := range c.svc.lifecycles {
			go l.OnConnected(context.Background(), &conn)
		}
	}

	c.conn = &conn

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

	l.Debug("saving response and notifying executor")

	// Persist response in buffer, which is polled by executor.
	err := c.svc.stateManager.SaveResponse(ctx, c.conn.EnvID, data.RequestId, data)
	if err != nil {
		return fmt.Errorf("could not save response: %w", err)
	}

	// Send a best-effort PubSub message to fast-track the response,
	// this is unreliable and must be combined with a reliable store like the buffer above.
	err = c.svc.receiver.NotifyExecutor(ctx, data)
	if err != nil {
		return fmt.Errorf("could not notify executor: %w", err)
	}

	replyAck, err := proto.Marshal(&connectpb.WorkerReplyAckData{
		RequestId: data.RequestId,
	})
	if err != nil {
		return fmt.Errorf("could not marshal reply ack: %w", err)
	}

	if err := wsproto.Write(ctx, c.ws, &connectpb.ConnectMessage{
		Kind:    connectpb.GatewayMessageType_WORKER_REPLY_ACK,
		Payload: replyAck,
	}); err != nil {
		return fmt.Errorf("could not send reply ack: %w", err)
	}

	return nil
}

func (c *connectionHandler) updateConnStatus(status connectpb.ConnectionStatus) error {
	c.updateLock.Lock()
	defer c.updateLock.Unlock()

	// Always update the connection status, do not use context cancellation
	return c.svc.stateManager.UpsertConnection(context.Background(), c.conn, status, time.Now())
}
