package redis_state

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
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
	"github.com/inngest/expr"
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

	return New(
		ctx,
		WithUnshardedClient(u),
		WithShardedClient(NewShardedClient(ShardedClientOpts{
			UnshardedClient:        u,
			FunctionRunStateClient: r,
			BatchClient:            r,
			StateDefaultKey:        StateDefaultKey,
			QueueDefaultKey:        QueueDefaultKey,
			FnRunIsSharded:         AlwaysShardOnRun,
		})),
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

	m.unshardedMgr = unshardedMgr{
		u: m.unsafeUnshardedClientDoNotUse,
	}

	return m, nil
}

// WithShardedClient uses an already connected redis client.
func WithShardedClient(s *ShardedClient) Opt {
	return func(m *mgr) {
		m.unsafeShardedClientDoNotUse = s
	}
}

// WithUnshardedClient uses an already connected redis client.
func WithUnshardedClient(u *UnshardedClient) Opt {
	return func(m *mgr) {
		m.unsafeUnshardedClientDoNotUse = u
	}
}

type mgr struct {
	// unsafe: Operate on sharded manager instead.
	unsafeShardedClientDoNotUse *ShardedClient

	// unsafe: Operate on unsharded manager instead.
	unsafeUnshardedClientDoNotUse *UnshardedClient

	shardedMgr
	unshardedMgr
}

type shardedMgr struct {
	s *ShardedClient
}

type unshardedMgr struct {
	u *UnshardedClient
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
	res := client.Do(ctx, func(c rueidis.Client) rueidis.Completed {
		return c.B().Set().Key(
			fnRunState.kg.Idempotency(ctx, isSharded, input.Identifier),
		).Value("").Nx().Ex(consts.FunctionIdempotencyPeriod).Build()
	})
	set, err := res.AsBool()
	if (err == nil || rueidis.IsRedisNil(err)) && !set {
		return nil, state.ErrIdentifierExists
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
			fnRunState.kg.Events(ctx, isSharded, input.Identifier),
			fnRunState.kg.RunMetadata(ctx, isSharded, input.Identifier.RunID),
			fnRunState.kg.Actions(ctx, isSharded, input.Identifier),
			fnRunState.kg.Stack(ctx, isSharded, input.Identifier.RunID),
			fnRunState.kg.ActionInputs(ctx, isSharded, input.Identifier),
		},
		args,
	).AsInt64()

	if err != nil {
		return nil, fmt.Errorf("error storing run state in redis: %w", err)
	}
	if status == 1 {
		return nil, state.ErrIdentifierExists
	}

	return state.NewStateInstance(
			input.Identifier,
			metadata.Metadata(),
			input.EventBatchData,
			input.Steps,
			make([]string, 0),
		),
		nil
}

func (m shardedMgr) UpdateMetadata(ctx context.Context, accountID uuid.UUID, runID ulid.ULID, md state.MetadataUpdate) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "UpdateMetadata"), redis_telemetry.ScopeFnRunState)

	input := []string{
		"0", // Force planning / disable immediate execution
		strconv.Itoa(consts.RequestVersionUnknown), // Request version
		"0", // start time default value
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

func (m shardedMgr) IsComplete(ctx context.Context, accountId uuid.UUID, runID ulid.ULID) (bool, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "IsComplete"), redis_telemetry.ScopeFnRunState)

	fnRunState := m.s.FunctionRunState()

	r, isSharded := fnRunState.Client(ctx, accountId, runID)

	val, err := r.Do(ctx, func(client rueidis.Client) rueidis.Completed {
		return client.B().Hget().Key(fnRunState.kg.RunMetadata(ctx, isSharded, runID)).Field("status").Build()
	}).AsBytes()
	if err != nil {
		return false, err
	}
	return !bytes.Equal(val, []byte("0")), nil
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

	var (
		events []json.RawMessage
		v1id   = state.Identifier{
			RunID:      runID,
			WorkflowID: fnID,
			AccountID:  accountId,
		}
	)

	r, isSharded := fnRunState.Client(ctx, accountId, runID)

	byt, err := r.Do(ctx, func(client rueidis.Client) rueidis.Completed {
		return client.B().Get().Key(fnRunState.kg.Events(ctx, isSharded, v1id)).Build()
	}).AsBytes()
	if err == nil {
		if err := json.Unmarshal(byt, &events); err != nil {
			return nil, fmt.Errorf("failed to unmarshal batch; %w", err)
		}
		return events, nil
	}

	// Pre-batch days for backcompat.
	byt, err = r.Do(ctx, func(client rueidis.Client) rueidis.Completed {
		return client.B().Get().Key(fnRunState.kg.Event(ctx, isSharded, v1id)).Build()
	}).AsBytes()
	if err != nil {
		if err == rueidis.Nil {
			return nil, state.ErrEventNotFound
		}
		return nil, fmt.Errorf("failed to get event; %w", err)
	}
	return []json.RawMessage{byt}, nil
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
		return client.B().Hgetall().Key(fnRunState.kg.Actions(ctx, isSharded, v1id)).Build()
	}).AsStrMap()
	if err != nil {
		return nil, fmt.Errorf("failed loading actions; %w", err)
	}
	for stepID, marshalled := range rmap {
		steps[stepID] = json.RawMessage(marshalled)
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
	switch metadata.Version {
	case 0: // pre-batch days
		byt, err := r.Do(ctx, func(client rueidis.Client) rueidis.Completed {
			return client.B().Get().Key(fnRunState.kg.Event(ctx, isSharded, id)).Build()
		}).AsBytes()
		if err != nil {
			if err == rueidis.Nil {
				return nil, state.ErrEventNotFound
			}
			return nil, fmt.Errorf("failed to get event; %w", err)
		}
		event := map[string]any{}
		if err := json.Unmarshal(byt, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal event; %w", err)
		}
		events = []map[string]any{event}
	default: // current default is 1
		// Load the batch of events
		byt, err := r.Do(ctx, func(client rueidis.Client) rueidis.Completed {
			return client.B().Get().Key(fnRunState.kg.Events(ctx, isSharded, id)).Build()
		}).AsBytes()
		if err != nil {
			return nil, fmt.Errorf("failed to get batch; %w", err)
		}
		if err := json.Unmarshal(byt, &events); err != nil {
			return nil, fmt.Errorf("failed to unmarshal batch; %w", err)
		}
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
		return client.B().Hgetall().Key(fnRunState.kg.Actions(ctx, isSharded, id)).Build()
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

func (m shardedMgr) StackIndex(ctx context.Context, accountId uuid.UUID, runID ulid.ULID, stepID string) (int, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "StackIndex"), redis_telemetry.ScopeFnRunState)

	fnRunState := m.s.FunctionRunState()

	r, isSharded := fnRunState.Client(ctx, accountId, runID)
	stack, err := r.Do(ctx, func(client rueidis.Client) rueidis.Completed {
		return client.B().Lrange().Key(fnRunState.kg.Stack(ctx, isSharded, runID)).Start(0).Stop(-1).Build()
	}).AsStrSlice()
	if err != nil {
		return 0, err
	}
	if len(stack) == 0 {
		return 0, nil
	}
	for n, i := range stack {
		if i == stepID {
			return n + 1, nil
		}

	}
	return 0, fmt.Errorf("step not found in stack: %s", stepID)
}

func (m shardedMgr) SaveResponse(ctx context.Context, i state.Identifier, stepID, marshalledOuptut string) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "SaveResponse"), redis_telemetry.ScopeFnRunState)

	fnRunState := m.s.FunctionRunState()

	r, isSharded := fnRunState.Client(ctx, i.AccountID, i.RunID)

	keys := []string{
		fnRunState.kg.Actions(ctx, isSharded, i),
		fnRunState.kg.RunMetadata(ctx, isSharded, i.RunID),
		fnRunState.kg.Stack(ctx, isSharded, i.RunID),
		fnRunState.kg.ActionInputs(ctx, isSharded, i),
	}
	args := []string{stepID, marshalledOuptut}

	index, err := retriableScripts["saveResponse"].Exec(
		redis_telemetry.WithScriptName(ctx, "saveResponse"),
		r,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error saving response: %w", err)
	}
	if index == -1 {
		// This is a duplicate response, so we don't need to do anything.
		return state.ErrDuplicateResponse
	}
	return nil
}

func (m unshardedMgr) SavePause(ctx context.Context, p state.Pause) error {
	packed, err := json.Marshal(p)
	if err != nil {
		return err
	}

	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "SavePause"), redis_telemetry.ScopePauses)

	// `evt` is used to search for pauses based on event names. We only want to
	// do this if this pause is not part of an invoke. If it is, we don't want
	// to index it by event name as the pause will be processed by correlation
	// ID.
	evt := ""
	if p.Event != nil && (p.InvokeCorrelationID == nil || *p.InvokeCorrelationID == "") {
		evt = *p.Event
	}

	corrId := ""
	if p.InvokeCorrelationID != nil {
		corrId = *p.InvokeCorrelationID
	}

	extendedExpiry := time.Until(p.Expires.Time().Add(10 * time.Minute)).Seconds()
	nowUnixSeconds := time.Now().Unix()

	pause := m.u.Pauses()

	// Warning: We need to access global keys, which must be colocated on the same Redis cluster
	global := m.u.Global()

	keys := []string{
		pause.kg.Pause(ctx, p.ID),
		pause.kg.PauseEvent(ctx, p.WorkspaceID, evt),
		global.kg.Invoke(ctx, p.WorkspaceID),
		pause.kg.PauseIndex(ctx, "add", p.WorkspaceID, evt),
		pause.kg.PauseIndex(ctx, "exp", p.WorkspaceID, evt),
		pause.kg.RunPauses(ctx, p.Identifier.RunID),
	}

	args, err := StrSlice([]any{
		string(packed),
		p.ID.String(),
		evt,
		corrId,
		// Add at least 10 minutes to this pause, allowing us to process the
		// pause by ID for 10 minutes past expiry.
		int(extendedExpiry),
		nowUnixSeconds,
	})
	if err != nil {
		return err
	}

	status, err := scripts["savePause"].Exec(
		redis_telemetry.WithScriptName(ctx, "savePause"),
		pause.Client(),
		keys,
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error finalizing: %w", err)
	}

	switch status {
	case 0:
		return nil
	case 1:
		return state.ErrPauseAlreadyExists
	}
	return fmt.Errorf("unknown response saving pause: %d", status)
}

func (m unshardedMgr) LeasePause(ctx context.Context, id uuid.UUID) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "LeasePause"), redis_telemetry.ScopePauses)

	args, err := StrSlice([]any{
		time.Now().UnixMilli(),
		state.PauseLeaseDuration.Seconds(),
	})
	if err != nil {
		return err
	}

	pause := m.u.Pauses()

	status, err := scripts["leasePause"].Exec(
		redis_telemetry.WithScriptName(ctx, "leasePause"),
		pause.Client(),
		// keys will be sharded/unsharded depending on RunID
		[]string{pause.kg.Pause(ctx, id), pause.kg.PauseLease(ctx, id)},
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error leasing pause: %w", err)
	}
	switch status {
	case 0:
		return nil
	case 1:
		return state.ErrPauseLeased
	case 2:
		return state.ErrPauseNotFound
	default:
		return fmt.Errorf("unknown response leasing pause: %d", status)
	}
}

// Delete deletes state from the state store.  Previously, we would handle this in a
// lifecycle.  Now, state stores must account for deletion directly.  Note that if the
// state store is queue-aware, it must delete queue items for the run also.  This may
// not always be the case.
//
// Returns a boolean indicating whether it performed deletion. If the run had
// parallel steps then it may be false, since parallel steps cause the function
// end to be reached multiple times in a single run
func (m mgr) Delete(ctx context.Context, i state.Identifier) (bool, error) {
	// Ensure this context isn't cancelled;  this is called in a goroutine.
	callCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	performedDeletion, err := m.shardedMgr.delete(ctx, callCtx, i)
	if err != nil {
		return false, err
	}

	err = m.deletePausesForRun(ctx, callCtx, i)
	if err != nil {
		return false, err
	}

	return performedDeletion, nil
}

func (m shardedMgr) delete(ctx context.Context, callCtx context.Context, i state.Identifier) (bool, error) {
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

	if err := r.Do(callCtx, func(client rueidis.Client) rueidis.Completed {
		return client.B().Expire().Key(key).Seconds(int64(consts.FunctionIdempotencyPeriod.Seconds())).Build()
	}).Error(); err != nil {
		return false, err
	}

	// Clear all other data for a job.
	keys := []string{
		fnRunState.kg.Actions(ctx, isSharded, i),
		fnRunState.kg.RunMetadata(ctx, isSharded, i.RunID),
		fnRunState.kg.Events(ctx, isSharded, i),
		fnRunState.kg.Stack(ctx, isSharded, i.RunID),

		// XXX: remove these in a state store refactor.
		fnRunState.kg.Event(ctx, isSharded, i),
		fnRunState.kg.History(ctx, isSharded, i.RunID),
		fnRunState.kg.Errors(ctx, isSharded, i),
	}

	performedDeletion := false
	for _, k := range keys {
		result := r.Do(callCtx, func(client rueidis.Client) rueidis.Completed {
			return client.B().Del().Key(k).Build()
		})

		// We should check a single key rather than all keys, to avoid races.
		// We'll somewhat arbitrarily pick RunMetadata
		if k == fnRunState.kg.RunMetadata(ctx, isSharded, i.RunID) {
			if count, _ := result.ToInt64(); count > 0 {
				performedDeletion = true
			}
		}

		if err := result.Error(); err != nil {
			return false, err
		}
	}

	return performedDeletion, nil
}

func (m unshardedMgr) deletePausesForRun(ctx context.Context, callCtx context.Context, i state.Identifier) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "deletePausesForRun"), redis_telemetry.ScopePauses)

	pause := m.u.Pauses()

	// Fetch all pauses for the run
	if pauseIDs, err := pause.Client().Do(callCtx, pause.Client().B().Smembers().Key(pause.kg.RunPauses(ctx, i.RunID)).Build()).AsStrSlice(); err == nil {
		for _, id := range pauseIDs {
			pauseID, _ := uuid.Parse(id)
			_ = m.DeletePauseByID(ctx, pauseID)
		}
	}

	return pause.Client().Do(callCtx, pause.Client().B().Del().Key(pause.kg.RunPauses(ctx, i.RunID)).Build()).Error()
}

func (m unshardedMgr) DeletePauseByID(ctx context.Context, pauseID uuid.UUID) error {
	// Attempt to fetch this pause.
	pause, err := m.PauseByID(ctx, pauseID)
	if err == nil && pause != nil {
		return m.DeletePause(ctx, *pause)
	}

	// This won't delete event keys nicely, but still gets the pause yeeted.
	return m.DeletePause(ctx, state.Pause{
		ID: pauseID,
	})
}

func (m unshardedMgr) DeletePause(ctx context.Context, p state.Pause) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "DeletePause"), redis_telemetry.ScopePauses)

	pause := m.u.Pauses()

	global := m.u.Global()

	// Add a default event here, which is null and overwritten by everything.  This is necessary
	// to keep the same cluster key.
	eventKey := pause.kg.PauseEvent(ctx, p.WorkspaceID, "-")
	if p.Event != nil {
		eventKey = pause.kg.PauseEvent(ctx, p.WorkspaceID, *p.Event)
	}

	evt := ""
	if p.Event != nil && (p.InvokeCorrelationID == nil || *p.InvokeCorrelationID == "") {
		evt = *p.Event
	}

	corrId := ""
	if p.InvokeCorrelationID != nil && *p.InvokeCorrelationID != "" {
		corrId = *p.InvokeCorrelationID
	}

	pauseKey := pause.kg.Pause(ctx, p.ID)
	pauseStepKey := pause.kg.PauseStep(ctx, p.Identifier, p.Incoming)
	runPausesKey := pause.kg.RunPauses(ctx, p.Identifier.RunID)

	keys := []string{
		pauseKey,
		pauseStepKey,
		eventKey,
		// Warning: We need to access global keys, which must be colocated on the same Redis cluster
		global.kg.Invoke(ctx, p.WorkspaceID),
		pause.kg.PauseIndex(ctx, "add", p.WorkspaceID, evt),
		pause.kg.PauseIndex(ctx, "exp", p.WorkspaceID, evt),
		runPausesKey,
	}

	status, err := scripts["deletePause"].Exec(
		redis_telemetry.WithScriptName(ctx, "deletePause"),
		pause.Client(),
		keys,
		[]string{
			p.ID.String(),
			corrId,
		},
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error deleting pause: %w", err)
	}

	switch status {
	case 0:
		return nil
	default:
		return fmt.Errorf("unknown response deleting pause: %d", status)
	}
}

func (m mgr) ConsumePause(ctx context.Context, pauseID uuid.UUID, data any) error {
	p, err := m.unshardedMgr.PauseByID(ctx, pauseID)
	if err != nil {
		return err
	}

	err = m.shardedMgr.consumePause(ctx, p, data)
	if err != nil {
		return err
	}

	// The pause was now consumed, so let's clean up
	return m.unshardedMgr.DeletePause(ctx, *p)
}

func (m shardedMgr) consumePause(ctx context.Context, p *state.Pause, data any) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "consumePause"), redis_telemetry.ScopePauses)

	fnRunState := m.s.FunctionRunState()
	client, isSharded := fnRunState.Client(ctx, p.Identifier.AccountID, p.Identifier.RunID)

	marshalledData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("cannot marshal data to store in state: %w", err)
	}

	keys := []string{
		fnRunState.kg.Actions(ctx, isSharded, p.Identifier),
		fnRunState.kg.Stack(ctx, isSharded, p.Identifier.RunID),
		fnRunState.kg.RunMetadata(ctx, isSharded, p.Identifier.RunID),
	}

	args, err := StrSlice([]any{
		p.DataKey,
		string(marshalledData),
	})
	if err != nil {
		return err
	}

	status, err := retriableScripts["consumePause"].Exec(
		redis_telemetry.WithScriptName(ctx, "consumePause"),
		client,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error consuming pause: %w", err)
	}
	switch status {
	case 0:
		return nil
	case 1:
		return nil // Pause already consumed
	default:
		return fmt.Errorf("unknown response leasing pause: %d", status)
	}
}

func (m unshardedMgr) EventHasPauses(ctx context.Context, workspaceID uuid.UUID, event string) (bool, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "EventHasPauses"), redis_telemetry.ScopePauses)

	pause := m.u.Pauses()
	key := pause.kg.PauseEvent(ctx, workspaceID, event)
	cmd := pause.Client().B().Exists().Key(key).Build()
	return pause.Client().Do(ctx, cmd).AsBool()
}

func (m unshardedMgr) PauseByID(ctx context.Context, pauseID uuid.UUID) (*state.Pause, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "PauseByID"), redis_telemetry.ScopePauses)

	pauses := m.u.Pauses()
	cmd := pauses.Client().B().Get().Key(pauses.kg.Pause(ctx, pauseID)).Build()
	str, err := pauses.Client().Do(ctx, cmd).ToString()
	if err == rueidis.Nil {
		return nil, state.ErrPauseNotFound
	}
	if err != nil {
		return nil, err
	}
	pause := &state.Pause{}
	err = json.Unmarshal([]byte(str), pause)
	return pause, err
}

func (m unshardedMgr) PauseByInvokeCorrelationID(ctx context.Context, wsID uuid.UUID, correlationID string) (*state.Pause, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "PauseByInvokeCorrelationID"), redis_telemetry.ScopePauses)

	global := m.u.Global()
	key := global.kg.Invoke(ctx, wsID)
	cmd := global.Client().B().Hget().Key(key).Field(correlationID).Build()
	pauseIDstr, err := global.Client().Do(ctx, cmd).ToString()
	if err == rueidis.Nil {
		return nil, state.ErrInvokePauseNotFound
	}
	if err != nil {
		return nil, err
	}

	pauseID, err := uuid.Parse(pauseIDstr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pauseID UUID: %w", err)
	}
	return m.PauseByID(ctx, pauseID)
}

func (m unshardedMgr) PausesByID(ctx context.Context, ids ...uuid.UUID) ([]*state.Pause, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "PausesByID"), redis_telemetry.ScopePauses)

	pause := m.u.Pauses()
	if len(ids) == 0 {
		return nil, nil
	}

	keys := make([]string, len(ids))
	for n, id := range ids {
		keys[n] = pause.kg.Pause(ctx, id)
	}

	cmd := pause.Client().B().Mget().Key(keys...).Build()
	strings, err := pause.Client().Do(ctx, cmd).AsStrSlice()
	if err == rueidis.Nil {
		return nil, state.ErrPauseNotFound
	}
	if err != nil {
		return nil, err
	}

	var merr error

	pauses := []*state.Pause{}
	for _, item := range strings {
		if len(item) == 0 {
			continue
		}

		pause := &state.Pause{}
		err = json.Unmarshal([]byte(item), pause)
		if err != nil {
			merr = errors.Join(merr, err)
			continue
		}
		pauses = append(pauses, pause)
	}

	return pauses, merr
}

// PauseByStep returns a specific pause for a given workflow run, from a given step.
//
// This is required when continuing a step function from an async step, ie. one that
// has deferred results which must be continued by resuming the specific pause set
// up for the given step ID.
func (m unshardedMgr) PauseByStep(ctx context.Context, i state.Identifier, actionID string) (*state.Pause, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "PauseByStep"), redis_telemetry.ScopePauses)

	pauses := m.u.Pauses()

	// Access sharded value first
	cmd := pauses.Client().B().Get().Key(pauses.kg.PauseStep(ctx, i, actionID)).Build()
	str, err := pauses.Client().Do(ctx, cmd).ToString()

	if err == rueidis.Nil {
		return nil, state.ErrPauseNotFound
	}
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(str)
	if err != nil {
		return nil, err
	}

	// Then access value
	cmd = pauses.Client().B().Get().Key(pauses.kg.Pause(ctx, id)).Build()
	byt, err := pauses.Client().Do(ctx, cmd).AsBytes()

	if err == rueidis.Nil {
		return nil, state.ErrPauseNotFound
	}
	if err != nil {
		return nil, err
	}

	pause := &state.Pause{}
	err = json.Unmarshal(byt, pause)
	return pause, err
}

// PausesByEvent returns all pauses for a given event within a workspace.
func (m unshardedMgr) PausesByEvent(ctx context.Context, workspaceID uuid.UUID, event string) (state.PauseIterator, error) {
	return m.pausesByEvent(ctx, workspaceID, event, time.Time{})
}

// pausesByEvent returns all pauses for a given event within a workspace.
func (m unshardedMgr) pausesByEvent(ctx context.Context, workspaceID uuid.UUID, event string, aggregateStart time.Time) (state.PauseIterator, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "PausesByEvent"), redis_telemetry.ScopePauses)

	pauses := m.u.Pauses()

	key := pauses.kg.PauseEvent(ctx, workspaceID, event)
	// If there are > 1000 keys in the hmap, use scanning

	cntCmd := pauses.Client().B().Hlen().Key(key).Build()
	cnt, err := pauses.Client().Do(ctx, cntCmd).AsInt64()

	if err != nil || cnt > 1000 {
		key := pauses.kg.PauseEvent(ctx, workspaceID, event)
		iter := &scanIter{
			count:          cnt,
			r:              pauses.Client(),
			aggregateStart: aggregateStart,
		}
		err := iter.init(ctx, key, 1000)
		return iter, err
	}

	// If there are less than a thousand items, query the keys
	// for iteration.
	iter := &bufIter{r: pauses.Client(), aggregateStart: aggregateStart}
	err = iter.init(ctx, key)
	return iter, err
}

func (m unshardedMgr) PausesByEventSince(ctx context.Context, workspaceID uuid.UUID, event string, since time.Time) (state.PauseIterator, error) {
	start := time.Now()

	if since.IsZero() {
		return m.pausesByEvent(ctx, workspaceID, event, start)
	}

	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "PausesByEventSince"), redis_telemetry.ScopePauses)

	pauses := m.u.Pauses()

	// Load all items in the set.
	cmd := pauses.Client().B().
		Zrangebyscore().
		Key(pauses.kg.PauseIndex(ctx, "add", workspaceID, event)).
		Min(strconv.Itoa(int(since.Unix()))).
		Max("+inf").
		Build()
	ids, err := pauses.Client().Do(ctx, cmd).AsStrSlice()
	if err != nil {
		return nil, err
	}

	iter := &keyIter{
		r:     pauses.Client(),
		kf:    pauses.kg,
		start: start,
	}
	err = iter.init(ctx, ids, 100)
	return iter, err
}

func (m unshardedMgr) EvaluablesByID(ctx context.Context, ids ...uuid.UUID) ([]expr.Evaluable, error) {
	items, err := m.PausesByID(ctx, ids...)
	if err != nil {
		return nil, err
	}
	evaluables := make([]expr.Evaluable, len(items))
	for n, i := range items {
		evaluables[n] = i
	}
	return evaluables, nil
}

func (m unshardedMgr) LoadEvaluablesSince(ctx context.Context, workspaceID uuid.UUID, eventName string, since time.Time, do func(context.Context, expr.Evaluable) error) error {

	// Keep a list of pauses that should be deleted because they've expired.
	//
	// Note that we don't do this in the iteration loop, as redis can use either HSCAN or
	// MGET;  deleting during iteration may lead to skipped items.
	expired := []*state.Pause{}

	it, err := m.PausesByEventSince(ctx, workspaceID, eventName, since)
	if err != nil {
		return err
	}
	for it.Next(ctx) {
		pause := it.Val(ctx)
		if pause == nil {
			continue
		}

		if pause.Expires.Time().Before(time.Now()) {
			shouldDelete := pause.Expires.Time().Add(consts.PauseExpiredDeletionGracePeriod).Before(time.Now())
			if shouldDelete {
				expired = append(expired, pause)
			}
			continue
		}

		if err := do(ctx, pause); err != nil {
			return err
		}
	}

	// GC pauses on fetch.
	for _, pause := range expired {
		_ = m.DeletePause(ctx, *pause)
	}

	if it.Error() != context.Canceled && it.Error() != scanDoneErr {
		return it.Error()
	}

	return nil
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

var scanDoneErr = fmt.Errorf("scan done")

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
	// Reset the index.
	i.i = -1

	if i.cursor == 0 {
		// We're done, no need to fetch.
		return scanDoneErr
	}

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
	}

	return fmt.Errorf("Scanned max times without finding pause values")
}

func (i *scanIter) Next(ctx context.Context) bool {
	i.l.Lock()
	defer i.l.Unlock()

	if i.i >= (len(i.vals.Elements) - 1) {
		err := i.fetch(ctx)
		if err == scanDoneErr {
			// No more present.
			i.err = context.Canceled
			if !i.aggregateStart.IsZero() {
				dur := time.Since(i.aggregateStart).Milliseconds()
				metrics.HistogramAggregatePausesLoadDuration(ctx, dur, metrics.HistogramOpt{
					PkgName: pkgName,
					// TODO: tag workspace ID eventually??
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
	status, err := strconv.Atoi(v)
	if err != nil {
		return nil, fmt.Errorf("invalid function status stored in run metadata: %#v", v)
	}
	m.Status = enums.RunStatus(status)

	parseInt := func(v string) (int, error) {
		str, ok := data[v]
		if !ok {
			return 0, fmt.Errorf("no '%s' stored in run metadata", v)
		}
		val, err := strconv.Atoi(str)
		if err != nil {
			return 0, fmt.Errorf("invalid '%s' stored in run metadata", v)
		}
		return val, nil
	}

	m.StateSize, _ = parseInt("state_size")
	m.EventSize, _ = parseInt("event_size")
	m.StepCount, _ = parseInt("step_count")

	if val, ok := data["version"]; ok && val != "" {
		v, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid metadata version detected: %#v", val)
		}

		m.Version = v
	}

	if val, ok := data["rv"]; ok && val != "" {
		v, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid hash version detected: %#v", val)
		}
		m.RequestVersion = v
	}

	if val, ok := data["sat"]; ok && val != "" {
		v, err := strconv.ParseInt(val, 10, 64)
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
	vals  []string
	idx   int64
	err   error
	start time.Time
}

func (i *keyIter) Error() error {
	return i.err
}

func (i *keyIter) init(ctx context.Context, keys []string, chunk int64) error {
	i.keys = keys
	i.chunk = chunk
	err := i.fetch(ctx)
	if err == scanDoneErr {
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
		})
		return scanDoneErr
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
	if err == scanDoneErr {
		return false
	}
	return err == nil
}

func (i *keyIter) Val(ctx context.Context) *state.Pause {
	if len(i.vals) == 0 {
		return nil
	}

	val := i.vals[0]
	i.vals = i.vals[1:]
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
