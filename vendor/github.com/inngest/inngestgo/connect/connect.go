package connect

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/inngest/inngestgo/internal/connectworker"
	"github.com/inngest/inngestgo/internal/fn"
	"github.com/inngest/inngestgo/internal/sdkrequest"
	"github.com/inngest/inngestgo/internal/types"
)

// ConnectionState is the state of the worker connection that State returns.
type ConnectionState = connectworker.ConnectionState

const (
	ConnectionStateConnecting = connectworker.ConnectionStateConnecting
	ConnectionStateActive     = connectworker.ConnectionStateActive
	// ConnectionStatePaused is retained for source compatibility. The manager
	// no longer enters PAUSED; local shutdown is CLOSING and websocket
	// generation pause writes are modeled separately.
	ConnectionStatePaused       = connectworker.ConnectionStatePaused
	ConnectionStateReconnecting = connectworker.ConnectionStateReconnecting
	ConnectionStateClosing      = connectworker.ConnectionStateClosing
	ConnectionStateClosed       = connectworker.ConnectionStateClosed
)

// WorkerConnection is the worker connection that Connect returns.  it stays
// open, and reconnects after a lost gateway connection, until Close.
type WorkerConnection = connectworker.WorkerConnection

// ResponseAcknowlegeDeadline is how long the worker waits for the gateway to
// ACK a reply.  a reply with no ACK by then goes to the buffer, and the worker
// sends it to the API with POST /v0/connect/flush.
const ResponseAcknowlegeDeadline = connectworker.ResponseAcknowlegeDeadline

var (
	// ErrUnauthenticated means /v0/connect/start rejected the signing key.
	// the worker then retries with HashedSigningKeyFallback.  without a
	// fallback key, or when the fallback key fails too, the worker stops.
	ErrUnauthenticated = connectworker.ErrUnauthenticated
	// ErrTooManyConnections means the account has no connections left.  on
	// the first connection, Connect returns it at once.
	ErrTooManyConnections = connectworker.ErrTooManyConnections

	// NewWorkerPool starts the goroutines that serve executor requests.  its
	// handler argument takes an unexported type, so code outside this module
	// cannot build one.
	NewWorkerPool = connectworker.NewWorkerPool
)

// FunctionInvoker runs one executor request for a function.  slug is the
// function's fully qualified ID, and stepId is nil when the request targets
// the function's only step.  the returned value is the function's output.  the
// opcodes report the steps that this call reached, and mean the run is not
// done.  an error sends an ERROR reply, which the executor retries unless the
// error is a NoRetryError.
type FunctionInvoker interface {
	InvokeFunction(ctx context.Context, slug string, stepId *string, request sdkrequest.Request) (any, []sdkrequest.GeneratorOpcode, error)
}

// ConnectApp is one app that the worker serves.
type ConnectApp struct {
	// AppName is the app's ID.  requests for this app run on the
	// FunctionInvoker that Connect has under the same name.
	AppName string
	// Functions are the function configs that the gateway syncs for this
	// app.
	Functions []fn.SyncConfig
	// AppVersion is optional.  it goes into the handshake with the app.
	AppVersion *string
}

// Opts represents required options for creating a persistent connection for communicating
// with an Inngest server.
type Opts struct {
	// Env is the Inngest environment name.  the worker sends it in the
	// X-Inngest-Env header and in the handshake.
	Env  *string
	Apps []ConnectApp

	// Capabilities go into the handshake as JSON.
	Capabilities types.Capabilities

	// HashedSigningKey authenticates /v0/connect/start, reply flushes, and
	// state loads from the API.  Connect fails without it unless IsDev is
	// set.  HashedSigningKeyFallback is used after the API rejects
	// HashedSigningKey.
	HashedSigningKey         []byte
	HashedSigningKeyFallback []byte

	// MaxWorkerConcurrency caps how many requests the worker runs at once.
	// the gateway routes no more than this many to workers that share an
	// InstanceID.  when nil, the INNGEST_CONNECT_MAX_WORKER_CONCURRENCY env
	// var sets it.  0 means no limit.
	MaxWorkerConcurrency *int64

	// MessageReadLimit sets the max number of bytes to read for a single WebSocket message.
	// By default (nil or 0), the connection has a message read limit of 10MB.
	// Set to -1 to disable the limit.
	// Set to any positive value to use a custom limit.
	MessageReadLimit *int64

	// MissedGatewayHeartbeatTolerance is the number of gateway heartbeat periods
	// that may pass without a gateway heartbeat before reconnecting.
	// Defaults to 3. Negative values use the default.
	MissedGatewayHeartbeatTolerance *int

	// APIBaseURL is the Inngest API that the worker calls to start a
	// connection, flush replies, and load function state.
	APIBaseURL string
	// IsDev lets Connect run without a signing key, which the dev server
	// does not need.
	IsDev bool

	// InstanceID is a stable ID for this worker, such as a hostname.  when
	// nil, the worker uses the hostname.
	InstanceID *string

	// Platform, SDKVersion, and SDKLanguage go into the handshake.  error
	// replies also carry SDKLanguage and SDKVersion.
	Platform    *string
	SDKVersion  string
	SDKLanguage string

	// RewriteGatewayEndpoint changes the gateway URL that /v0/connect/start
	// returns, before the worker dials it.  an error stops that connection
	// attempt, and the worker retries.
	RewriteGatewayEndpoint func(endpoint url.URL) (url.URL, error)
}

// Connect opens a worker connection to the gateway for the given apps.  each
// request runs on the FunctionInvoker registered under the request's app
// name.
func Connect(ctx context.Context, opts Opts, invokers map[string]FunctionInvoker, logger *slog.Logger) (WorkerConnection, error) {
	l := logger.With("mode", "connect")

	apps, appSlugs, err := appConfigurations(opts.Apps)
	if err != nil {
		l.Error("could not establish connection", "error", err)
		return nil, fmt.Errorf("could not establish connection: %w", err)
	}
	l.Debug("using provided functions", "slugs", appSlugs)

	return connectworker.Connect(ctx, connectworker.Opts{
		Env:                             opts.Env,
		Apps:                            apps,
		Capabilities:                    opts.Capabilities,
		HashedSigningKey:                opts.HashedSigningKey,
		HashedSigningKeyFallback:        opts.HashedSigningKeyFallback,
		MaxWorkerConcurrency:            opts.MaxWorkerConcurrency,
		MessageReadLimit:                opts.MessageReadLimit,
		MissedGatewayHeartbeatTolerance: opts.MissedGatewayHeartbeatTolerance,
		APIBaseURL:                      opts.APIBaseURL,
		IsDev:                           opts.IsDev,
		InstanceID:                      opts.InstanceID,
		Platform:                        opts.Platform,
		SDKVersion:                      opts.SDKVersion,
		SDKLanguage:                     opts.SDKLanguage,
		RewriteGatewayEndpoint:          opts.RewriteGatewayEndpoint,
		Handler: invokerHandler{
			invokers:   invokers,
			opts:       opts,
			logger:     l,
			httpClient: &http.Client{},
		},
	}, logger)
}
