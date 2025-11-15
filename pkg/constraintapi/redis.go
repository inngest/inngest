package constraintapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/util"
	"github.com/inngest/inngest/pkg/util/errs"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
)

const (
	MaximumAllowedRequestDelay = time.Second
	// OperationIdempotencyTTL represents the time the same response will be returned after a successful request.
	// Depending on the operation, this should be low (Otherwise, Acquire may return an already expired lease)
	// TODO: Figure out a reasonable operation idempotency TTL (maybe per-operation)
	OperationIdempotencyTTL       = 5 * time.Second
	CheckIdempotencyTTL           = 5 * time.Second
	ConstraintCheckIdempotencyTTL = 5 * time.Minute
)

var enableDebugLogs = false

type redisCapacityManager struct {
	// Until fully rolled out, the Constraint API will use the existing data stores
	// for accessing and modifying existing constraint state, as well as lease-related state.
	//
	// This means, we need to connect to all queue shards, as well as the instance
	// responsible for storing rate limit state.
	//
	// In a future release, we will gracefully migrate all constraint and lease state to a
	// dedicated horizontally-scalable and account-sharded backing data store.
	queueShards     map[string]rueidis.Client
	rateLimitClient rueidis.Client

	clock clockwork.Clock

	rateLimitKeyPrefix  string
	queueStateKeyPrefix string

	numScavengerShards int
}

type redisCapacityManagerOption func(m *redisCapacityManager)

func WithQueueShards(shards map[string]rueidis.Client) redisCapacityManagerOption {
	return func(m *redisCapacityManager) {
		m.queueShards = shards
	}
}

func WithQueueStateKeyPrefix(prefix string) redisCapacityManagerOption {
	return func(m *redisCapacityManager) {
		m.queueStateKeyPrefix = prefix
	}
}

func WithRateLimitClient(client rueidis.Client) redisCapacityManagerOption {
	return func(m *redisCapacityManager) {
		m.rateLimitClient = client
	}
}

func WithRateLimitKeyPrefix(prefix string) redisCapacityManagerOption {
	return func(m *redisCapacityManager) {
		m.rateLimitKeyPrefix = prefix
	}
}

func WithClock(clock clockwork.Clock) redisCapacityManagerOption {
	return func(m *redisCapacityManager) {
		m.clock = clock
	}
}

func WithNumScavengerShards(numShards int) redisCapacityManagerOption {
	return func(m *redisCapacityManager) {
		m.numScavengerShards = numShards
	}
}

func NewRedisCapacityManager(
	options ...redisCapacityManagerOption,
) (*redisCapacityManager, error) {
	manager := &redisCapacityManager{}

	for _, rcmo := range options {
		rcmo(manager)
	}

	if manager.rateLimitClient == nil || manager.queueShards == nil {
		return nil, fmt.Errorf("missing clients")
	}

	if manager.clock == nil {
		manager.clock = clockwork.NewRealClock()
	}

	return manager, nil
}

// keyScavengerShard represents the top-level sharded sorted set containing individual accounts
func (r *redisCapacityManager) keyScavengerShard(prefix string, shard int) string {
	return fmt.Sprintf("{%s}:css:%d", prefix, shard)
}

// keyAccountLeases represents active leases for the account
func (r *redisCapacityManager) keyAccountLeases(prefix string, accountID uuid.UUID) string {
	return fmt.Sprintf("{%s}:%s:leaseq", prefix, accountID)
}

// keyRequestState returns the key storing per-operation request details
func (r *redisCapacityManager) keyRequestState(prefix string, accountID uuid.UUID, operationIdempotencyKey string) string {
	return fmt.Sprintf("{%s}:%s:rs:%s", prefix, accountID, util.XXHash(operationIdempotencyKey))
}

// keyOperationIdempotency returns the operation idempotency key for operation retries
func (r *redisCapacityManager) keyOperationIdempotency(prefix string, accountID uuid.UUID, operation, idempotencyKey string) string {
	return fmt.Sprintf("{%s}:%s:ik:op:%s:%s", prefix, accountID, operation, util.XXHash(idempotencyKey))
}

// keyConstraintCheckIdempotency returns the operation idempotency key for constraint check retries
func (r *redisCapacityManager) keyConstraintCheckIdempotency(prefix string, accountID uuid.UUID, idempotencyKey string) string {
	return fmt.Sprintf("{%s}:%s:ik:cc:%s", prefix, accountID, util.XXHash(idempotencyKey))
}

// keyLeaseDetails returns the key to the hash including the lease idempotency key, lease run ID, and operation idempotency key
func (r *redisCapacityManager) keyLeaseDetails(prefix string, accountID uuid.UUID, leaseID ulid.ULID) string {
	return fmt.Sprintf("{%s}:%s:ld:%s", prefix, accountID, leaseID)
}

// clientAndPrefix returns the Redis client and Lua key prefix for the first stage of the Constraint API.
//
// Since we are colocating lease data with the existing state, we will have to use the
// same Redis hash tag to avoid Lua errors and inconsistencies on the old and new scripts.
//
// This is essentially required for backward- and forward-compatibility.
func (r *redisCapacityManager) clientAndPrefix(m MigrationIdentifier) (string, rueidis.Client, error) {
	// TODO: Once we support new data stores, we can return those clients here, including a per-account hash tag prefix, e.g. <accountID>

	if m.IsRateLimit {
		return r.rateLimitKeyPrefix, r.rateLimitClient, nil
	}

	shard, ok := r.queueShards[m.QueueShard]
	if !ok {
		return "", nil, fmt.Errorf("unknown queue shard %q", m.QueueShard)
	}

	return r.queueStateKeyPrefix, shard, nil
}

// redisRequestState represents the data structure stored for every request
// This is used by subsequent calls to Extend, Release to properly handle the lease lifecycle
//
// NOTE: This does not represent one individual lease but is used by
// all leases generated in the Acquire call.
type redisRequestState struct {
	OperationIdempotencyKey string    `json:"k,omitempty"`
	EnvID                   uuid.UUID `json:"e,omitempty"`
	FunctionID              uuid.UUID `json:"f,omitempty"`

	// SortedConstraints represents the list of constraints
	// included in the request sorted to execute in the expected
	// order. Configuration limits are now embedded directly in each constraint.
	SortedConstraints []SerializedConstraintItem `json:"s"`

	// ConfigVersion represents the function version used for this request
	ConfigVersion int `json:"cv,omitempty"`

	// RequestedAmount represents the Amount field in the Acquire request
	RequestedAmount int `json:"r,omitempty"`

	// GrantedAmount is populated in Lua during Acquire and represents the actual capacity granted to the request (how many leases were generated)
	GrantedAmount int `json:"g,omitempty"`

	// ActiveAmount represents the total number of active leases (where Release was not yet called)
	ActiveAmount int `json:"a,omitempty"`

	// MaximumLifetime is optional and represenst the maximum lifetime for leases generated by this request.
	// This is enforced during ExtendLease.
	MaximumLifetimeMillis int64 `json:"l,omitempty"`

	// LeaseIdempotencyKeys stores the idempotency used to generate leases
	LeaseIdempotencyKeys []string `json:"lik,omitempty"`

	// LeaseRunIDs stores the run IDs associated with lease IDs
	LeaseRunIDs map[string]ulid.ULID `json:"lri,omitempty"`
}

func buildRequestState(req *CapacityAcquireRequest, keyPrefix string) (*redisRequestState, []ConstraintItem) {
	state := &redisRequestState{
		OperationIdempotencyKey: req.IdempotencyKey,
		EnvID:                   req.EnvID,
		FunctionID:              req.FunctionID,
		RequestedAmount:         req.Amount,
		MaximumLifetimeMillis:   req.MaximumLifetime.Milliseconds(),
		ConfigVersion:           req.Configuration.FunctionVersion,

		LeaseRunIDs:          req.LeaseRunIDs,
		LeaseIdempotencyKeys: req.LeaseIdempotencyKeys,

		// These keys are set during Acquire and Release respectively
		GrantedAmount: 0,
		ActiveAmount:  0,
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

	return state, constraints
}

type acquireScriptResponse struct {
	Status        int `json:"s"`
	Requested     int `json:"r"`
	Granted       int `json:"g"`
	GrantedLeases []struct {
		LeaseID             ulid.ULID `json:"lid"`
		LeaseIdempotencyKey string    `json:"lik"`
	} `json:"l"`
	LimitingConstraints []int    `json:"lc"`
	FairnessReduction   int      `json:"fr"`
	RetryAt             int      `json:"ra"`
	Debug               []string `json:"d"`
}

// Acquire implements CapacityManager.
func (r *redisCapacityManager) Acquire(ctx context.Context, req *CapacityAcquireRequest) (*CapacityAcquireResponse, errs.InternalError) {
	// Validate request
	if err := req.Valid(); err != nil {
		return nil, errs.Wrap(0, false, "invalid request: %w", err)
	}

	now := r.clock.Now()

	// TODO: Add metric for this
	// NOTE: This will include request latency (marshaling, network delays),
	// and it might not work for retries, as those retain the same CurrentTime value.
	// TODO: Ensure retries have the updated CurrentTime
	requestLatency := now.Sub(req.CurrentTime)
	if requestLatency > MaximumAllowedRequestDelay {
		// TODO : Set proper error code
		return nil, errs.Wrap(0, false, "exceeded maximum allowed request delay")
	}

	// Retrieve client and key prefix for current constraints
	// NOTE: We will no longer need this once we move to a dedicated store for constraint state
	keyPrefix, client, err := r.clientAndPrefix(req.Migration)
	if err != nil {
		return nil, errs.Wrap(0, false, "failed to get client: %w", err)
	}

	// TODO: Should we get the current time again/cancel if too much time passed up until here?
	leaseExpiry := now.Add(req.Duration)

	// Generate lease IDs
	initialLeaseIDs := make([]ulid.ULID, len(req.LeaseIdempotencyKeys))
	for i := range req.LeaseIdempotencyKeys {
		leaseID, err := ulid.New(ulid.Timestamp(leaseExpiry), rand.Reader)
		if err != nil {
			return nil, errs.Wrap(0, true, "failed to generate lease IDs: %w", err)
		}
		initialLeaseIDs[i] = leaseID
	}

	requestState, sortedConstraints := buildRequestState(req, keyPrefix)

	// TODO: Deterministically compute this based on numScavengerShards and accountID
	scavengerShard := 0

	// Build Lua request

	keys := []string{
		r.keyRequestState(keyPrefix, req.AccountID, req.IdempotencyKey),
		r.keyOperationIdempotency(keyPrefix, req.AccountID, "acq", req.IdempotencyKey),
		r.keyConstraintCheckIdempotency(keyPrefix, req.AccountID, req.IdempotencyKey),
		r.keyScavengerShard(keyPrefix, scavengerShard),
		r.keyAccountLeases(keyPrefix, req.AccountID),
	}

	enableDebugLogsVal := "0"
	if enableDebugLogs {
		enableDebugLogsVal = "1"
	}

	args, err := strSlice([]any{
		// This will be marshaled
		requestState,
		req.AccountID,
		now.UnixMilli(), // current time in milliseconds for throttle
		now.UnixNano(),  // current time in nanoseconds for rate limiting

		leaseExpiry.UnixMilli(),
		keyPrefix,
		initialLeaseIDs,

		util.XXHash(req.IdempotencyKey), // hashed operation idempotency key
		int(OperationIdempotencyTTL.Seconds()),
		int(ConstraintCheckIdempotencyTTL.Seconds()),

		enableDebugLogsVal,
	})
	if err != nil {
		return nil, errs.Wrap(0, false, "invalid args: %w", err)
	}

	rawRes, err := scripts["acquire"].Exec(ctx, client, keys, args).AsBytes()
	if err != nil {
		return nil, errs.Wrap(0, false, "acquire script failed: %w", err)
	}

	parsedResponse := acquireScriptResponse{}
	err = json.Unmarshal(rawRes, &parsedResponse)
	if err != nil {
		return nil, errs.Wrap(0, false, "invalid response structure: %w", err)
	}

	leases := make([]CapacityLease, len(parsedResponse.GrantedLeases))
	for i, v := range parsedResponse.GrantedLeases {
		leases[i] = CapacityLease{
			LeaseID:        v.LeaseID,
			IdempotencyKey: v.LeaseIdempotencyKey,
		}
	}

	var limitingConstraints []ConstraintItem
	if len(parsedResponse.LimitingConstraints) > 0 {
		limitingConstraints = make([]ConstraintItem, len(parsedResponse.LimitingConstraints))
		for i, limitingConstraintIndex := range parsedResponse.LimitingConstraints {
			limitingConstraints[i] = sortedConstraints[limitingConstraintIndex-1]
		}
	}

	switch parsedResponse.Status {
	case 1, 3:
		// success or idempotency
		return &CapacityAcquireResponse{
			Leases:              leases,
			LimitingConstraints: limitingConstraints,
			FairnessReduction:   parsedResponse.FairnessReduction,
			internalDebugState:  parsedResponse,
		}, nil

	case 2:
		// lacking capacity
		return &CapacityAcquireResponse{
			Leases:              leases,
			LimitingConstraints: limitingConstraints,
			RetryAfter:          time.UnixMilli(int64(parsedResponse.RetryAt)),
			FairnessReduction:   parsedResponse.FairnessReduction,
			internalDebugState:  parsedResponse,
		}, nil

	default:
		return nil, errs.Wrap(0, false, "unexpected status code %v", parsedResponse.Status)
	}
}

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
	*checkRequestData,
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

	// NOTE: We fingerprint the query to apply basic response caching.
	// As Check can be expensive, we don't want to run unnecessary queries
	// that may impact lease and constraint enforcement operations.
	var hash string
	{
		dataBytes, err := json.Marshal(state)
		if err != nil {
			return nil, nil, "", fmt.Errorf("could not marshal request: %w", err)
		}

		fingerprint := sha256.New()
		_, err = fingerprint.Write(dataBytes)
		if err != nil {
			return nil, nil, "", fmt.Errorf("could not fingerprint query: %w", err)
		}
		hash = hex.EncodeToString(fingerprint.Sum(nil))
	}

	return state, constraints, hash, nil
}

type checkScriptResponse struct {
	Status              int   `json:"s"`
	AvailableCapacity   int   `json:"a"`
	LimitingConstraints []int `json:"lc"`
	ConstraintUsage     []struct {
		Usage int `json:"u"`
		Limit int `json:"l"`
	} `json:"cu"`
	FairnessReduction int      `json:"fr"`
	RetryAt           int      `json:"ra"`
	Debug             []string `json:"d"`
}

// Check implements CapacityManager.
func (r *redisCapacityManager) Check(ctx context.Context, req *CapacityCheckRequest) (*CapacityCheckResponse, errs.UserError, errs.InternalError) {
	// Validate request
	if err := req.Valid(); err != nil {
		return nil, nil, errs.Wrap(0, false, "invalid request: %w", err)
	}

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
	if enableDebugLogs {
		enableDebugLogsVal = "1"
	}

	now := r.clock.Now()

	args, err := strSlice([]any{
		data,
		keyPrefix,
		req.AccountID,
		now.UnixMilli(),
		now.UnixNano(),
		CheckIdempotencyTTL.Seconds(),
		enableDebugLogsVal,
	})
	if err != nil {
		return nil, nil, errs.Wrap(0, false, "invalid args: %w", err)
	}

	rawRes, err := scripts["check"].Exec(ctx, client, keys, args).AsBytes()
	if err != nil {
		return nil, nil, errs.Wrap(0, false, "check script failed: %w", err)
	}

	parsedResponse := checkScriptResponse{}
	err = json.Unmarshal(rawRes, &parsedResponse)
	if err != nil {
		return nil, nil, errs.Wrap(0, false, "invalid response structure: %w", err)
	}

	var limitingConstraints []ConstraintItem
	if len(parsedResponse.LimitingConstraints) > 0 {
		limitingConstraints = make([]ConstraintItem, len(parsedResponse.LimitingConstraints))
		for i, limitingConstraintIndex := range parsedResponse.LimitingConstraints {
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
		return &CapacityCheckResponse{
			LimitingConstraints: limitingConstraints,
			FairnessReduction:   parsedResponse.FairnessReduction,
			RetryAfter:          time.UnixMilli(int64(parsedResponse.RetryAt)),
			AvailableCapacity:   parsedResponse.AvailableCapacity,
			Usage:               constraintUsage,
			internalDebugState:  parsedResponse,
		}, nil, nil
	default:
		return nil, nil, errs.Wrap(0, false, "unexpected status code %v", parsedResponse.Status)
	}
}

type extendLeaseScriptResponse struct {
	Status  int       `json:"s"`
	Debug   []string  `json:"d"`
	LeaseID ulid.ULID `json:"lid"`
}

// ExtendLease implements CapacityManager.
func (r *redisCapacityManager) ExtendLease(ctx context.Context, req *CapacityExtendLeaseRequest) (*CapacityExtendLeaseResponse, errs.InternalError) {
	// Validate request
	if err := req.Valid(); err != nil {
		return nil, errs.Wrap(0, false, "invalid request: %w", err)
	}

	now := r.clock.Now()

	// Retrieve client and key prefix for current constraints
	// NOTE: We will no longer need this once we move to a dedicated store for constraint state
	keyPrefix, client, err := r.clientAndPrefix(req.Migration)
	if err != nil {
		return nil, errs.Wrap(0, false, "failed to get client: %w", err)
	}

	// TODO: Deterministically compute this based on numScavengerShards and accountID
	scavengerShard := 0

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
	if enableDebugLogs {
		enableDebugLogsVal = "1"
	}

	args, err := strSlice([]any{
		keyPrefix,
		req.AccountID,
		req.LeaseID.String(),
		newLeaseID.String(),
		now.UnixMilli(), // current time in milliseconds for throttle
		leaseExpiry.UnixMilli(),
		int(OperationIdempotencyTTL.Seconds()),
		enableDebugLogsVal,
	})
	if err != nil {
		return nil, errs.Wrap(0, false, "invalid args: %w", err)
	}

	rawRes, err := scripts["extend"].Exec(ctx, client, keys, args).AsBytes()
	if err != nil {
		return nil, errs.Wrap(0, false, "extend script failed: %w", err)
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
		// TODO: Track status (1: cleaned up, 2: cleaned up or lease superseded, 3: lease expired)
		return res, nil
	case 4:
		// TODO: track success
		return res, nil
	default:
		return nil, errs.Wrap(0, false, "unexpected status code %v", parsedResponse.Status)
	}
}

type releaseScriptResponse struct {
	Status int      `json:"s"`
	Debug  []string `json:"d"`

	// Remaining specifies the number of remaining leases
	// generated in the same Acquire operation
	Remaining int `json:"r"`
}

// Release implements CapacityManager.
func (r *redisCapacityManager) Release(ctx context.Context, req *CapacityReleaseRequest) (*CapacityReleaseResponse, errs.InternalError) {
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

	// TODO: Deterministically compute this based on numScavengerShards and accountID
	scavengerShard := 0

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
		// TODO: Track status (1: cleaned up, 2: cleaned up or lease superseded, 3: lease expired)
		return res, nil
	case 3:
		// TODO: track success
		return res, nil
	default:
		return nil, errs.Wrap(0, false, "unexpected status code %v", parsedResponse.Status)
	}
}
