package constraintapi

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"

	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/util/errs"
	"github.com/oklog/ulid/v2"
)

type extendLeaseScriptResponse struct {
	Status  int                 `json:"s"`
	Debug   flexibleStringArray `json:"d"`
	LeaseID ulid.ULID           `json:"lid"`
}

// ExtendLease implements CapacityManager.
func (r *redisCapacityManager) ExtendLease(ctx context.Context, req *CapacityExtendLeaseRequest) (*CapacityExtendLeaseResponse, errs.InternalError) {
	l := logger.StdlibLogger(ctx)

	// Validate request
	if err := req.Valid(); err != nil {
		return nil, errs.Wrap(0, false, "invalid request: %w", err)
	}

	l = l.With(
		"account_id", req.AccountID,
		"lease_id", req.LeaseID,
	)

	now := r.clock.Now()

	// Retrieve client and key prefix for current constraints
	// NOTE: We will no longer need this once we move to a dedicated store for constraint state
	keyPrefix, client, err := r.clientAndPrefix(req.Migration)
	if err != nil {
		return nil, errs.Wrap(0, false, "failed to get client: %w", err)
	}

	// Deterministically compute this based on numScavengerShards and accountID
	scavengerShard := r.scavengerShard(ctx, req.AccountID)

	leaseExpiry := now.Add(req.Duration)
	newLeaseID, err := ulid.New(ulid.Timestamp(leaseExpiry), rand.Reader)
	if err != nil {
		return nil, errs.Wrap(0, false, "failed to generate new lease ID: %w", err)
	}

	keys := []string{
		r.keyOperationIdempotency(keyPrefix, req.AccountID, "ext", req.IdempotencyKey),
		r.keyScavengerShard(keyPrefix, scavengerShard),
		r.keyAccountLeases(keyPrefix, req.AccountID),
		r.keyLeaseDetails(keyPrefix, req.AccountID, req.LeaseID),
		r.keyLeaseDetails(keyPrefix, req.AccountID, newLeaseID),
	}

	enableDebugLogsVal := "0"
	if enableDebugLogs || r.enableDebugLogs {
		enableDebugLogsVal = "1"
	}

	scopedKeyPrefix := fmt.Sprintf("{%s}:%s", keyPrefix, accountScope(req.AccountID))

	args, err := strSlice([]any{
		scopedKeyPrefix,
		req.AccountID,
		req.LeaseID.String(),
		newLeaseID.String(),
		now.UnixMilli(), // current time in milliseconds for throttle
		leaseExpiry.UnixMilli(),
		int(r.operationIdempotencyTTL.Seconds()),
		enableDebugLogsVal,
	})
	if err != nil {
		return nil, errs.Wrap(0, false, "invalid args: %w", err)
	}

	l.Trace(
		"prepared extend call",
		"req", req,
		"keys", keys,
		"args", args,
	)

	rawRes, internalErr := executeLuaScript(ctx, "extend", client, r.clock, keys, args)
	if internalErr != nil {
		return nil, internalErr
	}

	parsedResponse := extendLeaseScriptResponse{}
	err = json.Unmarshal(rawRes, &parsedResponse)
	if err != nil {
		return nil, errs.Wrap(0, false, "invalid response structure: %w", err)
	}

	res := &CapacityExtendLeaseResponse{
		internalDebugState: parsedResponse,
	}
	if parsedResponse.LeaseID != ulid.Zero {
		res.LeaseID = &parsedResponse.LeaseID
	}

	switch parsedResponse.Status {
	case 1, 2, 3:
		l.Trace("capacity lease in extend call already cleaned up")

		// TODO: Track status (1: cleaned up, 2: cleaned up or lease superseded, 3: lease expired)
		return res, nil
	case 4:
		l.Trace("extended capacity lease")

		if len(r.lifecycles) > 0 {
			for _, hook := range r.lifecycles {
				err := hook.OnCapacityLeaseExtended(ctx, OnCapacityLeaseExtendedData{
					AccountID:  req.AccountID,
					Duration:   req.Duration,
					OldLeaseID: req.LeaseID,
					NewLeaseID: parsedResponse.LeaseID,
				})
				if err != nil {
					return nil, errs.Wrap(0, false, "extend lifecycle failed: %w", err)
				}
			}
		}

		// TODO: track success
		return res, nil
	default:
		return nil, errs.Wrap(0, false, "unexpected status code %v", parsedResponse.Status)
	}
}
