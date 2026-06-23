package connectv0

import (
	"errors"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/publicerr"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"google.golang.org/protobuf/proto"
)

// extendLease is an HTTP fallback for extending request leases when the
// WebSocket connection is unavailable (e.g., during gateway drain/reconnection).
//
// This bypasses the gateway entirely — the SDK calls the API server directly,
// which extends the lease in the shared Redis state.
func (cr *connectApiRouter) extendLease(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.StdlibLogger(ctx)

	// Auth — same pattern as start/flush
	hashedSigningKey := r.Header.Get("Authorization")
	if hashedSigningKey == "" && !cr.Dev {
		_ = publicerr.WriteHTTP(w, publicerr.Errorf(401, "missing Authorization header"))
		return
	}
	if hashedSigningKey != "" && len(hashedSigningKey) > 7 {
		hashedSigningKey = hashedSigningKey[7:]
	}

	envOverride := r.Header.Get("X-Inngest-Env")

	res, err := cr.RequestAuther.AuthenticateRequest(ctx, hashedSigningKey, envOverride)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 401, "authentication failed"))
		return
	}
	if res == nil {
		_ = publicerr.WriteHTTP(w, publicerr.Errorf(401, "authentication failed"))
		return
	}

	// Parse request body
	byt, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, "could not read request body"))
		return
	}
	if len(byt) == 0 {
		_ = publicerr.WriteHTTP(w, publicerr.Errorf(400, "missing request body"))
		return
	}

	var data connectpb.WorkerRequestExtendLeaseData
	if err := proto.Unmarshal(byt, &data); err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, "could not unmarshal request"))
		return
	}

	leaseID, err := ulid.Parse(data.LeaseId)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, "invalid lease ID"))
		return
	}

	envID, err := uuid.Parse(data.EnvId)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, "invalid env ID"))
		return
	}

	// Get the assigned worker instance ID for this request
	instanceID, err := cr.ConnectRequestStateManager.GetAssignedWorkerID(ctx, envID, data.RequestId)
	if err != nil {
		l.Error("could not get assigned worker ID", "err", err, "req_id", data.RequestId)
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "could not get assigned worker"))
		return
	}

	// Extend the lease. We pass isWorkerCapacityUnlimited=true because this is
	// only extending an existing lease, not assigning new work. Capacity tracking
	// is handled by the normal WebSocket path when the connection is restored.
	newLeaseID, err := cr.ConnectRequestStateManager.ExtendRequestLease(
		ctx,
		envID,
		instanceID,
		data.RequestId,
		leaseID,
		consts.ConnectWorkerRequestLeaseDuration,
		true, // skip capacity enforcement for HTTP fallback
	)
	if err != nil {
		switch {
		case errors.Is(err, state.ErrRequestLeaseExpired),
			errors.Is(err, state.ErrRequestLeased),
			errors.Is(err, state.ErrRequestLeaseNotFound),
			errors.Is(err, state.ErrRequestWorkerDoesNotExist):
			l.Debug("lease extension failed", "err", err, "req_id", data.RequestId)
		default:
			l.Error("unexpected error extending lease via HTTP", "err", err, "req_id", data.RequestId)
		}

		// Nack: no new lease ID
		resp, _ := proto.Marshal(&connectpb.WorkerRequestExtendLeaseAckData{
			RequestId:    data.RequestId,
			AccountId:    data.AccountId,
			EnvId:        data.EnvId,
			AppId:        data.AppId,
			FunctionSlug: data.FunctionSlug,
			NewLeaseId:   nil,
		})
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write(resp)
		return
	}

	// Ack with new lease ID
	var newLeaseIDStr *string
	if newLeaseID != nil {
		s := newLeaseID.String()
		newLeaseIDStr = &s
	}

	resp, err := proto.Marshal(&connectpb.WorkerRequestExtendLeaseAckData{
		RequestId:    data.RequestId,
		AccountId:    data.AccountId,
		EnvId:        data.EnvId,
		AppId:        data.AppId,
		FunctionSlug: data.FunctionSlug,
		NewLeaseId:   newLeaseIDStr,
	})
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "could not marshal response"))
		return
	}

	l.Debug("extended lease via HTTP fallback", "req_id", data.RequestId)

	w.Header().Set("Content-Type", "application/octet-stream")
	_, _ = w.Write(resp)
}
