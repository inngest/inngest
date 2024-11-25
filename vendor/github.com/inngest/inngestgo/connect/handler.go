package connect

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/coder/websocket"
	"github.com/inngest/inngest/pkg/connect/wsproto"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/sdk"
	"github.com/inngest/inngest/pkg/syscode"
	connectproto "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/inngest/inngestgo/internal/sdkrequest"
	"github.com/pbnjay/memory"
	"golang.org/x/sync/errgroup"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	WorkerHeartbeatInterval = 10 * time.Second
)

func Connect(ctx context.Context, opts Opts, invoker FunctionInvoker, logger *slog.Logger) error {
	ch := &connectHandler{
		logger:                 logger,
		invoker:                invoker,
		opts:                   opts,
		notifyConnectDoneChan:  make(chan connectReport),
		notifyConnectedChan:    make(chan struct{}),
		initiateConnectionChan: make(chan struct{}),
	}

	wp := NewWorkerPool(ctx, opts.WorkerConcurrency, ch.processExecutorRequest)
	ch.workerPool = wp

	defer func() {
		// TODO Push remaining messages to another destination for processing?
	}()

	err := ch.Connect(ctx)
	if err != nil {
		return fmt.Errorf("could not establish connection: %w", err)
	}

	return nil
}

type FunctionInvoker interface {
	InvokeFunction(ctx context.Context, slug string, stepId *string, request sdkrequest.Request) (any, []state.GeneratorOpcode, error)
}

type Opts struct {
	AppName string
	Env     *string

	Functions    []sdk.SDKFunction
	Capabilities sdk.Capabilities

	HashedSigningKey         []byte
	HashedSigningKeyFallback []byte

	WorkerConcurrency int

	APIBaseUrl   string
	IsDev        bool
	DevServerUrl string
	ConnectUrls  []string

	InstanceId *string
	BuildId    *string

	Platform    *string
	SDKVersion  string
	SDKLanguage string
}

type connectHandler struct {
	opts Opts

	invoker FunctionInvoker

	logger *slog.Logger

	messageBuffer     []*connectproto.ConnectMessage
	messageBufferLock sync.Mutex

	hostsManager *hostsManager

	workerPool *workerPool

	// Notify when connect finishes (either with an error or because the context got canceled)
	notifyConnectDoneChan chan connectReport

	// Notify when connection is established
	notifyConnectedChan chan struct{}

	// Channel to imperatively initiate a connection
	initiateConnectionChan chan struct{}
}

// authContext is wrapper for information related to authentication
type authContext struct {
	hashedSigningKey []byte
	fallback         bool
}

func (h *connectHandler) Connect(ctx context.Context) error {
	signingKey := h.opts.HashedSigningKey
	if len(signingKey) == 0 {
		return fmt.Errorf("hashed signing key is required")
	}

	auth := authContext{hashedSigningKey: signingKey}

	numCpuCores := runtime.NumCPU()
	totalMem := memory.TotalMemory()

	marshaledFns, err := json.Marshal(h.opts.Functions)
	if err != nil {
		return fmt.Errorf("failed to serialize connect config: %w", err)
	}

	marshaledCapabilities, err := json.Marshal(h.opts.Capabilities)
	if err != nil {
		return fmt.Errorf("failed to serialize connect config: %w", err)
	}

	hosts := h.connectURLs()
	if len(hosts) == 0 {
		return fmt.Errorf("no connect URLs provided")
	}

	h.hostsManager = newHostsManager(hosts)

	var attempts int

	// We construct a connection loop, which will attempt to reconnect on failure
	// Instead of doing a simple, synchronous loop, we use channels to communicate connection status changes,
	// allowing to instantiate a new connection while the previous one is still running.
	// This is crucial for handling gateway draining scenarios.
	eg := errgroup.Group{}
	eg.Go(func() error {
		for {
			select {
			// If the context is canceled, we should not attempt to reconnect
			case <-ctx.Done():
				return nil

			// Reset attempts when connection succeeded
			case <-h.notifyConnectedChan:
				attempts = 0
				continue

			// Handle connection done events
			case msg := <-h.notifyConnectDoneChan:
				h.logger.Error("connect failed", "err", err, "reconnect", msg.reconnect)

				if !msg.reconnect {
					return err
				}

				// Some errors should be handled differently (e.g. auth failed)
				if msg.err != nil {
					closeErr := websocket.CloseError{}
					if errors.As(err, &closeErr) {
						switch closeErr.Reason {
						// If auth failed, retry with fallback key
						case syscode.CodeConnectAuthFailed:
							if auth.fallback {
								return fmt.Errorf("failed to authenticate with fallback key, exiting")
							}

							signingKeyFallback := h.opts.HashedSigningKeyFallback
							if len(signingKeyFallback) == 0 {
								return fmt.Errorf("fallback signing key is required")
							}

							auth = authContext{hashedSigningKey: signingKeyFallback, fallback: true}

							// continue to reconnect logic

						// Retry on the following error codes
						case syscode.CodeConnectGatewayClosing, syscode.CodeConnectInternal, syscode.CodeConnectWorkerHelloTimeout:
							// continue to reconnect logic
						default:
							// If we received a reason  that's non-retriable, stop here.
							return fmt.Errorf("connect failed with error code %q", closeErr.Reason)
						}
					}
				}

				// continue to reconnect logic
				h.logger.Debug("reconnecting", "attempts", attempts)

			case <-h.initiateConnectionChan:
			}

			if attempts == 5 {
				return fmt.Errorf("could not establish connection after 5 attempts")
			}

			attempts++

			go h.connect(ctx, connectionEstablishData{
				hashedSigningKey:      auth.hashedSigningKey,
				numCpuCores:           int32(numCpuCores),
				totalMem:              int64(totalMem),
				marshaledFns:          marshaledFns,
				marshaledCapabilities: marshaledCapabilities,
			})
		}
	})

	// Initiate the first connection
	h.initiateConnectionChan <- struct{}{}

	// Wait until connection loop finishes
	if err := eg.Wait(); err != nil {
		return fmt.Errorf("could not establish connection: %w", err)
	}

	// Send out buffered messages, using new connection if necessary!
	h.messageBufferLock.Lock()
	defer h.messageBufferLock.Unlock()
	if len(h.messageBuffer) > 0 {
		//  Send buffered messages via a working connection
		err = h.withTemporaryConnection(connectionEstablishData{
			hashedSigningKey:      auth.hashedSigningKey,
			numCpuCores:           int32(numCpuCores),
			totalMem:              int64(totalMem),
			marshaledFns:          marshaledFns,
			marshaledCapabilities: marshaledCapabilities,
		}, func(ws *websocket.Conn) error {
			// Send buffered messages
			err := h.sendBufferedMessages(ws)
			if err != nil {
				return fmt.Errorf("could not send buffered messages: %w", err)
			}

			return nil
		})
		if err != nil {
			h.logger.Error("could not establish connection for sending buffered messages", "err", err)
		}

		// TODO Push remaining messages to another destination for processing?
	}

	return nil
}

func (h *connectHandler) sendBufferedMessages(ws *websocket.Conn) error {
	processed := 0
	for _, msg := range h.messageBuffer {
		// always send the message, even if the context is cancelled
		err := wsproto.Write(context.Background(), ws, msg)
		if err != nil {
			// Only send buffered messages once
			h.messageBuffer = h.messageBuffer[processed:]

			h.logger.Error("failed to send buffered message", "err", err)
			return fmt.Errorf("could not send buffered message: %w", err)
		}

		h.logger.Debug("sent buffered message", "msg", msg)
		processed++
	}
	h.messageBuffer = nil
	return nil
}

var errGatewayDraining = errors.New("gateway is draining")

func (h *connectHandler) processExecutorRequest(msg workerPoolMsg) {
	defer h.workerPool.Done()

	// Always make sure the invoke finishes properly
	processCtx := context.Background()

	err := h.handleInvokeMessage(processCtx, msg.ws, msg.msg)

	// When we encounter an error, we cannot retry the connection from inside the goroutine.
	// If we're dealing with connection loss, the next read loop will fail with the same error
	// and handle the reconnection.
	if err != nil {
		cerr := websocket.CloseError{}
		if errors.As(err, &cerr) {
			h.logger.Error("gateway connection closed with reason", "reason", cerr.Reason)
			return
		}

		if errors.Is(err, io.EOF) {
			h.logger.Error("gateway connection closed unexpectedly", "err", err)
			return
		}

		// TODO If error is not connection-related, should we retry? Send the buffered message?
	}
}

func (h *connectHandler) connectURLs() []string {
	if len(h.opts.ConnectUrls) > 0 {
		return h.opts.ConnectUrls
	}

	if h.opts.IsDev {
		return []string{fmt.Sprintf("%s/connect", strings.Replace(h.opts.DevServerUrl, "http", "ws", 1))}
	}

	return nil
}

func (h *connectHandler) instanceId() string {
	if h.opts.InstanceId != nil {
		return *h.opts.InstanceId
	}

	hostname, _ := os.Hostname()
	if hostname != "" {
		return hostname
	}

	// TODO Is there any stable identifier that can be used as a fallback?
	return "<missing-instance-id>"
}
