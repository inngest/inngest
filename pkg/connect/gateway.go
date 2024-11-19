package connect

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/gowebpki/jcs"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/connect/types"
	"github.com/inngest/inngest/pkg/connect/wsproto"
	"github.com/inngest/inngest/pkg/syscode"
	"github.com/inngest/inngest/proto/gen/connect/v1"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
)

func closeWithConnectError(ws *websocket.Conn, serr *SocketError) {
	// reason must be limited to 125 bytes and should not be dynamic,
	// so we restrict it to the known syscodes to prevent unintentional overflows
	_ = ws.Close(serr.StatusCode, serr.SysCode)
}

func (c *connectGatewaySvc) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set the context as needed. Use of r.Context() is not recommended
		// to avoid surprising behavior (see http.Hijacker).
		ctx, cancel := context.WithCancel(c.runCtx)
		defer cancel()

		ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			Subprotocols: []string{
				types.GatewaySubProtocol,
			},
		})
		if err != nil {
			return
		}
		defer func() {
			c.logger.Debug("Closing WebSocket connection")

			_ = ws.CloseNow()
		}()

		c.logger.Debug("WebSocket connection established, sending hello")

		{
			err = wsproto.Write(ctx, ws, &connect.ConnectMessage{
				Kind: connect.GatewayMessageType_GATEWAY_HELLO,
			})
			if err != nil {
				c.logger.Error("could not send hello", "err", err)
				closeWithConnectError(ws, &SocketError{
					SysCode:    syscode.CodeConnectInternal,
					StatusCode: websocket.StatusInternalError,
					Msg:        "could not send gateway hello",
				})

				return
			}
		}

		conn, serr := c.establishConnection(ctx, ws)
		if serr != nil {
			c.logger.Error("error establishing connection", "error", serr)
			closeWithConnectError(ws, serr)

			return
		}

		var closeReason string
		var closeReasonLock sync.Mutex

		log := c.logger.With("account_id", conn.Data.AuthData.AccountId)

		// Once connection is established, we must make sure to update the state on any disconnect,
		// regardless of whether it's permanent or temporary
		defer func() {
			err := c.stateManager.DeleteConnection(ctx, conn)
			switch err {
			case nil, state.ConnDeletedWithGroupErr:
				// no-op
			default:
				log.Error("error deleting connection from state", "error", err)
			}

			for _, lifecycle := range c.lifecycles {
				lifecycle.OnDisconnected(ctx, closeReason)
			}
		}()

		err = conn.Sync(ctx, c.stateManager)
		if err != nil {
			log.Error("error handling sync", "error", err)

			// Allow returning user-facing errors to hint about invalid config, etc.
			serr := SocketError{}
			if errors.As(err, &serr) {
				closeWithConnectError(ws, &serr)
				return
			}

			closeWithConnectError(ws, &SocketError{
				SysCode:    syscode.CodeConnectInternal,
				StatusCode: websocket.StatusInternalError,
				Msg:        "Internal error",
			})

			return
		}

		app, err := c.dbcqrs.GetAppByName(ctx, conn.Data.AppName)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			log.Error("could not get app by name", "appName", conn.Data.AppName, "err", err)
			closeWithConnectError(ws, &SocketError{
				SysCode:    syscode.CodeConnectInternal,
				StatusCode: websocket.StatusInternalError,
				Msg:        "could not get app by name",
			})
			return
		}

		if errors.Is(err, sql.ErrNoRows) || app == nil {
			log.Error("could not find app", "appName", conn.Data.AppName)
			closeWithConnectError(ws, &SocketError{
				SysCode:    syscode.CodeConnectInternal,
				StatusCode: websocket.StatusInternalError,
				Msg:        "could not find app",
			})
			return
		}

		log = log.With("app_id", app.ID)

		log.Debug("found app, connection is ready")

		eg := errgroup.Group{}

		// Wait for relevant messages and forward them over the WebSocket connection
		go c.receiveRouterMessages(ctx, app.ID, ws, log)

		// Run loop
		eg.Go(func() error {
			for {
				// If the context is canceled due to a shutdown or other reason, we need to close the connection
				// and let the client know we're going away intentionally.
				if ctx.Err() != nil {
					serr := &SocketError{
						SysCode:    syscode.CodeConnectGatewayClosing,
						StatusCode: websocket.StatusGoingAway,
						Msg:        "run loop context ended",
					}
					closeWithConnectError(ws, serr)
					return err
				}

				var msg connect.ConnectMessage
				err := wsproto.Read(ctx, ws, &msg)
				if err != nil {
					closeErr := websocket.CloseError{}
					if errors.As(err, &closeErr) {
						// If client connection closed unexpectedly, we should store the reason, if set.
						// If the reason is set, it may have been an intentional close, so the connection
						// may not be re-established.
						closeReasonLock.Lock()
						closeReason = closeErr.Reason
						closeReasonLock.Unlock()
						return closeErr
					}

					// If the parent context timed out or got canceled, we should signal the client that we're going away,
					// and it should reconnect to another gateway.
					if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
						serr := &SocketError{
							SysCode:    syscode.CodeConnectGatewayClosing,
							StatusCode: websocket.StatusGoingAway,
							Msg:        "read context ended",
						}
						closeWithConnectError(ws, serr)
						return err
					}

					log.Error("failed to read websocket message", "err", err)

					// If we failed to read the message for another reason, we should probably reconnect as well.
					closeWithConnectError(ws, &SocketError{
						SysCode:    syscode.CodeConnectInternal,
						StatusCode: websocket.StatusInternalError,
						Msg:        "could not read next message",
					})
				}

				serr := c.handleIncomingWebSocketMessage(ctx, app.ID, &msg, log)
				if serr != nil {
					closeWithConnectError(ws, serr)
					return serr
				}
			}
		})

		// Let the worker know we're ready to receive messages
		{
			err = wsproto.Write(ctx, ws, &connect.ConnectMessage{
				Kind: connect.GatewayMessageType_GATEWAY_CONNECTION_READY,
			})
			if err != nil {
				c.logger.Error("could not send connection ready", "err", err)
				closeWithConnectError(ws, &SocketError{
					SysCode:    syscode.CodeConnectInternal,
					StatusCode: websocket.StatusInternalError,
					Msg:        "could not send gateway connection ready",
				})

				return
			}
		}

		// TODO Update connection state to start receiving messages
		// TODO Mark connection as ready to receive traffic unless we require manual client ready signal (optional)

		if err := eg.Wait(); err != nil {
			return
		}
	})
}

func (c *connectGatewaySvc) handleIncomingWebSocketMessage(ctx context.Context, appId uuid.UUID, msg *connect.ConnectMessage, log *slog.Logger) *SocketError {
	log.Debug("received WebSocket message", "kind", msg.Kind.String())

	switch msg.Kind {
	case connect.GatewayMessageType_WORKER_REQUEST_ACK:
		{
			var data connect.WorkerRequestAckData
			if err := proto.Unmarshal(msg.Payload, &data); err != nil {
				// This should never happen: Failing the ack means we will redeliver the same request even though
				// the worker already started processing it.
				return &SocketError{
					SysCode:    syscode.CodeConnectWorkerRequestAckInvalidPayload,
					StatusCode: websocket.StatusPolicyViolation,
					Msg:        "invalid payload in worker request ack",
				}
			}

			// This will be sent exactly once, as the router selected this gateway to handle the request
			err := c.receiver.AckMessage(ctx, appId, data.RequestId)
			if err != nil {
				// This should never happen: Failing the ack means we will redeliver the same request even though
				// the worker already started processing it.
				log.Error("failed to ack message", "err", err)
				return &SocketError{
					SysCode:    syscode.CodeConnectInternal,
					StatusCode: websocket.StatusInternalError,
					Msg:        "could not ack message",
				}
			}

			// TODO Should we send a reverse ack to the worker to start processing the request?

			return nil
		}
	case connect.GatewayMessageType_WORKER_REPLY:
		// Handle SDK reply
		err := c.handleSdkReply(ctx, log, appId, msg)
		if err != nil {
			// TODO Should we actually close the connection here?
			return &SocketError{
				SysCode:    syscode.CodeConnectInternal,
				StatusCode: websocket.StatusInternalError,
				Msg:        "could not handle SDK reply",
			}
		}
	default:
		// TODO Should we actually close the connection here?
		return &SocketError{
			SysCode:    syscode.CodeConnectRunInvalidMessage,
			StatusCode: websocket.StatusPolicyViolation,
			Msg:        fmt.Sprintf("invalid message kind %q", msg.Kind),
		}
	}

	return nil
}

func (c *connectGatewaySvc) receiveRouterMessages(ctx context.Context, appId uuid.UUID, ws *websocket.Conn, log *slog.Logger) {
	// Receive execution-related messages for the app, forwarded by the router.
	// The router selects only one gateway to handle a request from a pool of one or more workers (and thus WebSockets)
	// running for each app.
	err := c.receiver.ReceiveRoutedRequest(ctx, c.gatewayId, appId, func(rawBytes []byte, data *connect.GatewayExecutorRequestData) {
		log := log.With(
			"req_id", data.RequestId,
			"fn_slug", data.FunctionSlug,
			"step_id", data.StepId,
		)

		log.Debug("gateway received msg")

		// Forward message to SDK!
		err := wsproto.Write(ctx, ws, &connect.ConnectMessage{
			Kind:    connect.GatewayMessageType_GATEWAY_EXECUTOR_REQUEST,
			Payload: rawBytes,
		})
		if err != nil {
			log.Error("failed to forward message to worker", "err", err)

			// TODO The connection cannot be used, we need to let the executor know!
			return
		}
	})
	if err != nil {
		log.Error("failed to receive routed requests", "err", err)

		// TODO Log error, retry?
		return
	}
}

func (c *connectGatewaySvc) establishConnection(ctx context.Context, ws *websocket.Conn) (*state.Connection, *SocketError) {
	var (
		initialMessageData connect.WorkerConnectRequestData
		initialMessage     connect.ConnectMessage
	)

	shorterContext, cancelShorter := context.WithTimeout(ctx, 5*time.Second)
	defer cancelShorter()

	err := wsproto.Read(shorterContext, ws, &initialMessage)
	if err != nil {
		code := syscode.CodeConnectInternal
		statusCode := websocket.StatusInternalError
		msg := err.Error()

		if errors.Is(err, context.DeadlineExceeded) {
			code = syscode.CodeConnectWorkerHelloTimeout
			statusCode = websocket.StatusPolicyViolation
			msg = "Timeout waiting for worker SDK connect message"

			c.logger.Debug("Timeout waiting for worker SDK connect message")
		}

		return nil, &SocketError{
			SysCode:    code,
			StatusCode: statusCode,
			Msg:        msg,
		}
	}

	if initialMessage.Kind != connect.GatewayMessageType_WORKER_CONNECT {
		c.logger.Debug("initial worker SDK message was not connect")

		return nil, &SocketError{
			SysCode:    syscode.CodeConnectWorkerHelloInvalidMsg,
			StatusCode: websocket.StatusPolicyViolation,
			Msg:        "Invalid first message, expected sdk-connect",
		}
	}

	if err := proto.Unmarshal(initialMessage.Payload, &initialMessageData); err != nil {
		c.logger.Debug("initial SDK message contained invalid Protobuf")

		return nil, &SocketError{
			SysCode:    syscode.CodeConnectWorkerHelloInvalidPayload,
			StatusCode: websocket.StatusPolicyViolation,
			Msg:        "Invalid Protobuf in SDK connect message",
		}
	}

	var authResp *AuthResponse
	{
		// Run auth, add to distributed state
		authResp, err = c.auther(ctx, &initialMessageData)
		if err != nil {
			c.logger.Error("connect auth failed", "err", err)
			return nil, &SocketError{
				SysCode:    syscode.CodeConnectInternal,
				StatusCode: websocket.StatusInternalError,
				Msg:        "Internal error",
			}
		}

		if authResp == nil {
			c.logger.Debug("Auth failed")
			return nil, &SocketError{
				SysCode:    syscode.CodeConnectAuthFailed,
				StatusCode: websocket.StatusPolicyViolation,
				Msg:        "Authentication failed",
			}
		}

		initialMessageData.AuthData.AccountId = authResp.AccountID.String()
		initialMessageData.AuthData.EnvId = authResp.EnvID.String()

		c.logger.Debug("SDK successfully authenticated", "authResp", authResp)
	}

	log := c.logger.With("account_id", initialMessageData.AuthData.AccountId)

	var functionHash []byte
	{
		b, err := jcs.Transform(initialMessageData.Config.Functions)
		if err != nil {
			c.logger.Error("transforming function config failed", "err", err)
			return nil, &SocketError{
				SysCode:    syscode.CodeConnectInternal,
				StatusCode: websocket.StatusInternalError,
				Msg:        "Internal error",
			}
		}

		res := sha256.Sum256(b)
		functionHash = res[:]
	}

	sessionDetails := &connect.SessionDetails{
		SessionId:    initialMessageData.SessionId,
		FunctionHash: functionHash,
	}

	workerGroup, err := NewWorkerGroupFromConnRequest(ctx, &initialMessageData, authResp, sessionDetails)
	if err != nil {
		log.Error("could not create worker group for request", "err", err)
		return nil, &SocketError{
			SysCode:    syscode.CodeConnectInternal,
			StatusCode: websocket.StatusInternalError,
			Msg:        "Internal error",
		}
	}

	conn := state.Connection{
		Data:    &initialMessageData,
		Session: sessionDetails,
		Group:   workerGroup,
	}

	if err := c.stateManager.AddConnection(ctx, &conn); err != nil {
		log.Error("adding connection state failed", "err", err)
		return nil, &SocketError{
			SysCode:    syscode.CodeConnectInternal,
			StatusCode: websocket.StatusInternalError,
			Msg:        "connection not stored",
		}
	}

	// TODO Connection should not be marked as ready to receive traffic until the read loop is set up, sync is handled, and the client optionally sent a ready signal
	for _, l := range c.lifecycles {
		go l.OnConnected(ctx, &conn)
	}

	return &conn, nil
}

func (c *connectGatewaySvc) handleSdkReply(ctx context.Context, log *slog.Logger, appId uuid.UUID, msg *connect.ConnectMessage) error {
	var data connect.SDKResponse
	if err := proto.Unmarshal(msg.Payload, &data); err != nil {
		return fmt.Errorf("invalid response type: %w", err)
	}

	log.Debug("notifying executor about response", "status", data.Status.String(), "no_retry", data.NoRetry, "retry_after", data.RetryAfter)

	err := c.receiver.NotifyExecutor(ctx, appId, &data)
	if err != nil {
		return fmt.Errorf("could not notify executor: %w", err)
	}

	return nil
}
