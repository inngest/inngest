package connect

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/inngest/inngest/pkg/syscode"
	connectproto "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/inngest/inngestgo/internal/fn"
	"github.com/inngest/inngestgo/internal/sdkrequest"
	"github.com/inngest/inngestgo/internal/types"
	"github.com/pbnjay/memory"
	"golang.org/x/sync/errgroup"
)

const (
	defaultMaxWorkerConcurrency = int64(0)
	defaultWorkerPoolSize       = 1_000
	maxWorkerPoolSize           = 100_000_000
	maxWorkerConcurrencyEnvKey  = "INNGEST_CONNECT_MAX_WORKER_CONCURRENCY"
)

type ConnectionState string

const (
	ConnectionStateConnecting   ConnectionState = "CONNECTING"
	ConnectionStateActive       ConnectionState = "ACTIVE"
	ConnectionStatePaused       ConnectionState = "PAUSED"
	ConnectionStateReconnecting ConnectionState = "RECONNECTING"
	ConnectionStateClosing      ConnectionState = "CLOSING"
	ConnectionStateClosed       ConnectionState = "CLOSED"
)

type WorkerConnection interface {
	// State returns the current connection state.
	State() ConnectionState

	// Close connection gracefully.
	Close() error
}

// Connect
func Connect(ctx context.Context, opts Opts, invokers map[string]FunctionInvoker, logger *slog.Logger) (WorkerConnection, error) {
	apiClient := newWorkerApiClient(opts.APIBaseURL, opts.Env)

	// Once the worker is running, it can only be stopped by calling Close()
	doneCtx, cancelDone := context.WithCancel(context.Background())

	l := logger.With("mode", "connect")

	ch := &connectHandler{
		logger:                 l,
		invokers:               invokers,
		opts:                   opts,
		notifyConnectDoneChan:  make(chan connectReport),
		notifyConnectedChan:    make(chan struct{}),
		initiateConnectionChan: make(chan struct{}),
		apiClient:              apiClient,
		messageBuffer:          newMessageBuffer(apiClient, logger),
		state:                  ConnectionStateConnecting,

		workerCtx:       doneCtx,
		cancelWorkerCtx: cancelDone,
	}

	// define a worker pool size based on the max worker concurrency
	workerPoolSize := defaultWorkerPoolSize
	if opts.MaxWorkerConcurrency != nil && *opts.MaxWorkerConcurrency > 0 {
		workerPoolSize = min(int(*opts.MaxWorkerConcurrency), maxWorkerPoolSize)
	}

	wp := NewWorkerPool(ctx, workerPoolSize, ch.processExecutorRequest)
	ch.workerPool = wp

	conn, err := ch.Connect(ctx)
	if err != nil {
		l.Error("could not establish connection", "error", err)
		return nil, fmt.Errorf("could not establish connection: %w", err)
	}

	return conn, nil
}

type FunctionInvoker interface {
	InvokeFunction(ctx context.Context, slug string, stepId *string, request sdkrequest.Request) (any, []sdkrequest.GeneratorOpcode, error)
}

type ConnectApp struct {
	AppName    string
	Functions  []fn.SyncConfig
	AppVersion *string
}

// Opts represents required options for creating a persistent connection for communicating
// with an Inngest server.
type Opts struct {
	Env  *string
	Apps []ConnectApp

	Capabilities types.Capabilities

	HashedSigningKey         []byte
	HashedSigningKeyFallback []byte

	MaxWorkerConcurrency *int64

	// MessageReadLimit sets the max number of bytes to read for a single WebSocket message.
	// By default (nil or 0), the connection has a message read limit of 10MB.
	// Set to -1 to disable the limit.
	// Set to any positive value to use a custom limit.
	MessageReadLimit *int64

	APIBaseURL string
	IsDev      bool

	InstanceID *string

	Platform    *string
	SDKVersion  string
	SDKLanguage string

	RewriteGatewayEndpoint func(endpoint url.URL) (url.URL, error)
}

type connectHandler struct {
	opts Opts

	invokers map[string]FunctionInvoker

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

	// Global connection state

	stateLock sync.RWMutex
	state     ConnectionState

	workerCtx       context.Context
	cancelWorkerCtx context.CancelFunc
	gracefulCloseEg errgroup.Group
	auth            authContext
	closed          atomic.Bool
}

// authContext is wrapper for information related to authentication
type authContext struct {
	hashedSigningKey []byte
	fallback         bool
}

func (h *connectHandler) Connect(ctx context.Context) (WorkerConnection, error) {
	// While the worker is starting, it can be canceled using the passed context
	startCtx, cancelStart := context.WithTimeout(ctx, time.Second*30)
	defer cancelStart()

	l := h.logger

	signingKey := h.opts.HashedSigningKey
	if len(signingKey) == 0 && !h.opts.IsDev {
		return nil, fmt.Errorf("hashed signing key is required")
	}

	h.auth = authContext{hashedSigningKey: signingKey}

	numCpuCores := runtime.NumCPU()
	totalMem := memory.TotalMemory()

	apps := make([]*connectproto.AppConfiguration, len(h.opts.Apps))
	appSlugs := make(map[string]map[string][]string)
	for i, app := range h.opts.Apps {
		functionSlugs := make(map[string][]string)
		for _, function := range app.Functions {
			stepUrls := make([]string, len(function.Steps))
			j := 0
			for _, step := range function.Steps {
				stepUrls[j] = step.Runtime["url"].(string)
				j++
			}

			functionSlugs[function.Slug] = stepUrls
		}
		appSlugs[app.AppName] = functionSlugs

		marshaledFns, err := json.Marshal(app.Functions)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize connect config: %w", err)
		}

		apps[i] = &connectproto.AppConfiguration{
			AppName:    app.AppName,
			AppVersion: app.AppVersion,
			Functions:  marshaledFns,
		}
	}

	l.Debug("using provided functions", "slugs", appSlugs)

	marshaledCapabilities, err := json.Marshal(h.opts.Capabilities)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize connect config: %w", err)
	}

	var attempts int

	isInitialConnection := true
	initialConnectionDone := make(chan error)

	// We construct a connection loop, which will attempt to reconnect on failure
	// Instead of doing a simple, synchronous loop, we use channels to communicate connection status changes,
	// allowing to instantiate a new connection while the previous one is still running.
	// This is crucial for handling gateway draining scenarios.
	runLoop := errgroup.Group{}
	runLoop.Go(func() error {
		for {
			select {
			// If the context is canceled, we should not attempt to reconnect
			case <-h.workerCtx.Done():
				return nil

			// Reset attempts when connection succeeded
			case <-h.notifyConnectedChan:
				l.Debug("connected")
				if isInitialConnection {
					isInitialConnection = false
					initialConnectionDone <- nil
				}
				h.setState(ConnectionStateActive, "connection established", "attempt", attempts)
				attempts = 0
				continue

			// Handle connection done events
			case msg := <-h.notifyConnectDoneChan:
				if msg.err != nil {
					l.Error("connection ended with error", "err", msg.err, "reconnect", msg.reconnect)
				} else {
					l.Debug("connection ended", "reconnect", msg.reconnect)
				}

				if !msg.reconnect {
					h.setState(ConnectionStateClosed, "connection ended without reconnect", "err", msg.err)

					if isInitialConnection {
						isInitialConnection = false
						initialConnectionDone <- err
					}

					return err
				}

				h.setState(ConnectionStateReconnecting, "connection ended with reconnect", "err", msg.err, "attempt", attempts)

				// Some errors should be handled differently (e.g. auth failed)
				if msg.err != nil {
					if errors.Is(msg.err, ErrTooManyConnections) {
						// If limits are exceed in initial connection, return immediately
						if isInitialConnection {
							isInitialConnection = false
							initialConnectionDone <- fmt.Errorf("too many connections, please disconnect other workers or upgrade your billing plan for more concurrent connections")
							return err
						}
					}

					if errors.Is(msg.err, ErrUnauthenticated) {
						if h.auth.fallback {
							err := fmt.Errorf("failed to authenticate with fallback key, exiting")
							if isInitialConnection {
								isInitialConnection = false
								initialConnectionDone <- err
							}

							return err
						}

						signingKeyFallback := h.opts.HashedSigningKeyFallback

						if len(signingKeyFallback) == 0 {
							err := fmt.Errorf("fallback signing key is required")

							if isInitialConnection {
								isInitialConnection = false
								initialConnectionDone <- err
							}

							return err
						}

						h.auth = authContext{hashedSigningKey: signingKeyFallback, fallback: true}
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
							err := fmt.Errorf("connect failed with error code %q", closeErr.Reason)

							if isInitialConnection {
								isInitialConnection = false
								initialConnectionDone <- err
							}

							// If we received a reason that's non-retriable, stop here.
							return fmt.Errorf("connect failed with error code %q", closeErr.Reason)
						}
					}
				}

				// Attempt to flush messages before reconnecting
				if h.messageBuffer.hasMessages() {
					err := h.messageBuffer.flush(h.auth.hashedSigningKey)
					if err != nil {
						l.Error("could not send buffered messages", "err", err)
					}
				}

				// continue to reconnect logic
				delay := expBackoff(attempts)

				l.Debug("reconnecting", "delay", delay.String(), "attempts", attempts)

				select {
				case <-time.After(delay):
					break
				case <-h.workerCtx.Done():
					if isInitialConnection {
						isInitialConnection = false
						initialConnectionDone <- nil
					}

					l.Info("canceled context while waiting to reconnect")
					return nil
				}

			case <-h.initiateConnectionChan:
			}

			if attempts == 5 {
				err := fmt.Errorf("could not establish connection after 5 attempts")

				if isInitialConnection {
					isInitialConnection = false
					initialConnectionDone <- err
				}

				return err
			}

			attempts++

			connectCtx := ctx
			if isInitialConnection {
				connectCtx = startCtx
			}

			go h.connect(connectCtx, connectionEstablishData{
				hashedSigningKey:      h.auth.hashedSigningKey,
				numCpuCores:           int32(numCpuCores),
				totalMem:              int64(totalMem),
				marshaledCapabilities: marshaledCapabilities,
				apps:                  apps,
			})
		}
	})

	// Handle run loop closure gracefully, this is also triggered on Close()
	h.gracefulCloseEg = errgroup.Group{}
	h.gracefulCloseEg.Go(func() error {
		// Wait for run loop to complete (maximum attempts reached, context canceled)
		runLoopErr := runLoop.Wait()
		if runLoopErr != nil {
			l.Error("could not connect", "err", runLoopErr)
		}

		l.Debug("run loop ended")

		// Wait until current connection is fully terminated
		select {
		case <-ctx.Done():
		case <-time.After(5 * time.Second):
			l.Warn("shutting down without final signal")
		case <-h.notifyConnectDoneChan:
			l.Debug("got connection done signal")
		}

		// Always send out buffered messages using API
		if h.messageBuffer.hasMessages() {
			// Send buffered messages
			err := h.messageBuffer.flush(h.auth.hashedSigningKey)
			if err != nil {
				l.Error("could not send buffered messages", "err", err)
			}

			// TODO Push remaining messages to another destination for processing?
		}

		l.Debug("connect handler done")
		return nil
	})

	// Initiate the first connection
	h.initiateConnectionChan <- struct{}{}

	// Wait until connected (or context is closed)
	select {
	case <-startCtx.Done():
		_ = h.Close()

		return nil, fmt.Errorf("context canceled while establishing connection")
	case err := <-initialConnectionDone:
		if err != nil {
			return nil, fmt.Errorf("could not establish connection: %w", err)
		}
	}

	return h, nil
}

func (h *connectHandler) Close() error {
	// If connection was already closed, this is a no-op.
	if h.closed.Swap(true) {
		h.logger.Debug("close ignored; connection already closed", "state", h.State())
		return nil
	}

	if h.cancelWorkerCtx == nil {
		err := fmt.Errorf("connection was not fully set up")
		h.logger.Error("could not close connection", "err", err, "state", h.State())
		return err
	}

	h.setState(ConnectionStateClosing, "close requested")
	h.cancelWorkerCtx()

	// Wait until connection loop finishes
	err := h.gracefulCloseEg.Wait()
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	h.setState(ConnectionStateClosed, "close complete")

	return nil
}

func (h *connectHandler) State() ConnectionState {
	h.stateLock.RLock()
	defer h.stateLock.RUnlock()
	return h.state
}

func (h *connectHandler) setState(state ConnectionState, reason string, attrs ...any) {
	h.stateLock.Lock()
	previous := h.state
	h.state = state
	h.stateLock.Unlock()

	logAttrs := []any{
		"from", previous,
		"to", state,
		"reason", reason,
	}
	logAttrs = append(logAttrs, attrs...)
	h.logger.Debug("worker connection state transition", logAttrs...)
}

var errGatewayDraining = errors.New("gateway is draining")

func (h *connectHandler) processExecutorRequest(msg workerPoolMsg) {
	defer h.workerPool.Done()

	// Always make sure the invoke finishes properly
	processCtx := context.Background()

	err := h.handleInvokeMessage(processCtx, msg.preparedConn, msg.msg)
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
	if h.opts.InstanceID != nil {
		return *h.opts.InstanceID
	}

	hostname, _ := os.Hostname()
	if hostname != "" {
		return hostname
	}

	// TODO Is there any stable identifier that can be used as a fallback?
	return "<missing-instance-id>"
}

// maxWorkerConcurrency returns the maximum number of worker concurrency to use.
func (h *connectHandler) maxWorkerConcurrency() *int64 {
	// user provided max worker concurrency
	if h.opts.MaxWorkerConcurrency != nil {
		return h.opts.MaxWorkerConcurrency
	}

	// environment variable
	envValue := os.Getenv(maxWorkerConcurrencyEnvKey)
	if envValue != "" {
		if concurrency, err := strconv.ParseInt(envValue, 10, 64); err == nil {
			return &concurrency
		}
		// ignore error
	}

	// default max worker concurrency
	concurrency := defaultMaxWorkerConcurrency
	return &concurrency
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
