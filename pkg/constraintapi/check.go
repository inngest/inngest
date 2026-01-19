package constraintapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/util/errs"
	"github.com/redis/rueidis"
)

type checkRequestData struct {
	EnvID      uuid.UUID `json:"e,omitempty"`
	FunctionID uuid.UUID `json:"f,omitempty"`

	// SortedConstraints represents the list of constraints
	// included in the request sorted to execute in the expected
	// order. Configuration limits are now embedded directly in each constraint.
	SortedConstraints []SerializedConstraintItem `json:"s"`

	// ConfigVersion represents the function version used for this request
	ConfigVersion int `json:"cv,omitempty"`
}

func buildCheckRequestData(req *CapacityCheckRequest, keyPrefix string) (
	[]byte,
	[]ConstraintItem,
	string,
	error,
) {
	state := &checkRequestData{
		EnvID:         req.EnvID,
		FunctionID:    req.FunctionID,
		ConfigVersion: req.Configuration.FunctionVersion,
	}

	// Sort and serialize constraints with embedded configuration limits
	constraints := req.Constraints
	sortConstraints(constraints)

	serialized := make([]SerializedConstraintItem, len(constraints))
	for i := range constraints {
		serialized[i] = constraints[i].ToSerializedConstraintItem(
			req.Configuration,
			req.AccountID,
			req.EnvID,
			req.FunctionID,
			keyPrefix,
		)
	}

	state.SortedConstraints = serialized

	dataBytes, err := json.Marshal(state)
	if err != nil {
		return nil, nil, "", fmt.Errorf("could not marshal request: %w", err)
	}

	// NOTE: We fingerprint the query to apply basic response caching.
	// As Check can be expensive, we don't want to run unnecessary queries
	// that may impact lease and constraint enforcement operations.
	var hash string
	{
		fingerprint := sha256.New()
		_, err = fingerprint.Write(dataBytes)
		if err != nil {
			return nil, nil, "", fmt.Errorf("could not fingerprint query: %w", err)
		}
		hash = hex.EncodeToString(fingerprint.Sum(nil))
	}

	return dataBytes, constraints, hash, nil
}

type checkScriptResponse struct {
	Status              int              `json:"s"`
	AvailableCapacity   int              `json:"a"`
	LimitingConstraints flexibleIntArray `json:"lc"`
	ConstraintUsage     []struct {
		Usage int `json:"u"`
		Limit int `json:"l"`
	} `json:"cu"`
	FairnessReduction int                 `json:"fr"`
	RetryAt           int                 `json:"ra"`
	Debug             flexibleStringArray `json:"d"`
}

// Check implements CapacityManager.
func (r *redisCapacityManager) Check(ctx context.Context, req *CapacityCheckRequest) (*CapacityCheckResponse, errs.UserError, errs.InternalError) {
	l := logger.StdlibLogger(ctx)

	// Validate request
	if err := req.Valid(); err != nil {
		return nil, nil, errs.Wrap(0, false, "invalid request: %w", err)
	}

	l = l.With(
		"account_id", req.AccountID,
		"env_id", req.EnvID,
		"fn_id", req.FunctionID, // May be empty
	)

	// Retrieve client and key prefix for current constraints
	// NOTE: We will no longer need this once we move to a dedicated store for constraint state
	keyPrefix, client, err := r.clientAndPrefix(req.Migration)
	if err != nil {
		return nil, nil, errs.Wrap(0, false, "failed to get client: %w", err)
	}

	data, sortedConstraints, hash, err := buildCheckRequestData(req, keyPrefix)
	if err != nil {
		return nil, nil, errs.Wrap(0, false, "failed to construct request data: %w", err)
	}

	keys := []string{
		r.keyAccountLeases(keyPrefix, req.AccountID),
		r.keyOperationIdempotency(keyPrefix, req.AccountID, "chk", hash),
	}

	enableDebugLogsVal := "0"
	if enableDebugLogs || r.enableDebugLogs {
		enableDebugLogsVal = "1"
	}

	scopedKeyPrefix := fmt.Sprintf("{%s}:%s", keyPrefix, accountScope(req.AccountID))

	now := r.clock.Now()

	args, err := strSlice([]any{
		rueidis.BinaryString(data),
		scopedKeyPrefix,
		req.AccountID,
		now.UnixMilli(),
		now.UnixNano(),
		r.checkIdempotencyTTL.Seconds(),
		enableDebugLogsVal,
	})
	if err != nil {
		return nil, nil, errs.Wrap(0, false, "invalid args: %w", err)
	}

	l.Trace(
		"prepared check call",
		"req", req,
		"keys", keys,
		"args", args,
	)

	rawRes, internalErr := executeLuaScript(ctx, "check", req.Migration, client, r.clock, keys, args)
	if internalErr != nil {
		return nil, nil, internalErr
	}

	parsedResponse := checkScriptResponse{}
	err = json.Unmarshal(rawRes, &parsedResponse)
	if err != nil {
		return nil, nil, errs.Wrap(0, false, "invalid response structure: %w", err)
	}

	var limitingConstraints []ConstraintItem
	if len(parsedResponse.LimitingConstraints) > 0 {
		limitingConstraints = make([]ConstraintItem, len(parsedResponse.LimitingConstraints))
		for i, limitingConstraintIndex := range []int(parsedResponse.LimitingConstraints) {
			limitingConstraints[i] = req.Constraints[limitingConstraintIndex-1]
		}
	}

	constraintUsage := make([]ConstraintUsage, 0, len(req.Constraints))
	if len(parsedResponse.ConstraintUsage) > 0 {
		for i, v := range parsedResponse.ConstraintUsage {
			constraintUsage = append(constraintUsage, ConstraintUsage{
				Constraint: sortedConstraints[i],
				Limit:      v.Limit,
				Used:       v.Usage,
			})
		}
	}

	switch parsedResponse.Status {
	case 1:
		l.Trace("successful check request")

		retryAfter := time.UnixMilli(int64(parsedResponse.RetryAt))
		if retryAfter.Before(now) {
			retryAfter = time.Time{}
		}

		return &CapacityCheckResponse{
			LimitingConstraints: limitingConstraints,
			FairnessReduction:   parsedResponse.FairnessReduction,
			RetryAfter:          retryAfter,
			AvailableCapacity:   parsedResponse.AvailableCapacity,
			Usage:               constraintUsage,
			internalDebugState:  parsedResponse,
		}, nil, nil
	default:
		return nil, nil, errs.Wrap(0, false, "unexpected status code %v", parsedResponse.Status)
	}
}
