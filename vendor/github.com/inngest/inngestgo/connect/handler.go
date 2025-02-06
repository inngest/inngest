package connect

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/coder/websocket"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/sdk"
	"github.com/inngest/inngest/pkg/syscode"
	"github.com/inngest/inngestgo/internal/sdkrequest"
	"github.com/pbnjay/memory"
	"golang.org/x/sync/errgroup"
	"io"
	"log/slog"
	"net/url"
	"os"
	"runtime"
	"time"
)

const (
	WorkerHeartbeatInterval = 10 * time.Second
)

func Connect(ctx context.Context, opts Opts, invoker FunctionInvoker, logger *slog.Logger) error {
	apiClient := newWorkerApiClient(opts.APIBaseUrl, opts.Env)
	ch := &connectHandler{
		logger:                 logger,
		invoker:                invoker,
		opts:                   opts,
		notifyConnectDoneChan:  make(chan connectReport),
		notifyConnectedChan:    make(chan struct{}),
		initiateConnectionChan: make(chan struct{}),
		apiClient:              apiClient,
		messageBuffer:          newMessageBuffer(apiClient, logger),
	}

	wp := NewWorkerPool(ctx, opts.MaxConcurrency, ch.processExecutorRequest)
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

	MaxConcurrency int

	APIBaseUrl   string
	IsDev        bool
	DevServerUrl string

	BuildId    *string
	InstanceId *string

	Platform    *string
	SDKVersion  string
	SDKLanguage string

	RewriteGatewayEndpoint func(endpoint url.URL) (url.URL, error)
}

type connectHandler struct {
	opts Opts

	invoker FunctionInvoker

	logger *slog.Logger

	messageBuffer *messageBuffer

	workerPool *workerPool

	// Notify when connect finishes (either with an error or because the context got canceled)
	notifyConnectDoneChan chan connectReport

	// Notify when connection is established
	notifyConnectedChan chan struct{}

	// Channel to imperatively initiate a connection
	initiateConnectionChan chan struct{}

	apiClient *workerApiClient
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

	functionSlugs := make(map[string][]string)
	for _, function := range h.opts.Functions {
		stepUrls := make([]string, len(function.Steps))
		j := 0
		for _, step := range function.Steps {
			stepUrls[j] = step.Runtime["url"].(string)
			j++
		}

		functionSlugs[function.Slug] = stepUrls
	}

	h.logger.Debug("using provided functions", "slugs", functionSlugs)

	marshaledFns, err := json.Marshal(h.opts.Functions)
	if err != nil {
		return fmt.Errorf("failed to serialize connect config: %w", err)
	}

	marshaledCapabilities, err := json.Marshal(h.opts.Capabilities)
	if err != nil {
		return fmt.Errorf("failed to serialize connect config: %w", err)
	}

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
					if errors.Is(msg.err, ErrUnauthenticated) {
						if auth.fallback {
							return fmt.Errorf("failed to authenticate with fallback key, exiting")
						}

						signingKeyFallback := h.opts.HashedSigningKeyFallback
						if len(signingKeyFallback) == 0 {
							return fmt.Errorf("fallback signing key is required")
						}

						auth = authContext{hashedSigningKey: signingKeyFallback, fallback: true}
					}

					closeErr := websocket.CloseError{}
					if errors.As(msg.err, &closeErr) {
						switch closeErr.Reason {
						// If auth failed, retry with fallback key
						case syscode.CodeConnectAuthFailed:
							// already handled above

						// Retry on the following error codes
						case syscode.CodeConnectGatewayClosing, syscode.CodeConnectInternal, syscode.CodeConnectWorkerHelloTimeout:
							// continue to reconnect logic
						default:
							// If we received a reason that's non-retriable, stop here.
							return fmt.Errorf("connect failed with error code %q", closeErr.Reason)
						}
					}
				}

				// Attempt to flush messages before reconnecting
				if h.messageBuffer.hasMessages() {
					err := h.messageBuffer.flush(auth.hashedSigningKey)
					if err != nil {
						h.logger.Error("could not send buffered messages", "err", err)
					}
				}

				// continue to reconnect logic
				delay := expBackoff(attempts)

				h.logger.Debug("reconnecting", "delay", delay.String(), "attempts", attempts)

				select {
				case <-time.After(delay):
					break
				case <-ctx.Done():
					h.logger.Info("canceled context while waiting to reconnect")
					return nil
				}

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
	egError := eg.Wait()

	// Always send out buffered messages using API
	if h.messageBuffer.hasMessages() {
		// Send buffered messages
		err := h.messageBuffer.flush(auth.hashedSigningKey)
		if err != nil {
			h.logger.Error("could not send buffered messages", "err", err)
		}

		// TODO Push remaining messages to another destination for processing?
	}

	if egError != nil {
		return fmt.Errorf("could not establish connection: %w", err)
	}

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

func expBackoff(attempt int) time.Duration {
	backoffTimes := []time.Duration{
		time.Second, 2 * time.Second, 5 * time.Second, 10 * time.Second,
		20 * time.Second, 30 * time.Second, time.Minute, 2 * time.Minute, 5 * time.Minute,
	}

	if attempt >= len(backoffTimes) {
		return backoffTimes[len(backoffTimes)-1]
	}
	return backoffTimes[attempt]
}
