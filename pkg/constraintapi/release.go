package constraintapi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/conditional"
	"github.com/inngest/inngest/pkg/util/errs"
)

type releaseScriptResponse struct {
	Status int                 `json:"s"`
	Debug  flexibleStringArray `json:"d"`

	// Remaining specifies the number of remaining leases
	// generated in the same Acquire operation
	Remaining int `json:"r"`
}

// Release implements CapacityManager.
func (r *redisCapacityManager) Release(ctx context.Context, req *CapacityReleaseRequest) (*CapacityReleaseResponse, errs.InternalError) {
	// Set up conditional observability context for feature flag evaluation
	ctx = conditional.WithContext(ctx,
		conditional.WithAccountID(req.AccountID),
	)

	l := logger.StdlibLogger(ctx)

	// Validate request
	if err := req.Valid(); err != nil {
		return nil, errs.Wrap(0, false, "invalid request: %w", err)
	}

	l = l.With(
		"account_id", req.AccountID,
		"lease_id", req.LeaseID,
		"source", req.Source,
		"migration", req.Migration,
	)
	// Store configured logger in context so scoped loggers can reuse its fields
	ctx = logger.WithStdlib(ctx, l)

	// Retrieve client and key prefix for current constraints
	// NOTE: We will no longer need this once we move to a dedicated store for constraint state
	keyPrefix, client, err := r.clientAndPrefix(req.Migration)
	if err != nil {
		return nil, errs.Wrap(0, false, "could not get client: %w", err)
	}

	// Deterministically compute this based on numScavengerShards and accountID
	scavengerShard := r.scavengerShard(ctx, req.AccountID)

	keys := []string{
		r.keyOperationIdempotency(keyPrefix, req.AccountID, "rel", req.IdempotencyKey),
		r.keyScavengerShard(keyPrefix, scavengerShard),
		r.keyAccountLeases(keyPrefix, req.AccountID),
		r.keyLeaseDetails(keyPrefix, req.AccountID, req.LeaseID),
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
		int(r.operationIdempotencyTTL.Seconds()),
		enableDebugLogsVal,
	})
	if err != nil {
		return nil, errs.Wrap(0, false, "invalid args: %w", err)
	}

	l.Trace(
		"prepared release call",
		"req", req,
		"keys", keys,
		"args", args,
	)

	rawRes, internalErr := executeLuaScript(ctx, "release", req.Migration, client, r.clock, keys, args)
	if internalErr != nil {
		return nil, internalErr
	}

	parsedResponse := releaseScriptResponse{}
	err = json.Unmarshal(rawRes, &parsedResponse)
	if err != nil {
		return nil, errs.Wrap(0, false, "invalid response structure: %w", err)
	}

	res := &CapacityReleaseResponse{
		internalDebugState: parsedResponse,
	}

	switch parsedResponse.Status {
	case 1, 2:
		l.Trace("capacity lease already cleaned up in release")

		// TODO: Track status (1: cleaned up, 2: cleaned up)
		return res, nil
	case 3:
		logger.StdlibLogger(conditional.WithScope(ctx, "constraintapi.HighCardinality")).Debug("capacity released")

		if len(r.lifecycles) > 0 {
			for _, hook := range r.lifecycles {
				err := hook.OnCapacityLeaseReleased(ctx, OnCapacityLeaseReleasedData{
					AccountID: req.AccountID,
					LeaseID:   req.LeaseID,
				})
				if err != nil {
					return nil, errs.Wrap(0, false, "release lifecycle failed: %w", err)
				}
			}
		}

		// TODO: track success
		return res, nil
	default:
		return nil, errs.Wrap(0, false, "unexpected status code %v", parsedResponse.Status)
	}
}
