package redis_state

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/telemetry/redis_telemetry"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/config/registration"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
)

//go:embed lua/*
var embedded embed.FS

var (
	// scripts stores all embedded lua scripts on initialization
	scripts          = map[string]*rueidis.Lua{}
	retriableScripts = map[string]*RetriableLua{}
	include          = regexp.MustCompile(`-- \$include\(([\w.]+)\)`)

	// A number to version backend logic in order to prevent non-backward compatible
	// changes to break
	currentVersion = 1
)

func init() {
	// register the redis driver
	registration.RegisterState(func() any { return registration.StateConfig(&Config{}) })
	registration.RegisterQueue(func() any { return registration.QueueConfig(&queueConfig{}) })

	// read the lua scripts
	entries, err := embedded.ReadDir("lua")
	if err != nil {
		panic(fmt.Errorf("error reading redis lua dir: %w", err))
	}

	readRedisScripts("lua", entries)
}

func readRedisScripts(path string, entries []fs.DirEntry) {
	for _, e := range entries {
		// NOTE: When using embed go always uses forward slashes as a path
		// prefix. filepath.Join uses OS-specific prefixes which fails on
		// windows, so we construct the path using Sprintf for all platforms
		if e.IsDir() {
			entries, _ := embedded.ReadDir(fmt.Sprintf("%s/%s", path, e.Name()))
			readRedisScripts(path+"/"+e.Name(), entries)
			continue
		}

		byt, err := embedded.ReadFile(fmt.Sprintf("%s/%s", path, e.Name()))
		if err != nil {
			panic(fmt.Errorf("error reading redis lua script: %w", err))
		}

		name := path + "/" + e.Name()
		name = strings.TrimPrefix(name, "lua/")
		name = strings.TrimSuffix(name, ".lua")
		val := string(byt)

		// Add any includes.
		items := include.FindAllStringSubmatch(val, -1)
		if len(items) > 0 {
			// Replace each include
			for _, include := range items {
				byt, err = embedded.ReadFile(fmt.Sprintf("lua/includes/%s", include[1]))
				if err != nil {
					panic(fmt.Errorf("error reading redis lua include: %w", err))
				}
				val = strings.ReplaceAll(val, include[0], string(byt))
			}
		}

		scripts[name] = rueidis.NewLuaScript(val)
		retriableScripts[name] = NewClusterLuaScript(val)
	}
}

type queueConfig struct{}

func (c queueConfig) QueueName() string             { return "redis" }
func (c queueConfig) Queue() (osqueue.Queue, error) { return nil, nil }
func (c queueConfig) Consumer() osqueue.Consumer    { return nil }
func (c queueConfig) Producer() osqueue.Producer    { return nil }

// Config registers the configuration for the in-memory state store,
// and provides a factory for the state manager based off of the config.
type Config struct {
	// DSN contains the entire configuration in a single string, if
	// provided (eg. redis://user:pass@host:port/db)
	// DSN *string

	Host       string
	Port       int
	DB         int
	Username   string
	Password   string
	MaxRetries *int
	PoolSize   *int

	KeyPrefix string

	// Expiry represents the expiration time on values stored in state.
	// This defaults to 0, ie. no expiry TTL.
	Expiry time.Duration
}

func (c Config) StateName() string { return "redis" }

// SingleClusterManager returns a state manager connecting to just one Redis instance. Do not use this when separate instances
// should be used for sharded/unsharded data
func (c Config) SingleClusterManager(ctx context.Context) (state.Manager, error) {
	opts, err := c.ConnectOpts()
	if err != nil {
		return nil, err
	}

	r, err := rueidis.NewClient(opts)
	if err != nil {
		return nil, err
	}

	u := NewUnshardedClient(r, StateDefaultKey, QueueDefaultKey)
	s := NewShardedClient(ShardedClientOpts{
		UnshardedClient:        u,
		FunctionRunStateClient: r,
		BatchClient:            r,
		StateDefaultKey:        StateDefaultKey,
		QueueDefaultKey:        QueueDefaultKey,
		FnRunIsSharded:         AlwaysShardOnRun,
	})

	return New(
		ctx,
		WithShardedClient(s),
		WithPauseDeleter(NewPauseStore(u)),
	)
}

func (c Config) ConnectOpts() (rueidis.ClientOption, error) {
	opts := rueidis.ClientOption{
		InitAddress: []string{fmt.Sprintf("%s:%d", c.Host, c.Port)},
		ShuffleInit: true,
		SelectDB:    c.DB,
		Username:    c.Username,
		Password:    c.Password,
	}
	return opts, nil
}

// Opt represents an option to use when creating a redis-backed state store.
type Opt func(r *mgr)

// New returns a state manager which uses Redis as the backing state store.
//
// By default, this connects to a local Redis server.  Use WithConnectOpts to
// change how we connect to Redis.
func New(ctx context.Context, opts ...Opt) (state.Manager, error) {
	m := &mgr{}

	for _, opt := range opts {
		opt(m)
	}

	m.shardedMgr = shardedMgr{
		s: m.unsafeShardedClientDoNotUse,
	}

	return m, nil
}

// WithShardedClient uses an already connected redis client.
func WithShardedClient(s *ShardedClient) Opt {
	return func(m *mgr) {
		m.unsafeShardedClientDoNotUse = s
	}
}

// WithPauseDeleter adds a pause deletion handler that deletes pauses when runs are deleted.
func WithPauseDeleter(d state.PauseDeleter) Opt {
	return func(m *mgr) {
		m.pauseDeleter = d
	}
}

type mgr struct {
	// unsafe: Operate on sharded manager instead.
	unsafeShardedClientDoNotUse *ShardedClient

	pauseDeleter state.PauseDeleter

	shardedMgr
}

type shardedMgr struct {
	s *ShardedClient
}

type CompositePauseID struct {
	PauseID uuid.UUID
	RunID   *ulid.ULID
}

func (m shardedMgr) New(ctx context.Context, input state.Input) (state.State, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "New"), redis_telemetry.ScopeFnRunState)

	fnRunState := m.s.FunctionRunState()
	client, isSharded := fnRunState.Client(ctx, input.Identifier.AccountID, input.Identifier.RunID)

	// Firstly, check idempotency here.
	//
	// NOTE: We have to do this out of the new transaction as state is sharded by run ID.  this
	// does NOT work for idempotency keys which must be sharded by account ID.  unfortunately,
	// mixing the two leads to cross-slot queries, which fail hard.  in this case, we reduce
	// atomicity to improve idempotency.
	//
	// In future/other metadata stores this is (or will be) transactional.
	//
	{
		key := fnRunState.kg.Idempotency(ctx, isSharded, input.Identifier)
		runID, err := m.idempotencyCheck(ctx, client, key, input.Identifier)
		switch err {
		case nil: // no-op
		// NOTE:
		// This will happen as part of the transition of storing empty strings for idempotency
		// key to ULID values.
		// So if this error is returned, we should just continue with creating a new state, since
		// it could mean that the state is not actually created.
		case state.ErrInvalidIdentifier: // no-op
		default:
			return nil, err
		}

		// If a state already exists with the idempotency key, override the input's runID and continue
		if runID != nil && input.Identifier.RunID != *runID {
			input.Identifier.RunID = *runID
		}
	}

	// We marshal this ahead of creating a redis transaction as it's necessary
	// every time and reduces the duration that the lock is held.
	events, err := json.Marshal(input.EventBatchData)
	if err != nil {
		return nil, err
	}

	var stepsByt []byte
	if len(input.Steps) > 0 {
		stepsByt, err = json.Marshal(input.Steps)
		if err != nil {
			return nil, fmt.Errorf("error storing run state in redis when marshalling steps: %w", err)
		}
	}

	var stepInputsByt []byte
	if len(input.StepInputs) > 0 {
		stepInputsByt, err = json.Marshal(input.StepInputs)
		if err != nil {
			return nil, fmt.Errorf("error storing run state in redis when marshalling step inputs: %w", err)
		}
	}

	metadata := runMetadata{
		Identifier:     input.Identifier,
		Debugger:       input.Debugger,
		Version:        currentVersion,
		RequestVersion: consts.RequestVersionUnknown, // Always use -1 to indicate unset hash version until first request.
		Context:        input.Context,
		Status:         enums.RunStatusScheduled,
		SpanID:         input.SpanID,
		EventSize:      len(events),
		StateSize:      len(events) + len(stepsByt) + len(stepInputsByt),
		StepCount:      len(input.Steps),
	}
	if input.RunType != nil {
		metadata.RunType = *input.RunType
	}

	metadataByt, err := json.Marshal(metadata.Map())
	if err != nil {
		return nil, fmt.Errorf("error storing run state in redis: %w", err)
	}

	args, err := StrSlice([]any{
		events,
		metadataByt,
		stepsByt,
		stepInputsByt,
	})
	if err != nil {
		return nil, err
	}

	status, err := retriableScripts["new"].Exec(
		redis_telemetry.WithScriptName(ctx, "new"),
		client,
		[]string{
			fnRunState.kg.Events(ctx, isSharded, input.Identifier.WorkflowID, input.Identifier.RunID),
			fnRunState.kg.RunMetadata(ctx, isSharded, input.Identifier.RunID),
			fnRunState.kg.Actions(ctx, isSharded, input.Identifier.WorkflowID, input.Identifier.RunID),
			fnRunState.kg.Stack(ctx, isSharded, input.Identifier.RunID),
			fnRunState.kg.ActionInputs(ctx, isSharded, input.Identifier),
		},
		args,
	).AsInt64()
	if err != nil {
		return nil, fmt.Errorf("error storing run state in redis: %w", err)
	}
	switch status {
	case 0: // new
		return state.NewStateInstance(
				input.Identifier,
				metadata.Metadata(),
				input.EventBatchData,
				input.Steps,
				make([]string, 0),
			),
			nil
	case 1: // already exists
		// XXX: Returns a shell of a state with mutated identifier to the existing runID
		// It does not load the existing run state anymore.
		return state.NewStateInstance(
			input.Identifier,
			metadata.Metadata(),
			make([]map[string]any, 0),
			make([]state.MemoizedStep, 0),
			make([]string, 0),
		), state.ErrIdentifierExists

	default:
		return nil, fmt.Errorf("unknown status %d when attempting to create function state", status)
	}
}

// idempotencyCheck checks if the function state already exists, and return the runID of the existing state
// if it does
func (m shardedMgr) idempotencyCheck(ctx context.Context, rc RetriableClient, key string, id state.Identifier) (*ulid.ULID, error) {
	prev, err := rc.Do(ctx, func(c rueidis.Client) rueidis.Completed {
		return c.B().
			Set().
			Key(key).
			Value(id.RunID.String()).
			Nx().
			Get(). // retrieve the previous value if exists
			Ex(consts.FunctionIdempotencyPeriod).
			Build()
	}).ToString()
	if err == rueidis.Nil {
		return nil, nil // no previous state exists, entirely new
	}
	if err != nil {
		return nil, err
	}

	// When a run finishes, we prefix the run ID with the tombstone marker.
	// This is needed for scheduling idempotency:  if scheduling retries the new state op
	// and elsewhere we've updated with the tombstone prefix, scheduling can stop.
	// Realisitcally, the chances of this are low, as the entire run has to finish while
	// the scheduling op retries.
	if len(prev) > 0 && prev[0] == consts.FunctionIdempotencyTombstone {
		return nil, state.ErrIdentifierTombstone
	}

	// if there are existing values, the state might have already been created
	runID, err := ulid.Parse(prev)
	if err != nil {
		// there already is a value but is not a valid ULID
		return nil, state.ErrInvalidIdentifier
	}

	return &runID, nil
}

func (m shardedMgr) UpdateMetadata(ctx context.Context, accountID uuid.UUID, runID ulid.ULID, md state.MetadataUpdate) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "UpdateMetadata"), redis_telemetry.ScopeFnRunState)

	input := []string{
		"0", // Force planning / disable immediate execution
		strconv.Itoa(consts.RequestVersionUnknown), // Request version
		"0", // start time default value
		"0", // has AI default value
	}
	if md.DisableImmediateExecution {
		input[0] = "1"
	}
	if md.RequestVersion != consts.RequestVersionUnknown {
		input[1] = strconv.Itoa(md.RequestVersion)
	}
	if !md.StartedAt.IsZero() {
		input[2] = strconv.FormatInt(md.StartedAt.UnixMilli(), 10)
	}
	if md.HasAI {
		input[3] = "1"
	}

	fnRunState := m.s.FunctionRunState()
	client, isSharded := fnRunState.Client(ctx, accountID, runID)

	status, err := retriableScripts["updateMetadata"].Exec(
		redis_telemetry.WithScriptName(ctx, "updateMetadata"),
		client,
		[]string{
			fnRunState.kg.RunMetadata(ctx, isSharded, runID),
		},
		input,
	).AsInt64()
	if err != nil {
		return err
	}
	if status != 0 {
		return fmt.Errorf("unknown response updating metadata: %w", err)
	}
	return nil
}

func (m shardedMgr) Exists(ctx context.Context, accountId uuid.UUID, runID ulid.ULID) (bool, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "Exists"), redis_telemetry.ScopeFnRunState)

	fnRunState := m.s.FunctionRunState()
	r, isSharded := fnRunState.Client(ctx, accountId, runID)
	return r.Do(ctx, func(client rueidis.Client) rueidis.Completed {
		return client.B().Exists().Key(fnRunState.kg.RunMetadata(ctx, isSharded, runID)).Build()
	}).AsBool()
}

func (m shardedMgr) metadata(ctx context.Context, accountId uuid.UUID, runID ulid.ULID) (*runMetadata, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "metadata"), redis_telemetry.ScopeFnRunState)

	fnRunState := m.s.FunctionRunState()
	r, isSharded := fnRunState.Client(ctx, accountId, runID)
	val, err := r.Do(ctx, func(client rueidis.Client) rueidis.Completed {
		return client.B().Hgetall().Key(fnRunState.kg.RunMetadata(ctx, isSharded, runID)).Build()
	}).AsStrMap()
	if err != nil {
		return nil, err
	}
	return newRunMetadata(val)
}

func (m shardedMgr) Cancel(ctx context.Context, id state.Identifier) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "Cancel"), redis_telemetry.ScopeFnRunState)

	fnRunState := m.s.FunctionRunState()
	r, isSharded := fnRunState.Client(ctx, id.AccountID, id.RunID)
	status, err := retriableScripts["cancel"].Exec(
		redis_telemetry.WithScriptName(ctx, "cancel"),
		r,
		[]string{fnRunState.kg.RunMetadata(ctx, isSharded, id.RunID)},
		[]string{},
	).AsInt64()
	if err != nil && !rueidis.IsRedisNil(err) {
		return fmt.Errorf("error cancelling: %w", err)
	}
	switch status {
	case 0:
		return nil
	case 1:
		return state.ErrFunctionComplete
	case 2:
		return state.ErrFunctionFailed
	case 3:
		return state.ErrFunctionCancelled
	}
	return fmt.Errorf("unknown return value cancelling function: %d", status)
}

func (m shardedMgr) SetStatus(ctx context.Context, id state.Identifier, status enums.RunStatus) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "SetStatus"), redis_telemetry.ScopeFnRunState)

	fnRunState := m.s.FunctionRunState()
	r, isSharded := fnRunState.Client(ctx, id.AccountID, id.RunID)
	args, err := StrSlice([]any{
		int(status),
	})
	if err != nil {
		return err
	}

	_, err = retriableScripts["setStatus"].Exec(
		redis_telemetry.WithScriptName(ctx, "setStatus"),
		r,
		[]string{fnRunState.kg.RunMetadata(ctx, isSharded, id.RunID)},
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error cancelling: %w", err)
	}
	return nil
}

func (m shardedMgr) Metadata(ctx context.Context, accountId uuid.UUID, runID ulid.ULID) (*state.Metadata, error) {
	metadata, err := m.metadata(ctx, accountId, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to load metadata: %w", err)
	}
	meta := metadata.Metadata()
	return &meta, nil
}

func (m shardedMgr) LoadEvents(ctx context.Context, accountId uuid.UUID, fnID uuid.UUID, runID ulid.ULID) ([]json.RawMessage, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "LoadEvents"), redis_telemetry.ScopeFnRunState)

	fnRunState := m.s.FunctionRunState()

	var events []json.RawMessage

	r, isSharded := fnRunState.Client(ctx, accountId, runID)

	byt, err := r.Do(ctx, func(client rueidis.Client) rueidis.Completed {
		return client.B().Get().Key(fnRunState.kg.Events(ctx, isSharded, fnID, runID)).Build()
	}).AsBytes()
	if err != nil {
		if err == rueidis.Nil {
			return nil, state.ErrEventNotFound
		}
		return nil, fmt.Errorf("failed to get event; %w", err)
	}

	if err := json.Unmarshal(byt, &events); err != nil {
		return nil, fmt.Errorf("failed to unmarshal batch; %w", err)
	}
	return events, nil
}

func (m shardedMgr) LoadSteps(ctx context.Context, accountId uuid.UUID, fnID uuid.UUID, runID ulid.ULID) (map[string]json.RawMessage, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "LoadSteps"), redis_telemetry.ScopeFnRunState)

	fnRunState := m.s.FunctionRunState()

	var (
		steps = map[string]json.RawMessage{}
		v1id  = state.Identifier{
			RunID:      runID,
			WorkflowID: fnID,
			AccountID:  accountId,
		}
	)

	r, isSharded := fnRunState.Client(ctx, accountId, runID)

	// Load action inputs
	inputMap, err := r.Do(ctx, func(client rueidis.Client) rueidis.Completed {
		return client.B().Hgetall().Key(fnRunState.kg.ActionInputs(ctx, isSharded, v1id)).Build()
	}).AsStrMap()
	if err != nil {
		return nil, fmt.Errorf("failed loading action inputs; %w", err)
	}
	for stepID, marshalled := range inputMap {
		wrapper := map[string]json.RawMessage{
			"input": json.RawMessage(marshalled),
		}
		wrappedData, err := json.Marshal(wrapper)
		if err != nil {
			return nil, fmt.Errorf("failed to wrap action input for \"%s\"; %w", stepID, err)
		}
		steps[stepID] = wrappedData
	}

	// Load the actions.  This is a map of step IDs to JSON-encoded results.
	rmap, err := r.Do(ctx, func(client rueidis.Client) rueidis.Completed {
		return client.B().Hgetall().Key(fnRunState.kg.Actions(ctx, isSharded, fnID, runID)).Build()
	}).AsStrMap()
	if err != nil {
		return nil, fmt.Errorf("failed loading actions; %w", err)
	}
	for stepID, marshalled := range rmap {
		steps[stepID] = json.RawMessage(marshalled)
	}

	return steps, nil
}

func (m shardedMgr) LoadStepInputs(ctx context.Context, accountId uuid.UUID, fnID uuid.UUID, runID ulid.ULID) (map[string]json.RawMessage, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "LoadStepInputs"), redis_telemetry.ScopeFnRunState)

	fnRunState := m.s.FunctionRunState()

	var (
		steps = map[string]json.RawMessage{}
		v1id  = state.Identifier{
			RunID:      runID,
			WorkflowID: fnID,
			AccountID:  accountId,
		}
	)

	r, isSharded := fnRunState.Client(ctx, accountId, runID)

	// Load action inputs only
	inputMap, err := r.Do(ctx, func(client rueidis.Client) rueidis.Completed {
		return client.B().Hgetall().Key(fnRunState.kg.ActionInputs(ctx, isSharded, v1id)).Build()
	}).AsStrMap()
	if err != nil {
		return nil, fmt.Errorf("failed loading action inputs; %w", err)
	}
	for stepID, marshalled := range inputMap {
		steps[stepID] = json.RawMessage(marshalled)
	}

	return steps, nil
}

func (m shardedMgr) LoadStepsWithIDs(ctx context.Context, accountId uuid.UUID, fnID uuid.UUID, runID ulid.ULID, stepIDs []string) (map[string]json.RawMessage, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "LoadStepsWithIDs"), redis_telemetry.ScopeFnRunState)

	fnRunState := m.s.FunctionRunState()

	steps := map[string]json.RawMessage{}

	r, isSharded := fnRunState.Client(ctx, accountId, runID)

	for _, stepID := range stepIDs {
		result, err := r.Do(ctx, func(client rueidis.Client) rueidis.Completed {
			return client.B().Hget().Key(fnRunState.kg.Actions(ctx, isSharded, fnID, runID)).Field(stepID).Build()
		}).ToString()
		if err != nil && err != rueidis.Nil {
			return nil, fmt.Errorf("failed loading action for step %s; %w", stepID, err)
		}
		if err != rueidis.Nil {
			steps[stepID] = json.RawMessage(result)
		}
	}

	return steps, nil
}

func (m shardedMgr) Load(ctx context.Context, accountId uuid.UUID, runID ulid.ULID) (state.State, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "Load"), redis_telemetry.ScopeFnRunState)

	fnRunState := m.s.FunctionRunState()

	// XXX: Use a pipeliner to improve speed.
	metadata, err := m.metadata(ctx, accountId, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to load metadata; %w", err)
	}

	id := metadata.Identifier

	r, isSharded := fnRunState.Client(ctx, accountId, runID)

	// Load events.
	events := []map[string]any{}

	byt, err := r.Do(ctx, func(client rueidis.Client) rueidis.Completed {
		return client.B().Get().Key(fnRunState.kg.Events(ctx, isSharded, id.WorkflowID, runID)).Build()
	}).AsBytes()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return nil, state.ErrEventNotFound
		}
		return nil, fmt.Errorf("failed to get batch; %w", err)
	}
	if err := json.Unmarshal(byt, &events); err != nil {
		return nil, fmt.Errorf("failed to unmarshal batch; %w", err)
	}

	actions := []state.MemoizedStep{}

	// Load action inputs
	inputMap, err := r.Do(ctx, func(client rueidis.Client) rueidis.Completed {
		return client.B().Hgetall().Key(fnRunState.kg.ActionInputs(ctx, isSharded, id)).Build()
	}).AsStrMap()
	if err != nil {
		return nil, fmt.Errorf("failed loading action inputs; %w", err)
	}
	for stepID, marshalled := range inputMap {
		wrapper := map[string]json.RawMessage{
			"input": json.RawMessage(marshalled),
		}
		wrappedData, err := json.Marshal(wrapper)
		if err != nil {
			return nil, fmt.Errorf("failed to wrap action input for \"%s\"; %w", stepID, err)
		}
		actions = append(actions, state.MemoizedStep{
			ID:   stepID,
			Data: wrappedData,
		})
	}

	// Load the actions
	rmap, err := r.Do(ctx, func(client rueidis.Client) rueidis.Completed {
		return client.B().Hgetall().Key(fnRunState.kg.Actions(ctx, isSharded, id.WorkflowID, runID)).Build()
	}).AsStrMap()
	if err != nil {
		return nil, fmt.Errorf("failed loading actions; %w", err)
	}

	for stepID, marshalled := range rmap {
		var data any
		err = json.Unmarshal([]byte(marshalled), &data)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal step \"%s\" with data \"%s\"; %w", stepID, marshalled, err)
		}
		actions = append(actions, state.MemoizedStep{
			ID:   stepID,
			Data: data,
		})
	}

	meta := metadata.Metadata()

	stack, err := m.stack(ctx, id.AccountID, id.RunID)
	if err != nil {
		return nil, fmt.Errorf("error fetching stack: %w", err)
	}

	return state.NewStateInstance(id, meta, events, actions, stack), nil
}

func (m shardedMgr) stack(ctx context.Context, accountId uuid.UUID, runID ulid.ULID) ([]string, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "stack"), redis_telemetry.ScopeFnRunState)

	fnRunState := m.s.FunctionRunState()

	r, isSharded := fnRunState.Client(ctx, accountId, runID)
	stack, err := r.Do(ctx, func(client rueidis.Client) rueidis.Completed {
		return client.B().Lrange().Key(fnRunState.kg.Stack(ctx, isSharded, runID)).Start(0).Stop(-1).Build()
	}).AsStrSlice()
	if err != nil {
		return nil, fmt.Errorf("error fetching stack: %w", err)
	}
	return stack, nil
}

func (m shardedMgr) SaveResponse(ctx context.Context, i state.Identifier, stepID, marshalledOuptut string) (bool, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "SaveResponse"), redis_telemetry.ScopeFnRunState)

	fnRunState := m.s.FunctionRunState()

	r, isSharded := fnRunState.Client(ctx, i.AccountID, i.RunID)

	keys := []string{
		fnRunState.kg.Actions(ctx, isSharded, i.WorkflowID, i.RunID),
		fnRunState.kg.RunMetadata(ctx, isSharded, i.RunID),
		fnRunState.kg.Stack(ctx, isSharded, i.RunID),
		fnRunState.kg.ActionInputs(ctx, isSharded, i),
		fnRunState.kg.Pending(ctx, isSharded, i),
	}
	args := []string{stepID, marshalledOuptut}

	indexes, err := retriableScripts["saveResponse"].Exec(
		redis_telemetry.WithScriptName(ctx, "saveResponse"),
		r,
		keys,
		args,
	).AsIntSlice()
	if err != nil || len(indexes) == 0 {
		return false, fmt.Errorf("error saving response: %w (response: %v)", err, indexes)
	}
	switch indexes[0] {
	case -1:
		// This is a duplicate response, so we don't need to do anything.
		return false, state.ErrDuplicateResponse
	case -2:
		// This step was already saved with the current data.  Return an idempotent request, and check
		// the second response to see whether we have steps remaining.
		if len(indexes) == 1 {
			return false, state.ErrIdempotentResponse
		}
		return indexes[1] == 1, state.ErrIdempotentResponse
	case 0:
		return false, nil
	case 1:
		return true, nil
	default:
		return false, fmt.Errorf("unknown response saving response: %d", indexes[0])
	}
}

func (m shardedMgr) SavePending(ctx context.Context, i state.Identifier, pending []string) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "SavePending"), redis_telemetry.ScopeFnRunState)

	fnRunState := m.s.FunctionRunState()
	r, isSharded := fnRunState.Client(ctx, i.AccountID, i.RunID)

	byt, err := json.Marshal(pending)
	if err != nil {
		return fmt.Errorf("error marshalling pending steps: %w", err)
	}

	keys := []string{
		fnRunState.kg.Pending(ctx, isSharded, i),
	}

	args := []string{string(byt)}

	_, err = retriableScripts["savePending"].Exec(
		redis_telemetry.WithScriptName(ctx, "savePending"),
		r,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error saving pending: %w", err)
	}
	return nil
}

// Delete deletes state from the state store.  Previously, we would handle this in a
// lifecycle.  Now, state stores must account for deletion directly.  Note that if the
// state store is queue-aware, it must delete queue items for the run also.  This may
// not always be the case.
func (m mgr) Delete(ctx context.Context, i state.Identifier) error {
	err := m.shardedMgr.delete(ctx, ctx, i)
	if err != nil {
		return err
	}

	if m.pauseDeleter != nil {
		err = m.pauseDeleter.DeletePausesForRun(ctx, i.RunID, i.WorkspaceID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m shardedMgr) delete(ctx context.Context, callCtx context.Context, i state.Identifier) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "delete"), redis_telemetry.ScopeFnRunState)

	fnRunState := m.s.FunctionRunState()
	r, isSharded := fnRunState.Client(ctx, i.AccountID, i.RunID)

	key := i.Key
	if i.Key == "" {
		if md, err := m.Metadata(ctx, i.AccountID, i.RunID); err == nil {
			key = fnRunState.kg.Idempotency(ctx, isSharded, md.Identifier)
		}
	} else {
		key = fnRunState.kg.Idempotency(ctx, isSharded, i)
	}

	_ = r.Do(callCtx, func(client rueidis.Client) rueidis.Completed {
		// update the idempotency key to the tombstone prefix to indicate this run is done
		// so scheduling retries can detect and stop.
		val := fmt.Sprintf("%s%s", string(consts.FunctionIdempotencyTombstone), i.RunID)
		return client.B().Set().Key(key).Value(val).Xx().Keepttl().Build()
	}).Error()

	// Clear all other data for a job.
	keys := []string{
		fnRunState.kg.Events(ctx, isSharded, i.WorkflowID, i.RunID),
		fnRunState.kg.RunMetadata(ctx, isSharded, i.RunID),
		fnRunState.kg.Actions(ctx, isSharded, i.WorkflowID, i.RunID),
		fnRunState.kg.Stack(ctx, isSharded, i.RunID),
	}

	result := r.Do(callCtx, func(client rueidis.Client) rueidis.Completed {
		return client.B().Del().Key(keys...).Build()
	})

	if err := result.Error(); err != nil {
		return err
	}

	return nil
}

// ConsumePause consumes a pause, writing the consumed data to state.
func (m shardedMgr) ConsumePause(ctx context.Context, p state.Pause, opts state.ConsumePauseOpts) (state.ConsumePauseResult, error) {
	if opts.IdempotencyKey == "" {
		return state.ConsumePauseResult{}, state.ErrConsumePauseKeyMissing
	}

	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "ConsumePause"), redis_telemetry.ScopePauses)

	fnRunState := m.s.FunctionRunState()
	client, isSharded := fnRunState.Client(ctx, p.Identifier.AccountID, p.Identifier.RunID)

	var marshalledData []byte
	if b, ok := opts.Data.([]byte); ok && json.Valid(b) {
		// Already marshalled data we can just use it
		marshalledData = b
	} else {
		var err error
		marshalledData, err = json.Marshal(opts.Data)
		if err != nil {
			return state.ConsumePauseResult{}, fmt.Errorf("cannot marshal data to store in state: %w", err)
		}
	}

	keys := []string{
		fnRunState.kg.Actions(ctx, isSharded, p.Identifier.FunctionID, p.Identifier.RunID),
		fnRunState.kg.Stack(ctx, isSharded, p.Identifier.RunID),
		fnRunState.kg.RunMetadata(ctx, isSharded, p.Identifier.RunID),
		fnRunState.kg.Pending(ctx, isSharded, state.Identifier{
			RunID:      p.Identifier.RunID,
			WorkflowID: p.Identifier.FunctionID,
		}),
		fnRunState.kg.PauseConsumeKey(ctx, isSharded, p.Identifier.RunID, p.ID),
	}

	args, err := StrSlice([]any{
		p.DataKey,
		string(marshalledData),
		opts.IdempotencyKey,
		time.Now().Add(consts.FunctionIdempotencyPeriod).Unix(),
	})
	if err != nil {
		return state.ConsumePauseResult{}, err
	}

	status, err := retriableScripts["consumePause"].Exec(
		redis_telemetry.WithScriptName(ctx, "consumePause"),
		client,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return state.ConsumePauseResult{}, fmt.Errorf("error consuming pause: %w", err)
	}

	switch status {
	case -1:
		// This could be an ErrDuplicateResponse;  we're attempting to consume a pause twice.
		return state.ConsumePauseResult{}, nil
	case 0:
		return state.ConsumePauseResult{DidConsume: true}, nil
	case 1:
		return state.ConsumePauseResult{DidConsume: true, HasPendingSteps: true}, nil
	default:
		return state.ConsumePauseResult{}, fmt.Errorf("unknown response leasing pause: %d", status)
	}
}

type bufIter struct {
	r     rueidis.Client
	items []string
	idx   int64

	val *state.Pause
	err error

	l              sync.Mutex
	aggregateStart time.Time
}

func (i *bufIter) init(ctx context.Context, key string) error {
	var err error
	// If there are less than a thousand items, query the keys
	// for iteration.
	cmd := i.r.B().Hvals().Key(key).Build()
	i.items, err = i.r.Do(ctx, cmd).AsStrSlice()
	i.l = sync.Mutex{}
	return err
}

func (i *bufIter) Count() int {
	return len(i.items)
}

func (i *bufIter) Index() int64 {
	return i.idx
}

func (i *bufIter) Next(ctx context.Context) bool {
	i.l.Lock()
	defer i.l.Unlock()

	if len(i.items) == 0 {
		i.err = context.Canceled
		if !i.aggregateStart.IsZero() {
			dur := time.Since(i.aggregateStart).Milliseconds()
			metrics.HistogramAggregatePausesLoadDuration(ctx, dur, metrics.HistogramOpt{
				PkgName: pkgName,
				// TODO: tag workspace ID eventually??
				Tags: map[string]any{
					"iterator": "buffer",
				},
			})
		}
		return false
	}

	pause := &state.Pause{}
	i.err = json.Unmarshal([]byte(i.items[0]), pause)
	i.val = pause
	// Remove one from the slice.
	i.items = i.items[1:]
	i.idx++
	return i.err == nil
}

// Buffer by running an MGET to get the values of the pauses.
func (i *bufIter) Val(ctx context.Context) *state.Pause {
	return i.val
}

func (i *bufIter) Error() error {
	return i.err
}

var errScanDone = fmt.Errorf("scan done")

type scanIter struct {
	r   rueidis.Client
	key string
	// chunk is the size of scans to load in one.
	chunk int64

	// count is the cached number of items to return in Count(),
	// ie the hlen result when creating the iterator.
	count int64

	// iterator fields
	i      int
	cursor int
	idx    int64
	vals   rueidis.ScanEntry
	err    error

	l sync.Mutex

	aggregateStart time.Time
}

func (i *scanIter) Error() error {
	return i.err
}

func (i *scanIter) init(ctx context.Context, key string, chunk int64) error {
	i.key = key
	i.chunk = chunk
	cmd := i.r.B().Hscan().Key(key).Cursor(0).Count(i.chunk).Build()
	scan, err := i.r.Do(ctx, cmd).AsScanEntry()
	if err != nil {
		i.err = err
		return err
	}
	i.cursor = int(scan.Cursor)
	i.vals = scan
	i.i = -1
	i.l = sync.Mutex{}
	return nil
}

func (i *scanIter) Count() int {
	return int(i.count)
}

func (i *scanIter) Index() int64 {
	return i.idx
}

func (i *scanIter) fetch(ctx context.Context) error {
	if i.cursor == 0 {
		// We're done, no need to fetch.
		return errScanDone
	}

	// Reset the index.
	i.i = -1

	// Scan 100 times up until there are values
	for scans := 0; scans < 100; scans++ {
		cmd := i.r.B().Hscan().
			Key(i.key).
			Cursor(uint64(i.cursor)).
			Count(i.chunk).
			Build()

		scan, err := i.r.Do(ctx, cmd).AsScanEntry()
		if err != nil {
			return err
		}

		i.cursor = int(scan.Cursor)
		i.vals = scan

		if len(i.vals.Elements) > 0 {
			return nil
		}

		// Prevent starting a new iteration, otherwise we risk an infinite loop if the data isn't changing
		// and we get an empty scan with a 0 cursor which is actually possible in Redis.
		if i.cursor == 0 {
			return errScanDone
		}
	}

	return fmt.Errorf("Scanned max times without finding pause values")
}

func (i *scanIter) Next(ctx context.Context) bool {
	i.l.Lock()
	defer i.l.Unlock()

	if i.i >= (len(i.vals.Elements) - 1) {
		err := i.fetch(ctx)
		if err == errScanDone {
			// No more present.
			i.err = context.Canceled
			if !i.aggregateStart.IsZero() {
				dur := time.Since(i.aggregateStart).Milliseconds()
				metrics.HistogramAggregatePausesLoadDuration(ctx, dur, metrics.HistogramOpt{
					PkgName: pkgName,
					// TODO: tag workspace ID eventually??
					Tags: map[string]any{
						"iterator": "scan",
					},
				})
			}
			return false
		}
		if err != nil {
			i.err = err
			// Stop iterating, set error.
			return false
		}
	}
	// Skip the ID
	i.i++
	// Get the value.
	i.i++
	i.idx++
	return true
}

func (i *scanIter) Val(ctx context.Context) *state.Pause {
	if i.i == -1 || i.i >= len(i.vals.Elements) {
		return nil
	}

	val := i.vals.Elements[i.i]
	if val == "" {
		return nil
	}

	pause := &state.Pause{}
	err := json.Unmarshal([]byte(val), pause)
	if err != nil {
		return nil
	}
	return pause
}

func newRunMetadata(data map[string]string) (*runMetadata, error) {
	var err error
	m := &runMetadata{}

	// The V1 state identifier is the most important thing to be stored in state.  We must have this
	// as it contains tenant information.
	val, ok := data["id"]
	if !ok || val == "" {
		return nil, state.ErrRunNotFound
	}
	id := state.Identifier{}
	if err := json.Unmarshal([]byte(val), &id); err != nil {
		return nil, fmt.Errorf("unable to unmarshal metadata identifier: %s", val)
	}
	m.Identifier = id

	// Handle everything else optimistically
	v, ok := data["status"]
	if !ok {
		return nil, fmt.Errorf("no status stored in metadata")
	}
	status, err := strconv.Atoi(strings.TrimSuffix(v, ".0"))
	if err != nil {
		return nil, fmt.Errorf("invalid function status stored in run metadata: %#v", v)
	}
	m.Status = enums.RunStatus(status)

	parseInt := func(v string) (int, error) {
		str, ok := data[v]
		if !ok {
			return 0, fmt.Errorf("no '%s' stored in run metadata", v)
		}
		val, err := strconv.Atoi(strings.TrimSuffix(str, ".0"))
		if err != nil {
			return 0, fmt.Errorf("invalid '%s' stored in run metadata", v)
		}
		return val, nil
	}

	m.StateSize, _ = parseInt("state_size")
	m.EventSize, _ = parseInt("event_size")
	m.StepCount, _ = parseInt("step_count")

	if val, ok := data["version"]; ok && val != "" {
		v, err := strconv.Atoi(strings.TrimSuffix(val, ".0"))
		if err != nil {
			return nil, fmt.Errorf("invalid metadata version detected: %#v", val)
		}

		m.Version = v
	}

	if val, ok := data["rv"]; ok && val != "" {
		v, err := strconv.Atoi(strings.TrimSuffix(val, ".0"))
		if err != nil {
			return nil, fmt.Errorf("invalid hash version detected: %#v", val)
		}
		m.RequestVersion = v
	}

	if val, ok := data["sat"]; ok && val != "" {
		v, err := strconv.ParseInt(strings.TrimSuffix(val, ".0"), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid started at timestamp detected: %#v", val)
		}
		m.StartedAt = v
	}

	// The below fields are optional
	if val, ok := data["debugger"]; ok {
		if val == "true" || val == "1" {
			m.Debugger = true
		}
	}
	if val, ok := data["runType"]; ok {
		m.RunType = val
	}
	if val, ok := data["id"]; ok && val != "" {
		id := state.Identifier{}
		if err := json.Unmarshal([]byte(val), &id); err != nil {
			return nil, fmt.Errorf("unable to unmarshal metadata identifier: %s", val)
		}
		m.Identifier = id
	}
	if val, ok := data["ctx"]; ok && val != "" {
		ctx := map[string]any{}
		if err := json.Unmarshal([]byte(val), &ctx); err != nil {
			return nil, fmt.Errorf("unable to unmarshal metadata context: %s", val)
		}
		m.Context = ctx
	}
	if val, ok := data["die"]; ok {
		if val == "true" || val == "1" {
			m.DisableImmediateExecution = true
		}
	}

	if val, ok := data["hasAI"]; ok {
		if val == "true" || val == "1" {
			m.HasAI = true
		}
	}

	if val, ok := data["sid"]; ok {
		m.SpanID = val
	}

	return m, nil
}

// keyIter loads all pauses in batches given a list of IDs
type keyIter struct {
	r  rueidis.Client
	kf PauseKeyGenerator
	// chunk is the size of scans to load in one.
	chunk int64
	// keys stores pause IDs to fetch in batches
	keys []string
	// vals stores pauses as strings from MGET
	vals []string

	// scores stores pause creation times or index scores
	// they are conditionally used so the iterator works
	// just fine if it's empty
	scores []float64

	hasScores bool

	idx   int64
	err   error
	start time.Time
}

func (i *keyIter) Error() error {
	return i.err
}

func (i *keyIter) init(ctx context.Context, keys []string, scores []float64, chunk int64) error {
	i.keys = keys
	i.chunk = chunk
	i.scores = scores
	i.hasScores = len(scores) == len(keys)
	err := i.fetch(ctx)
	if err == errScanDone {
		return nil
	}
	return err
}

func (i *keyIter) Count() int {
	return len(i.keys)
}

func (i *keyIter) Index() int64 {
	return i.idx
}

func (i *keyIter) fetch(ctx context.Context) error {
	if len(i.keys) == 0 {
		// No more present.
		i.err = context.Canceled
		dur := time.Since(i.start).Milliseconds()
		metrics.HistogramAggregatePausesLoadDuration(ctx, dur, metrics.HistogramOpt{
			PkgName: pkgName,
			// TODO: tag workspace ID eventually??
			Tags: map[string]any{
				"iterator": "key",
			},
		})
		return errScanDone
	}

	var load []string
	if len(i.keys) > int(i.chunk) {
		load = i.keys[0:i.chunk]
		i.keys = i.keys[i.chunk:]
	} else {
		load = i.keys[:]
		i.keys = []string{}
	}

	for n, id := range load {
		load[n] = i.kf.Pause(ctx, uuid.MustParse(id))
	}

	cmd := i.r.B().Mget().Key(load...).Build()
	i.vals, i.err = i.r.Do(ctx, cmd).AsStrSlice()
	if rueidis.IsRedisNil(i.err) {
		// Somehow none of these pauses no longer exist, which is okay:
		// another concurrent thread may have already consumed it.
		i.err = nil
	}
	return i.err
}

func (i *keyIter) Next(ctx context.Context) bool {
	if len(i.vals) > 0 {
		return true
	}

	err := i.fetch(ctx)
	if err == errScanDone {
		return false
	}
	return err == nil
}

func (i *keyIter) Val(ctx context.Context) *state.Pause {
	var score float64
	if len(i.vals) == 0 {
		return nil
	}

	val := i.vals[0]
	i.vals = i.vals[1:]
	if i.hasScores {
		score = i.scores[0]
		i.scores = i.scores[1:]
	}
	if val == "" {
		return nil
	}

	pause := &state.Pause{}
	err := json.Unmarshal([]byte(val), pause)
	if err != nil {
		return nil
	}

	// Hack for older pauses that don't have a createdAt
	// persisted in the pause item.
	if i.hasScores && pause.CreatedAt.IsZero() {
		pause.CreatedAt = time.Unix(int64(score), 0)
	}

	return pause
}

// runMetadata is stored for each invocation of a function.  This is inserted when
// creating a new run, and stores the triggering event as well as workflow-specific
// metadata for the invocation.
type runMetadata struct {
	Identifier state.Identifier `json:"id"`
	Status     enums.RunStatus  `json:"status"`
	// These are the fields for standard state metadata.
	StateSize                 int            `json:"state_size"`
	EventSize                 int            `json:"event_size"`
	StepCount                 int            `json:"step_count"`
	Debugger                  bool           `json:"debugger"`
	RunType                   string         `json:"runType,omitempty"`
	ReplayID                  string         `json:"rID,omitempty"`
	Version                   int            `json:"version"`
	RequestVersion            int            `json:"rv"`
	Context                   map[string]any `json:"ctx,omitempty"`
	DisableImmediateExecution bool           `json:"die,omitempty"`
	SpanID                    string         `json:"sid"`
	StartedAt                 int64          `json:"sat,omitempty"`
	HasAI                     bool           `json:"hasAI,omitempty"`
}

func (r runMetadata) Map() map[string]any {
	return map[string]any{
		"id":       r.Identifier,
		"status":   int(r.Status), // Always store this as an int
		"debugger": r.Debugger,
		"runType":  r.RunType,
		"version":  r.Version,
		"rv":       r.RequestVersion,
		"ctx":      r.Context,
		"die":      r.DisableImmediateExecution,
		"sid":      r.SpanID,
		"sat":      r.StartedAt,
		"hasAI":    r.HasAI,
	}
}

func (r runMetadata) Metadata() state.Metadata {
	m := state.Metadata{
		Identifier:                r.Identifier,
		Debugger:                  r.Debugger,
		Status:                    r.Status,
		Version:                   r.Version,
		RequestVersion:            r.RequestVersion,
		Context:                   r.Context,
		DisableImmediateExecution: r.DisableImmediateExecution,
		SpanID:                    r.SpanID,
		HasAI:                     r.HasAI,
	}
	// 0 != time.IsZero
	// only convert to time if runMetadata's StartedAt is > 0
	if r.StartedAt > 0 {
		m.StartedAt = time.UnixMilli(r.StartedAt)
	}

	if r.RunType != "" {
		m.RunType = &r.RunType
	}
	return m
}

func StrSlice(args []any) ([]string, error) {
	res := make([]string, len(args))
	for i, item := range args {
		if s, ok := item.(fmt.Stringer); ok {
			res[i] = s.String()
			continue
		}

		switch v := item.(type) {
		case string:
			res[i] = v
		case []byte:
			res[i] = rueidis.BinaryString(v)
		case int:
			res[i] = strconv.Itoa(v)
		case bool:
			// Use 1 and 0 to signify true/false.
			if v {
				res[i] = "1"
			} else {
				res[i] = "0"
			}
		default:
			byt, err := json.Marshal(item)
			if err != nil {
				return nil, err
			}
			res[i] = rueidis.BinaryString(byt)
		}
	}
	return res, nil
}
