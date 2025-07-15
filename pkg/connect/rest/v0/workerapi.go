package connectv0

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/oklog/ulid/v2"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"

	connectConfig "github.com/inngest/inngest/pkg/config/connect"
	"github.com/inngest/inngest/pkg/connect/auth"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/inngest/inngest/proto/gen/connect/v1"
	pb "github.com/inngest/inngest/proto/gen/connect/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

func (cr *connectApiRouter) start(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	connectionID, err := ulid.New(ulid.Now(), rand.Reader)
	if err != nil {
		logger.StdlibLogger(ctx).Error("could not generate connection id", "err", err)

		_ = publicerr.WriteHTTP(w, publicerr.Errorf(500, "internal error"))
		return
	}

	l := logger.StdlibLogger(ctx).With("conn_id", connectionID)

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

	gatewayGroup, gatewayURL, err := cr.ConnectGatewayRetriever.RetrieveGateway(ctx, RetrieveGatewayOpts{
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
		GatewayEndpoint: gatewayURL.String(),
		GatewayGroup:    gatewayGroup,
		SessionToken:    token,
		SyncToken:       hashedSigningKey,
		ConnectionId:    connectionID.String(),
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
	l := logger.StdlibLogger(ctx)

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
		l.Error("could not authenticate connect start request", "err", err)

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
	_, span := cr.ConditionalTracer.NewSpan(traceCtx, "FlushMessage", res.AccountID, res.EnvID)
	defer span.End()

	// Marshal response before notifying executor, marshaling should never fail
	msg, err := proto.Marshal(&connect.FlushResponse{
		RequestId: reqBody.RequestId,
	})
	if err != nil {
		l.Error("could not marshal flush response", "err", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "could not marshal flush api response")

		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "could not marshal response"))
		return
	}

	// Reliable path: Buffer the response to be picked up by the executor
	err = cr.ConnectRequestStateManager.SaveResponse(ctx, res.EnvID, reqBody.RequestId, reqBody)
	if err != nil && !errors.Is(err, state.ErrResponseAlreadyBuffered) {
		l.Error("could not buffer response", "err", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "could not buffer response")

		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "could not buffer response"))
		return
	}

	if cr.ShouldUseGRPC(ctx, res.AccountID) {
		ip, err := cr.ConnectRequestStateManager.GetExecutorIP(ctx, res.EnvID, reqBody.RequestId)
		if err != nil {
			// Likely because the request lease has expired
			l.Error("could not get executor IP", "err", err)
			span.RecordError(err)
			span.SetStatus(codes.Error, "could not get executor IP")

			_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "could not get executor"))
			return
		}

		executorIP := ip.String()
		grpcURL := net.JoinHostPort(executorIP, fmt.Sprintf("%d", connectConfig.Executor(ctx).GRPCPort))

		var grpcClient pb.ConnectExecutorClient

		cr.grpcLock.RLock()
		grpcClient = cr.grpcClients[executorIP]
		cr.grpcLock.RUnlock()

		if grpcClient == nil {
			// Upgrade lock to make sure that only one instance is creating a grpc client
			cr.grpcLock.Lock()
			defer cr.grpcLock.Unlock()
			grpcClient = cr.grpcClients[executorIP]

			if grpcClient == nil {
				l.Info("grpc client not found for executor, creating one dynamically", "url", grpcURL)

				conn, err := grpc.NewClient(grpcURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
				if err != nil {
					l.Error("could not create grpc client", "url", grpcURL, "err", err)
					span.RecordError(err)
					span.SetStatus(codes.Error, "could not create grpc client")

					_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "could not create grpc client"))

					return
				}

				grpcClient = pb.NewConnectExecutorClient(conn)
				cr.grpcClients[executorIP] = grpcClient
			}
		}

		result, err := grpcClient.Reply(ctx, &pb.ReplyRequest{Data: reqBody})
		if err != nil || !result.Success {
			l.Error("could not notify executor to flush connect message", "err", err)
			span.RecordError(err)
			span.SetStatus(codes.Error, "could not notify executor to flush connect sdk response")

			_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "could not notify executor"))
			return
		}

	} else {
		// Unreliable fast-track: Notify the executor via PubSub (best-effort, this may be dropped)
		if err := cr.ConnectResponseNotifier.NotifyExecutor(ctx, reqBody); err != nil {
			l.Error("could not notify executor to flush connect message", "err", err)
			span.RecordError(err)
			span.SetStatus(codes.Error, "could not notify executor to flush connect sdk response")

			_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "could not notify executor"))

			return
		}
	}

	// Send response once executor was notified
	w.Header().Set("Content-Type", "application/octet-stream")
	_, _ = w.Write(msg)
}
