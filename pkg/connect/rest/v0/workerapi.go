package v0

import (
	"encoding/json"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/trace"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"io"
	"net/http"

	"github.com/golang/protobuf/proto"
	"github.com/inngest/inngest/pkg/connect/auth"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/inngest/inngest/proto/gen/connect/v1"
)

func (a *connectApiRouter) start(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	hashedSigningKey := r.Header.Get("Authorization")
	{
		if hashedSigningKey == "" && !a.Dev {
			_ = publicerr.WriteHTTP(w, publicerr.Errorf(401, "missing Authorization header"))
			return
		}

		if hashedSigningKey != "" && len(hashedSigningKey) > 7 {
			// Remove "Bearer " prefix
			hashedSigningKey = hashedSigningKey[7:]
		}
	}

	envOverride := r.Header.Get("X-Inngest-Env")

	res, err := a.RequestAuther.AuthenticateRequest(ctx, hashedSigningKey, envOverride)
	if err != nil {
		logger.StdlibLogger(ctx).Error("could not authenticate connect start request", "err", err, "key", hashedSigningKey, "env", envOverride)

		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 401, "authentication failed"))
		return
	}

	if res == nil {
		_ = publicerr.WriteHTTP(w, publicerr.Errorf(401, "authentication failed"))
		return
	}

	allowed, err := a.ConnectionLimiter.CheckConnectionLimit(ctx, res)
	if err != nil {
		logger.StdlibLogger(ctx).Error("could not check connection limit during start request", "err", err, "key", hashedSigningKey, "env", envOverride)

		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "could not check connection limit"))
		return
	}

	if !allowed {
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

	token, err := a.Signer.SignSessionToken(res.AccountID, res.EnvID, auth.DefaultExpiry)
	if err != nil {
		logger.StdlibLogger(ctx).Error("could not sign connect session token", "err", err, "key", hashedSigningKey, "env", envOverride)

		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "could not sign session token"))
		return
	}

	gatewayGroup, gatewayUrl, err := a.ConnectGatewayRetriever.RetrieveGateway(ctx, RetrieveGatewayOpts{
		AccountID:   res.AccountID,
		EnvID:       res.EnvID,
		Exclude:     reqBody.ExcludeGateways,
		RequestHost: r.Host,
	})
	if err != nil {
		logger.StdlibLogger(ctx).Error("could not retrieve connect gateway", "err", err, "key", hashedSigningKey, "env", envOverride)

		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "could not retrieve gateway"))
		return
	}

	msg, err := proto.Marshal(&connect.StartResponse{
		GatewayEndpoint: gatewayUrl.String(),
		GatewayGroup:    gatewayGroup,
		SessionToken:    token,
		SyncToken:       hashedSigningKey,
	})
	if err != nil {
		logger.StdlibLogger(ctx).Error("could not marshal start response", "err", err)

		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "could not marshal response"))
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	_, _ = w.Write(msg)
}

func (a *connectApiRouter) flushBuffer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	hashedSigningKey := r.Header.Get("Authorization")
	{
		if hashedSigningKey == "" && !a.Dev {
			_ = publicerr.WriteHTTP(w, publicerr.Errorf(401, "missing Authorization header"))
			return
		}

		if hashedSigningKey != "" && len(hashedSigningKey) > 7 {
			// Remove "Bearer " prefix
			hashedSigningKey = hashedSigningKey[7:]
		}
	}

	envOverride := r.Header.Get("X-Inngest-Env")

	res, err := a.RequestAuther.AuthenticateRequest(ctx, hashedSigningKey, envOverride)
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
	traceCtx, span := a.ConditionalTracer.NewSpan(traceCtx, "FlushMessage", res.AccountID, res.EnvID)
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

	if err := a.ConnectResponseNotifier.NotifyExecutor(ctx, reqBody); err != nil {
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
