package connectv0

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/oklog/ulid/v2"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"io"
	"net/http"

	"github.com/inngest/inngest/pkg/connect/auth"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/inngest/inngest/proto/gen/connect/v1"
	"google.golang.org/protobuf/proto"
)

func (cr *connectApiRouter) start(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	connectionId, err := ulid.New(ulid.Now(), rand.Reader)
	if err != nil {
		logger.StdlibLogger(ctx).Error("could not generate connection id", "err", err)

		_ = publicerr.WriteHTTP(w, publicerr.Errorf(500, "internal error"))
		return
	}

	l := logger.StdlibLogger(ctx).With("conn_id", connectionId)

	hashedSigningKey := r.Header.Get("Authorization")
	{
		if hashedSigningKey == "" && !cr.Dev {
			_ = publicerr.WriteHTTP(w, publicerr.Errorf(401, "missing Authorization header"))
			return
		}

		if hashedSigningKey != "" && len(hashedSigningKey) > 7 {
			// Remove "Bearer " prefix
			hashedSigningKey = hashedSigningKey[7:]
		}
	}

	envOverride := r.Header.Get("X-Inngest-Env")

	l = l.With("key", hashedSigningKey, "env", envOverride)

	res, err := cr.RequestAuther.AuthenticateRequest(ctx, hashedSigningKey, envOverride)
	if err != nil {
		l.Error("could not authenticate connect start request", "err", err)

		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 401, "authentication failed"))
		return
	}

	if res == nil {
		l.Debug("rejecting unauthorized connection")

		_ = publicerr.WriteHTTP(w, publicerr.Errorf(401, "authentication failed"))
		return
	}

	entitlements, err := cr.EntitlementProvider.RetrieveConnectEntitlements(ctx, res)
	if err != nil {
		l.Error("could not check connection limit during start request", "err", err)

		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "could not check connection limit"))
		return
	}

	if !entitlements.ConnectionAllowed {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 429, "reached max allowed connections"))
		return
	}

	byt, err := io.ReadAll(io.LimitReader(r.Body, 1024*1024))
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, "could not read request body"))
		return
	}

	reqBody := &connect.StartRequest{}
	if len(byt) > 0 {
		if err := proto.Unmarshal(byt, reqBody); err != nil {
			_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, "could not unmarshal request"))
			return
		}
	}

	token, err := cr.Signer.SignSessionToken(res.AccountID, res.EnvID, auth.DefaultExpiry, entitlements)
	if err != nil {
		l.Error("could not sign connect session token", "err", err)

		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "could not sign session token"))
		return
	}

	gatewayGroup, gatewayUrl, err := cr.ConnectGatewayRetriever.RetrieveGateway(ctx, RetrieveGatewayOpts{
		AccountID:   res.AccountID,
		EnvID:       res.EnvID,
		Exclude:     reqBody.ExcludeGateways,
		RequestHost: r.Host,
	})
	if err != nil {
		l.Error("could not retrieve connect gateway", "err", err)

		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "could not retrieve gateway"))
		return
	}

	msg, err := proto.Marshal(&connect.StartResponse{
		GatewayEndpoint: gatewayUrl.String(),
		GatewayGroup:    gatewayGroup,
		SessionToken:    token,
		SyncToken:       hashedSigningKey,
		ConnectionId:    connectionId.String(),
	})
	if err != nil {
		l.Error("could not marshal start response", "err", err)

		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "could not marshal response"))
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	_, _ = w.Write(msg)
}

func (cr *connectApiRouter) flushBuffer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	hashedSigningKey := r.Header.Get("Authorization")
	{
		if hashedSigningKey == "" && !cr.Dev {
			_ = publicerr.WriteHTTP(w, publicerr.Errorf(401, "missing Authorization header"))
			return
		}

		if hashedSigningKey != "" && len(hashedSigningKey) > 7 {
			// Remove "Bearer " prefix
			hashedSigningKey = hashedSigningKey[7:]
		}
	}

	envOverride := r.Header.Get("X-Inngest-Env")

	res, err := cr.RequestAuther.AuthenticateRequest(ctx, hashedSigningKey, envOverride)
	if err != nil {
		logger.StdlibLogger(ctx).Error("could not authenticate connect start request", "err", err, "key", hashedSigningKey, "env", envOverride)

		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 401, "authentication failed"))
		return
	}

	if res == nil {
		_ = publicerr.WriteHTTP(w, publicerr.Errorf(401, "authentication failed"))
		return
	}

	byt, err := io.ReadAll(r.Body)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, "could not read request body"))
		return
	}

	if len(byt) == 0 {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, "missing request body"))
		return
	}

	reqBody := &connect.SDKResponse{}
	if err := proto.Unmarshal(byt, reqBody); err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, "could not unmarshal request"))
		return
	}

	systemTraceCtx := propagation.MapCarrier{}
	if err := json.Unmarshal(reqBody.SystemTraceCtx, &systemTraceCtx); err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, "could not unmarshal system trace ctx"))
		return
	}

	traceCtx := trace.SystemTracer().Propagator().Extract(ctx, systemTraceCtx)
	// nolint:ineffassign,staticcheck
	traceCtx, span := cr.ConditionalTracer.NewSpan(traceCtx, "FlushMessage", res.AccountID, res.EnvID)
	defer span.End()

	// Marshal response before notifying executor, marshaling should never fail
	msg, err := proto.Marshal(&connect.FlushResponse{
		RequestId: reqBody.RequestId,
	})
	if err != nil {
		logger.StdlibLogger(ctx).Error("could not marshal flush response", "err", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "could not marshal flush api response")

		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "could not marshal response"))
		return
	}

	// Reliable path: Buffer the response to be picked up by the executor
	err = cr.ConnectRequestStateManager.SaveResponse(ctx, res.EnvID, reqBody.RequestId, reqBody)
	if err != nil && !errors.Is(err, state.ErrResponseAlreadyBuffered) {
		logger.StdlibLogger(ctx).Error("could not buffer response", "err", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "could not buffer response")

		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "could not buffer response"))
		return
	}

	// Unreliable fast-track: Notify the executor via PubSub (best-effort, this may be dropped)
	if err := cr.ConnectResponseNotifier.NotifyExecutor(ctx, reqBody); err != nil {
		logger.StdlibLogger(ctx).Error("could not notify executor to flush connect message", "err", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "could not notify executor to flush connect sdk response")

		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "could not notify executor"))

		return
	}

	// Send response once executor was notified
	w.Header().Set("Content-Type", "application/octet-stream")
	_, _ = w.Write(msg)
}
