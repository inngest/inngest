package inngestgo

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/coder/websocket"
	"github.com/inngest/inngest/pkg/connect/types"
	"github.com/inngest/inngest/pkg/connect/wsproto"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/inngest/inngest/pkg/syscode"
	connectproto "github.com/inngest/inngest/proto/gen/connect/v1"
	sdkerrors "github.com/inngest/inngestgo/errors"
	"github.com/pbnjay/memory"
	"google.golang.org/protobuf/proto"
	"io"
	"net/url"
	"runtime"
	"sync"

	"github.com/inngest/inngestgo/internal/sdkrequest"
	"github.com/oklog/ulid/v2"
	"os"
	"strings"
	"time"
)

type connectHandler struct {
	h *handler

	connectionId ulid.ULID

	messageBuffer     []*connectproto.ConnectMessage
	messageBufferLock sync.Mutex
}

// authContext is wrapper for information related to authentication
type authContext struct {
	signingKey string
	fallback   bool
}

func (h *connectHandler) connectURLs() []string {
	if h.h.isDev() {
		return []string{fmt.Sprintf("%s/connect", strings.Replace(DevServerURL(), "http", "ws", 1))}
	}

	if len(h.h.ConnectURLs) > 0 {
		return h.h.ConnectURLs
	}

	return nil
}

func (h *connectHandler) connectToGateway(ctx context.Context) (*websocket.Conn, error) {
	hosts := h.connectURLs()
	if len(hosts) == 0 {
		return nil, fmt.Errorf("no connect URLs provided")
	}

	for _, gatewayHost := range hosts {
		h.h.Logger.Debug("attempting to connect", "host", gatewayHost)

		// Establish WebSocket connection to one of the gateways
		ws, _, err := websocket.Dial(ctx, gatewayHost, &websocket.DialOptions{
			Subprotocols: []string{
				types.GatewaySubProtocol,
			},
		})
		if err != nil {
			// try next connection
			continue
		}

		return ws, nil
	}

	return nil, fmt.Errorf("could not establish outbound connection: no available gateway host")
}

func (h *handler) Connect(ctx context.Context) error {
	h.useConnect = true
	ch := connectHandler{h: h}
	return ch.Connect(ctx)
}

func (h *connectHandler) instanceId() string {
	if h.h.InstanceId != nil {
		return *h.h.InstanceId
	}

	hostname, _ := os.Hostname()
	if hostname != "" {
		return hostname
	}

	// TODO Is there any stable identifier that can be used as a fallback?
	return "<missing-instance-id>"
}

func (h *connectHandler) Connect(ctx context.Context) error {
	signingKey := h.h.GetSigningKey()
	if signingKey == "" {
		return fmt.Errorf("must provide signing key")
	}
	auth := authContext{signingKey: signingKey}

	numCpuCores := runtime.NumCPU()
	totalMem := memory.TotalMemory()

	connectPlaceholder := url.URL{
		Scheme: "ws",
		Host:   "connect",
	}

	fns, err := createFunctionConfigs(h.h.appName, h.h.funcs, connectPlaceholder, true)
	if err != nil {
		return fmt.Errorf("error creating function configs: %w", err)
	}

	marshaledFns, err := json.Marshal(fns)
	if err != nil {
		return fmt.Errorf("failed to serialize connect config: %w", err)
	}

	marshaledCapabilities, err := json.Marshal(capabilities)
	if err != nil {
		return fmt.Errorf("failed to serialize connect config: %w", err)
	}

	var attempts int
	for {
		attempts++

		shouldReconnect, err := h.connect(ctx, connectionEstablishData{
			signingKey:            auth.signingKey,
			numCpuCores:           int32(numCpuCores),
			totalMem:              int64(totalMem),
			marshaledFns:          marshaledFns,
			marshaledCapabilities: marshaledCapabilities,
		})

		h.h.Logger.Error("connect failed", "err", err, "reconnect", shouldReconnect)

		if !shouldReconnect {
			return err
		}

		closeErr := websocket.CloseError{}
		if errors.As(err, &closeErr) {
			switch closeErr.Reason {
			// If auth failed, retry with fallback key
			case syscode.CodeConnectAuthFailed:
				if auth.fallback {
					return fmt.Errorf("failed to authenticate with fallback key, exiting")
				}

				signingKeyFallback := h.h.GetSigningKeyFallback()
				if signingKeyFallback != "" {
					auth = authContext{signingKey: signingKeyFallback, fallback: true}
				}

				continue

			// Retry on the following error codes
			case syscode.CodeConnectGatewayClosing, syscode.CodeConnectInternal, syscode.CodeConnectWorkerHelloTimeout:
				continue

			default:
				// If we received a reason  that's non-retriable, stop here.
				return fmt.Errorf("connect failed with error code %q", closeErr.Reason)
			}
		}
	}
}

type connectionEstablishData struct {
	signingKey            string
	numCpuCores           int32
	totalMem              int64
	marshaledFns          []byte
	marshaledCapabilities []byte
}

func (h *connectHandler) connect(ctx context.Context, data connectionEstablishData) (reconnect bool, err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	connectTimeout, cancelConnectTimeout := context.WithTimeout(ctx, 10*time.Second)
	defer cancelConnectTimeout()

	ws, err := h.connectToGateway(connectTimeout)
	if err != nil {
		return false, fmt.Errorf("could not connect: %w", err)
	}
	defer func() {
		// TODO Do we need to include a reason here? If we only use this for unexpected disconnects, probably not
		_ = ws.CloseNow()
	}()

	// Connection ID is unique per connection, reconnections should get a new ID
	h.connectionId = ulid.MustNew(ulid.Now(), rand.Reader)

	h.h.Logger.Debug("connection established")

	// Wait for gateway hello message
	{
		initialMessageTimeout, cancelInitialTimeout := context.WithTimeout(ctx, 5*time.Second)
		defer cancelInitialTimeout()
		var helloMessage connectproto.ConnectMessage
		err = wsproto.Read(initialMessageTimeout, ws, &helloMessage)
		if err != nil {
			return true, fmt.Errorf("did not receive gateway hello message: %w", err)
		}

		if helloMessage.Kind != connectproto.GatewayMessageType_GATEWAY_HELLO {
			return true, fmt.Errorf("expected gateway hello message, got %s", helloMessage.Kind)
		}

		h.h.Logger.Debug("received gateway hello message")
	}

	// Send connect message
	{
		hashedKey, err := hashedSigningKey([]byte(data.signingKey))
		if err != nil {
			return false, fmt.Errorf("could not hash signing key: %w", err)
		}

		apiOrigin := defaultAPIOrigin
		if h.h.isDev() {
			apiOrigin = DevServerURL()
		}

		data, err := proto.Marshal(&connectproto.WorkerConnectRequestData{
			SessionId: &connectproto.SessionIdentifier{
				BuildId:      h.h.BuildId,
				InstanceId:   h.instanceId(),
				ConnectionId: h.connectionId.String(),
			},
			AuthData: &connectproto.AuthData{
				HashedSigningKey: hashedKey,
			},
			AppName: h.h.appName,
			Config: &connectproto.ConfigDetails{
				Capabilities: data.marshaledCapabilities,
				Functions:    data.marshaledFns,
				ApiOrigin:    apiOrigin,
			},
			SystemAttributes: &connectproto.SystemAttributes{
				CpuCores: data.numCpuCores,
				MemBytes: data.totalMem,
				Os:       runtime.GOOS,
			},
			Environment: h.h.Env,
			Platform:    Ptr(platform()),
			SdkVersion:  SDKVersion,
			SdkLanguage: SDKLanguage,
		})
		if err != nil {
			return false, fmt.Errorf("could not serialize sdk connect message: %w", err)
		}

		err = wsproto.Write(ctx, ws, &connectproto.ConnectMessage{
			Kind:    connectproto.GatewayMessageType_WORKER_CONNECT,
			Payload: data,
		})
		if err != nil {
			return true, fmt.Errorf("could not send initial message")
		}
	}

	// TODO Read gateway ready

	// TODO Send buffered but unsent messages if connection was re-established

	inProgress := sync.WaitGroup{}

	var (
		closeErr     error
		closeErrLock sync.Mutex
	)

	readLoopCtx, cancelReadLoop := context.WithCancel(ctx)
	go func() {
		// Close connection if run loop ends
		defer cancelReadLoop()

		for {
			if readLoopCtx.Err() != nil {
				break
			}

			var msg connectproto.ConnectMessage
			err = wsproto.Read(readLoopCtx, ws, &msg)
			if err != nil {
				// connection lost with reason
				cerr := websocket.CloseError{}
				if errors.As(err, &cerr) {
					h.h.Logger.Error("connection closed unexpectedly", "reason", cerr.Reason)
					closeErrLock.Lock()
					closeErr = cerr
					closeErrLock.Unlock()
					// Reconnect!
					return
				}

				// connection lost without reason
				if errors.Is(err, io.EOF) {
					h.h.Logger.Error("failed to read message from gateway, lost connection unexpectedly", "err", err)
					return
				}

				h.h.Logger.Error("failed to read message", "err", err)

				// The connection may still be active, but for some reason we couldn't read the message
				return
			}

			h.h.Logger.Debug("received gateway request", "msg", &msg)

			switch msg.Kind {
			case connectproto.GatewayMessageType_GATEWAY_EXECUTOR_REQUEST:
				// TODO: this should be a pool instead of dynamic goroutines
				// Handle invoke in a non-blocking way to allow for other messages to be processed
				inProgress.Add(1)
				go func() {
					defer inProgress.Done()

					// Always make sure the invoke finishes properly
					processCtx := context.Background()

					err := h.handleInvokeMessage(processCtx, ws, &msg)

					// When we encounter an error, we cannot retry the connection from inside the goroutine.
					// If we're dealing with connection loss, the next read loop will fail with the same error
					// and handle the reconnection.
					if err != nil {
						cerr := websocket.CloseError{}
						if errors.As(err, &cerr) {
							h.h.Logger.Error("gateway connection closed with reason", "reason", cerr.Reason)
							return
						}

						if errors.Is(err, io.EOF) {
							h.h.Logger.Error("gateway connection closed unexpectedly", "err", err)
							return
						}
					}
				}()
			default:
				h.h.Logger.Error("got unknown gateway request", "err", err)
				continue
			}
		}
	}()

	<-readLoopCtx.Done()

	// In case the gateway intentionally closed the connection, we'll receive a close error
	if closeErr != nil {
		return true, fmt.Errorf("connection closed unexpectedly: %w", closeErr)
	}

	// If read loop ended, this could be for two reasons
	// - Connection loss (io.EOF), read loop terminated intentionally
	// - Worker shutdown, parent context got canceled
	if ctx.Err() == nil {
		return true, fmt.Errorf("connection closed unexpectedly")
	}

	// Perform graceful shutdown routine

	// TODO Signal gateway that we won't process additional messages!

	// Wait until all in-progress requests are completed
	inProgress.Wait()

	// TODO Send out buffered messages, using new connection if necessary!

	_ = ws.Close(websocket.StatusNormalClosure, connectproto.WorkerDisconnectReason_WORKER_SHUTDOWN.String())

	return false, nil
}

func (h *connectHandler) handleInvokeMessage(ctx context.Context, ws *websocket.Conn, msg *connectproto.ConnectMessage) error {
	resp, err := h.connectInvoke(ctx, msg)
	if err != nil {
		h.h.Logger.Error("failed to handle sdk request", "err", err)
		// TODO Should we drop the connection? Continue receiving messages?
		return fmt.Errorf("could not handle sdk request: %w", err)
	}

	data, err := proto.Marshal(resp)
	if err != nil {
		h.h.Logger.Error("failed to serialize sdk response", "err", err)
		// TODO This should never happen; Signal that we should retry
		return fmt.Errorf("could not serialize sdk response: %w", err)
	}

	responseMessage := &connectproto.ConnectMessage{
		Kind:    connectproto.GatewayMessageType_WORKER_REPLY,
		Payload: data,
	}

	err = wsproto.Write(ctx, ws, responseMessage)
	if err != nil {
		h.h.Logger.Error("failed to send sdk response", "err", err)

		// Buffer message to retry
		h.messageBufferLock.Lock()
		h.messageBuffer = append(h.messageBuffer, responseMessage)
		h.messageBufferLock.Unlock()

		return fmt.Errorf("could not send sdk response: %w", err)
	}

	return nil
}

// connectInvoke is the counterpart to invoke for connect
func (h *connectHandler) connectInvoke(ctx context.Context, msg *connectproto.ConnectMessage) (*connectproto.SDKResponse, error) {
	body := connectproto.GatewayExecutorRequestData{}
	if err := proto.Unmarshal(msg.Payload, &body); err != nil {
		h.h.Logger.Error("error decoding gateway request data", "error", err)
		return nil, fmt.Errorf("invalid gateway message data: %w", err)
	}

	// Note: This still uses JSON
	// TODO Replace with Protobuf
	var request sdkrequest.Request
	if err := json.Unmarshal(body.RequestPayload, &request); err != nil {
		h.h.Logger.Error("error decoding sdk request", "error", err)
		return nil, publicerr.Error{
			Message: "malformed input",
			Status:  400,
		}
	}

	if request.UseAPI {
		// TODO: implement this
		// retrieve data from API
		// request.Steps =
		// request.Events =
		_ = 0 // no-op to avoid linter error
	}

	h.h.l.RLock()
	var fn ServableFunction
	for _, f := range h.h.funcs {
		if f.Slug() == body.FunctionSlug {
			fn = f
			break
		}
	}
	h.h.l.RUnlock()

	if fn == nil {
		// XXX: This is a 500 within the JS SDK.  We should probably change
		// the JS SDK's status code to 410.  404 indicates that the overall
		// API for serving Inngest isn't found.
		return nil, publicerr.Error{
			Message: fmt.Sprintf("function not found: %s", body.FunctionSlug),
			Status:  410,
		}
	}

	var stepId *string
	if body.StepId != nil && *body.StepId != "step" {
		stepId = body.StepId
	}

	resp, ops, err := invoke(ctx, fn, &request, stepId)

	// NOTE: When triggering step errors, we should have an OpcodeStepError
	// within ops alongside an error.  We can safely ignore that error, as it's
	// only used for checking whether the step used a NoRetryError or RetryAtError
	//
	// For that reason, we check those values first.
	noRetry := sdkerrors.IsNoRetryError(err)
	retryAt := sdkerrors.GetRetryAtTime(err)
	if len(ops) == 1 && ops[0].Op == enums.OpcodeStepError {
		// Now we've handled error types we can ignore step
		// errors safely.
		err = nil
	}

	// Now that we've handled the OpcodeStepError, if we *still* ahve
	// a StepError kind returned from a function we must have an unhandled
	// step error.  This is a NonRetryableError, as the most likely code is:
	//
	// 	_, err := step.Run(ctx, func() (any, error) { return fmt.Errorf("") })
	// 	if err != nil {
	// 	     return err
	// 	}
	if sdkerrors.IsStepError(err) {
		err = fmt.Errorf("Unhandled step error: %s", err)
		noRetry = true
	}

	// These may be added even for 2xx codes with step errors.
	var retryAfterVal *string
	if retryAt != nil {
		retryAfterVal = StrPtr(retryAt.Format(time.RFC3339))
	}

	if err != nil {
		h.h.Logger.Error("error calling function", "error", err)
		return &connectproto.SDKResponse{
			RequestId:  body.RequestId,
			Status:     connectproto.SDKResponseStatus_ERROR,
			Body:       []byte(fmt.Sprintf("error calling function: %s", err.Error())),
			NoRetry:    noRetry,
			RetryAfter: retryAfterVal,
		}, nil
	}

	if len(ops) > 0 {
		// Note: This still uses JSON
		// TODO Replace with Protobuf
		serializedOps, err := json.Marshal(ops)
		if err != nil {
			return nil, fmt.Errorf("could not serialize ops: %w", err)
		}

		// Return the function opcode returned here so that we can re-invoke this
		// function and manage state appropriately.  Any opcode here takes precedence
		// over function return values as the function has not yet finished.
		return &connectproto.SDKResponse{
			RequestId:  body.RequestId,
			Status:     connectproto.SDKResponseStatus_NOT_COMPLETED,
			Body:       serializedOps,
			NoRetry:    noRetry,
			RetryAfter: retryAfterVal,
		}, nil
	}

	// Note: This still uses JSON
	// TODO Replace with Protobuf
	serializedResp, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("could not serialize resp: %w", err)
	}

	// Return the function response.
	return &connectproto.SDKResponse{
		RequestId:  body.RequestId,
		Status:     connectproto.SDKResponseStatus_DONE,
		Body:       serializedResp,
		NoRetry:    noRetry,
		RetryAfter: retryAfterVal,
	}, nil
}
