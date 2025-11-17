package constraintapi

import (
	"context"
	"encoding/json"

	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/util/errs"
)

type releaseScriptResponse struct {
	Status int      `json:"s"`
	Debug  []string `json:"d"`

	// Remaining specifies the number of remaining leases
	// generated in the same Acquire operation
	Remaining int `json:"r"`
}

// Release implements CapacityManager.
func (r *redisCapacityManager) Release(ctx context.Context, req *CapacityReleaseRequest) (*CapacityReleaseResponse, errs.InternalError) {
	l := logger.StdlibLogger(ctx)

	// Validate request
	if err := req.Valid(); err != nil {
		return nil, errs.Wrap(0, false, "invalid request: %w", err)
	}

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
	if enableDebugLogs {
		enableDebugLogsVal = "1"
	}

	args, err := strSlice([]any{
		keyPrefix,
		req.AccountID,
		req.LeaseID.String(),
		int(OperationIdempotencyTTL.Seconds()),
		enableDebugLogsVal,
	})
	if err != nil {
		return nil, errs.Wrap(0, false, "invalid args: %w", err)
	}

	rawRes, err := scripts["release"].Exec(ctx, client, keys, args).AsBytes()
	if err != nil {
		return nil, errs.Wrap(0, false, "release script failed: %w", err)
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
		l.Trace("capacity released")

		// TODO: track success
		return res, nil
	default:
		return nil, errs.Wrap(0, false, "unexpected status code %v", parsedResponse.Status)
	}
}
