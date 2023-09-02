package redis_state

import (
	"bytes"
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

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/config/registration"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
)

//go:embed lua/*
var embedded embed.FS

var (
	// scripts stores all embedded lua scripts on initialization
	scripts = map[string]*rueidis.Lua{}
	include = regexp.MustCompile(`-- \$include\(([\w.]+)\)`)

	ErrNoFunctionLoader = fmt.Errorf("No function loader specified within redis state store")

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

func (c Config) Manager(ctx context.Context) (state.Manager, error) {
	opts, err := c.ConnectOpts()
	if err != nil {
		return nil, err
	}

	return New(
		ctx,
		WithConnectOpts(opts),
		WithKeyGenerator(DefaultKeyFunc{Prefix: c.KeyPrefix}),
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
	m := &mgr{
		kf: DefaultKeyFunc{},
	}

	for _, opt := range opts {
		opt(m)
	}

	if m.r == nil {
		var err error
		m.r, err = rueidis.NewClient(rueidis.ClientOption{
			InitAddress: []string{"localhost:6379"},
			Password:    "",
		})
		return m, err
	}

	if m.pauseR == nil {
		// Use the standard redis client for pauses
		m.pauseR = m.r
	}

	return m, nil
}

// WithConnectOpts allows you to customize the options used to connect to Redis.
//
// This panics if the client cannot connect.
func WithConnectOpts(o rueidis.ClientOption) Opt {
	return func(m *mgr) {
		var err error
		m.r, err = rueidis.NewClient(o)
		if err != nil {
			panic(fmt.Errorf("unable to connect to redis with client opts: %w", err))
		}
	}
}

// WithKeyPrefix uses a specific key prefix
func WithKeyPrefix(prefix string) Opt {
	return func(m *mgr) {
		m.kf = DefaultKeyFunc{
			Prefix: prefix,
		}
	}
}

// WithRedisClient uses an already connected redis client.
func WithRedisClient(r rueidis.Client) Opt {
	return func(m *mgr) {
		m.r = r
	}
}

// WithPauseRedisClient uses an already connected redis client for managing pauses.
func WithPauseRedisClient(r rueidis.Client) Opt {
	return func(m *mgr) {
		m.pauseR = r
	}
}

// WithKeyGenerator specifies the function to use when creating keys for
// each stored data type.
func WithKeyGenerator(kf KeyGenerator) Opt {
	return func(m *mgr) {
		m.kf = kf
	}
}

// WithOnComplete supplies a callback which is triggered any time a function
// run completes.
func WithFunctionCallbacks(f ...state.FunctionCallback) Opt {
	return func(m *mgr) {
		m.callbacks = f
	}
}

// WithFunctionLoader adds a function loader to the state interface.
//
// As of v0.13.0, function configuration is stored outside of the state store,
// either in a cache or a datastore.  Because this is read-heavy, this should
// be cached where possible.
func WithFunctionLoader(fl state.FunctionLoader) Opt {
	return func(m *mgr) {
		m.fl = fl
	}
}

type mgr struct {
	kf KeyGenerator
	fl state.FunctionLoader

	// this is the standard redis client for the state store.
	r rueidis.Client
	// this is the redis client for managing pauses.
	pauseR rueidis.Client

	callbacks []state.FunctionCallback
}

// OnFunctionStatus adds a callback to be called whenever functions
// transition status.
func (m *mgr) OnFunctionStatus(f state.FunctionCallback) {
	m.callbacks = append(m.callbacks, f)
}

func (m mgr) New(ctx context.Context, input state.Input) (state.State, error) {
	f, err := m.LoadFunction(ctx, input.Identifier)
	if err != nil {
		return nil, err
	}

	// We marshal this ahead of creating a redis transaction as it's necessary
	// every time and reduces the duration that the lock is held.
	events, err := json.Marshal(input.EventBatchData)
	if err != nil {
		return nil, err
	}

	metadata := runMetadata{
		Identifier: input.Identifier,
		Pending:    1,
		Debugger:   input.Debugger,
		Version:    currentVersion,
		Context:    input.Context,
	}
	if input.OriginalRunID != nil {
		metadata.OriginalRunID = input.OriginalRunID.String()
	}
	if input.RunType != nil {
		metadata.RunType = *input.RunType
	}

	metadataByt, err := json.Marshal(metadata.Map())
	if err != nil {
		return nil, fmt.Errorf("error storing run state in redis: %w", err)
	}

	var stepsByt []byte
	if len(input.Steps) > 0 {
		stepsByt, err = json.Marshal(input.Steps)
		if err != nil {
			return nil, fmt.Errorf("error storing run state in redis: %w", err)
		}
	}

	history := state.NewHistory()
	history.Type = enums.HistoryTypeFunctionStarted
	history.Identifier = input.Identifier
	history.CreatedAt = time.UnixMilli(int64(input.Identifier.RunID.Time()))

	args, err := StrSlice([]any{
		events,
		metadataByt,
		stepsByt,
		history,
		history.CreatedAt.UnixMilli(),
	})
	if err != nil {
		return nil, err
	}

	status, err := scripts["new"].Exec(
		ctx,
		m.r,
		[]string{
			m.kf.Idempotency(ctx, input.Identifier),
			m.kf.Events(ctx, input.Identifier),
			m.kf.RunMetadata(ctx, input.Identifier.RunID),
			m.kf.Actions(ctx, input.Identifier),
			m.kf.History(ctx, input.Identifier.RunID),
		},
		args,
	).AsInt64()

	if err != nil {
		return nil, fmt.Errorf("error storing run state in redis: %w", err)
	}

	if status == 1 {
		return nil, state.ErrIdentifierExists
	}

	go m.runCallbacks(ctx, input.Identifier, enums.RunStatusRunning)

	return state.NewStateInstance(
			*f,
			input.Identifier,
			metadata.Metadata(),
			input.EventBatchData,
			input.Steps,
			map[string]error{},
			make([]string, 0),
		),
		nil
}

func (m mgr) IsComplete(ctx context.Context, runID ulid.ULID) (bool, error) {
	cmd := m.r.B().Hget().Key(m.kf.RunMetadata(ctx, runID)).Field("pending").Build()
	val, err := m.r.Do(ctx, cmd).AsBytes()
	if err != nil {
		return false, err
	}
	return bytes.Equal(val, []byte("0")), nil
}

func (m mgr) metadata(ctx context.Context, runID ulid.ULID) (*runMetadata, error) {
	cmd := m.r.B().Hgetall().Key(m.kf.RunMetadata(ctx, runID)).Build()
	val, err := m.r.Do(ctx, cmd).AsStrMap()
	if err != nil {
		return nil, err
	}
	return NewRunMetadata(val)
}

func (m mgr) Cancel(ctx context.Context, id state.Identifier) error {
	now := time.Now()

	args, err := StrSlice([]any{
		state.History{
			ID:         state.HistoryID(),
			GroupID:    state.GroupIDFromContext(ctx),
			Type:       enums.HistoryTypeFunctionCancelled,
			Identifier: id,
			CreatedAt:  now,
		},
		now.UnixMilli(),
	})
	if err != nil {
		return err
	}

	status, err := scripts["cancel"].Exec(
		ctx,
		m.r,
		[]string{m.kf.RunMetadata(ctx, id.RunID), m.kf.History(ctx, id.RunID)},
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error cancelling: %w", err)
	}
	switch status {
	case 0:
		go m.runCallbacks(ctx, id, enums.RunStatusCancelled)
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

func (m mgr) SetStatus(ctx context.Context, id state.Identifier, status enums.RunStatus) error {
	now := time.Now()

	args, err := StrSlice([]any{
		int(status),
		state.History{
			ID:         state.HistoryID(),
			GroupID:    state.GroupIDFromContext(ctx),
			Type:       enums.HistoryTypeFunctionStatusUpdated,
			Identifier: id,
			CreatedAt:  now,
			Data: map[string]any{
				"status": status.String(),
			},
		},
		now.UnixMilli(),
	})
	if err != nil {
		return err
	}

	ret, err := scripts["setStatus"].Exec(
		ctx,
		m.r,
		[]string{m.kf.RunMetadata(ctx, id.RunID), m.kf.History(ctx, id.RunID)},
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error cancelling: %w", err)
	}
	if ret == 0 {
		go m.runCallbacks(ctx, id, status)
		return nil
	}
	return fmt.Errorf("unknown return value cancelling function: %d", status)
}

func (m mgr) Metadata(ctx context.Context, runID ulid.ULID) (*state.Metadata, error) {
	metadata, err := m.metadata(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to load metadata: %w", err)
	}
	meta := metadata.Metadata()
	return &meta, nil
}

func (m mgr) LoadFunction(ctx context.Context, id state.Identifier) (*inngest.Function, error) {
	if m.fl == nil {
		return nil, ErrNoFunctionLoader
	}
	return m.fl.LoadFunction(ctx, id)
}

func (m mgr) Load(ctx context.Context, runID ulid.ULID) (state.State, error) {
	// XXX: Use a pipeliner to improve speed.
	metadata, err := m.metadata(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to load metadata; %w", err)
	}

	id := metadata.Identifier

	fn, err := m.fl.LoadFunction(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("unable to load function from state function loader: %s: %w", id.WorkflowID, err)
	}

	// Load events.
	events := []map[string]any{}
	switch metadata.Version {
	case 0: // pre-batch days
		cmd := m.r.B().Get().Key(m.kf.Event(ctx, id)).Build()
		byt, err := m.r.Do(ctx, cmd).AsBytes()
		if err != nil {
			return nil, fmt.Errorf("failed to get event; %w", err)
		}
		event := map[string]any{}
		if err := json.Unmarshal(byt, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal event; %w", err)
		}
		events = []map[string]any{event}
	default: // current default is 1
		// Load the batch of events
		cmd := m.r.B().Get().Key(m.kf.Events(ctx, id)).Build()
		byt, err := m.r.Do(ctx, cmd).AsBytes()
		if err != nil {
			return nil, fmt.Errorf("failed to get batch; %w", err)
		}
		if err := json.Unmarshal(byt, &events); err != nil {
			return nil, fmt.Errorf("failed to unmarshal batch; %w", err)
		}
	}

	// Load the actions.  This is a map of step IDs to JSON-encoded results.
	cmd := m.r.B().Hgetall().Key(m.kf.Actions(ctx, id)).Build()
	rmap, err := m.r.Do(ctx, cmd).AsStrMap()
	if err != nil {
		return nil, fmt.Errorf("failed loading actions; %w", err)
	}
	actions := map[string]any{}
	for stepID, marshalled := range rmap {
		var data any
		err = json.Unmarshal([]byte(marshalled), &data)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal step \"%s\" with data \"%s\"; %w", stepID, marshalled, err)
		}
		actions[stepID] = data
	}

	// Load the errors.  This is a map of step IDs to error strings.
	// The original error type is not preserved.
	cmd = m.r.B().Hgetall().Key(m.kf.Errors(ctx, id)).Build()
	rmap, err = m.r.Do(ctx, cmd).AsStrMap()
	if err != nil {
		return nil, fmt.Errorf("failed to load errors; %w", err)
	}
	errors := map[string]error{}
	for stepID, str := range rmap {
		errors[stepID] = fmt.Errorf(str)
	}

	meta := metadata.Metadata()

	cmd = m.r.B().Lrange().Key(m.kf.Stack(ctx, id.RunID)).Start(0).Stop(-1).Build()
	stack, err := m.r.Do(ctx, cmd).AsStrSlice()
	if err != nil {
		return nil, fmt.Errorf("error fetching stack: %w", err)
	}

	return state.NewStateInstance(*fn, id, meta, events, actions, errors, stack), nil
}

func (m mgr) StackIndex(ctx context.Context, runID ulid.ULID, stepID string) (int, error) {
	cmd := m.r.B().Lrange().Key(m.kf.Stack(ctx, runID)).Start(0).Stop(-1).Build()
	stack, err := m.r.Do(ctx, cmd).AsStrSlice()
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

func (m mgr) SaveResponse(ctx context.Context, i state.Identifier, r state.DriverResponse, attempt int) (int, error) {
	var (
		data            any
		result          any
		err             error
		typ             enums.HistoryType
		funcFailHistory state.History
	)

	now := time.Now()

	if r.Err == nil {
		result = r.Output
		typ = enums.HistoryTypeStepCompleted
		if data, err = json.Marshal(r.Output); err != nil {
			return 0, fmt.Errorf("error marshalling step output: %w", err)
		}
	} else {
		typ = enums.HistoryTypeStepErrored
		data = output(map[string]any{
			"error":  r.Err,
			"output": r.Output,
		})
		result = data
		if r.Final() {
			typ = enums.HistoryTypeStepFailed
			funcFailHistory = state.History{
				ID:         state.HistoryID(),
				Type:       enums.HistoryTypeFunctionFailed,
				Identifier: i,
				CreatedAt:  now,
			}
		}
	}

	stepOutput := false
	if len(r.Generator) == 0 && (typ == enums.HistoryTypeStepCompleted || typ == enums.HistoryTypeStepFailed) {
		// This is only the step output if the step is complete and this
		// isn't a generator response.
		stepOutput = true
	}

	stepHistory := state.History{
		ID:         state.HistoryID(),
		GroupID:    state.GroupIDFromContext(ctx),
		Type:       typ,
		Identifier: i,
		CreatedAt:  now,
		Data: state.HistoryStep{
			ID:         r.Step.ID,
			Name:       r.Step.Name,
			Attempt:    attempt,
			Data:       result,
			StepOutput: stepOutput,
		},
	}

	args, err := StrSlice([]any{
		data,
		r.Step.ID,
		r.Err != nil,
		r.Final(),
		stepHistory,
		funcFailHistory,
		now.UnixMilli(),
	})
	if err != nil {
		return 0, err
	}

	index, err := scripts["saveResponse"].Exec(
		ctx,
		m.r,
		[]string{
			m.kf.Actions(ctx, i),
			m.kf.Errors(ctx, i),
			m.kf.RunMetadata(ctx, i.RunID),
			m.kf.History(ctx, i.RunID),
			m.kf.Stack(ctx, i.RunID),
		},
		args,
	).AsInt64()
	if err != nil {
		return 0, fmt.Errorf("error saving response: %w", err)
	}

	if r.Err != nil && r.Final() {
		// Trigger error callbacks
		go m.runCallbacks(ctx, i, enums.RunStatusFailed)
	}

	return int(index), nil
}

func (m mgr) Started(ctx context.Context, id state.Identifier, stepID string, attempt int) error {
	now := time.Now()
	byt, err := json.Marshal(state.History{
		ID:         state.HistoryID(),
		GroupID:    state.GroupIDFromContext(ctx),
		Type:       enums.HistoryTypeStepStarted,
		Identifier: id,
		CreatedAt:  now,
		Data: state.HistoryStep{
			ID:      stepID,
			Attempt: attempt,
		},
	})
	if err != nil {
		return err
	}
	cmd := m.r.B().Zadd().
		Key(m.kf.History(ctx, id.RunID)).
		ScoreMember().
		ScoreMember(float64(now.UnixMilli()), string(byt)).
		Build()
	return m.r.Do(ctx, cmd).Error()
}

func (m mgr) Scheduled(ctx context.Context, i state.Identifier, stepID string, attempt int, at *time.Time) error {
	now := time.Now()

	if at != nil && at.Before(time.Now()) {
		// No need to save time if it's before now.
		at = nil
	}

	args, err := StrSlice([]any{
		state.History{
			ID:         state.HistoryID(),
			GroupID:    state.GroupIDFromContext(ctx),
			Type:       enums.HistoryTypeStepScheduled,
			Identifier: i,
			CreatedAt:  now,
			Data: state.HistoryStep{
				ID:      stepID,
				Attempt: attempt,
				Data:    at,
			},
		},
		now.UnixMilli(),
	})
	if err != nil {
		return err
	}

	err = scripts["scheduled"].Exec(
		ctx,
		m.r,
		[]string{m.kf.RunMetadata(ctx, i.RunID), m.kf.History(ctx, i.RunID)},
		args,
	).Error()
	if err != nil {
		return fmt.Errorf("error updating scheduled state: %w", err)
	}
	return nil
}

func (m mgr) Finalized(ctx context.Context, i state.Identifier, stepID string, attempt int, withStatus ...enums.RunStatus) error {
	now := time.Now()

	// Don't set status by default.
	finalStatus := -1
	if len(withStatus) >= 1 {
		finalStatus = int(withStatus[0])
	}

	history := enums.HistoryTypeFunctionCompleted
	switch finalStatus {
	case int(enums.RunStatusFailed):
		history = enums.HistoryTypeFunctionFailed
	}

	args, err := StrSlice([]any{
		state.History{
			ID: state.HistoryID(),
			// Function completions have no group ID.
			Type:       history,
			Identifier: i,
			CreatedAt:  now,
		},
		now.UnixMilli(),
		int(finalStatus),
	})
	if err != nil {
		return err
	}

	status, err := scripts["finalize"].Exec(
		ctx,
		m.r,
		[]string{m.kf.RunMetadata(ctx, i.RunID), m.kf.History(ctx, i.RunID)},
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error finalizing: %w", err)
	}
	if status == 1 && len(withStatus) == 1 {
		go m.runCallbacks(ctx, i, withStatus[0])
		return nil
	}
	if status == 1 {
		go m.runCallbacks(ctx, i, enums.RunStatusCompleted)
	}
	return nil
}

func (m mgr) SavePause(ctx context.Context, p state.Pause) error {
	packed, err := json.Marshal(p)
	if err != nil {
		return err
	}

	evt := ""
	if p.Event != nil {
		evt = *p.Event
	}

	keys := []string{
		m.kf.PauseID(ctx, p.ID),
		m.kf.PauseStep(ctx, p.Identifier, p.Incoming),
		m.kf.PauseEvent(ctx, p.WorkspaceID, evt),
		m.kf.History(ctx, p.Identifier.RunID),
	}

	stepID := p.Incoming
	if p.DataKey != "" {
		stepID = p.DataKey
	}
	now := time.Now()
	log := state.History{
		ID:         state.HistoryID(),
		GroupID:    state.GroupIDFromContext(ctx),
		Type:       enums.HistoryTypeStepWaiting,
		Identifier: p.Identifier,
		CreatedAt:  now,
		Data: state.HistoryStep{
			ID:      stepID,
			Name:    p.StepName,
			Attempt: p.Attempt,
			Data: state.HistoryStepWaitingData{
				EventName:  p.Event,
				Expression: p.Expression,
				ExpiryTime: time.Time(p.Expires),
			},
		},
	}

	// Add 1 second because int will truncate the float. Otherwise, timeouts
	// will be 1 second less than configured.
	ttl := int(time.Until(p.Expires.Time()).Seconds()) + 1

	// Ensure the TTL is at least 1 second. This probably will always be true
	// since we're adding 1 second above. But you never know if some code
	// between expiry creation and here will take longer than expected.
	if ttl < 1 {
		ttl = 1
	}

	args, err := StrSlice([]any{
		string(packed),
		p.ID.String(),
		evt,
		ttl,
		// Add at least 10 minutes to this pause, allowing us to process the
		// pause by ID for 10 minutes past expiry.
		int(time.Until(p.Expires.Time().Add(10 * time.Minute)).Seconds()),
		log,
		now.UnixMilli(),
	})
	if err != nil {
		return err
	}

	status, err := scripts["savePause"].Exec(
		ctx,
		m.pauseR,
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

func (m mgr) LeasePause(ctx context.Context, id uuid.UUID) error {
	args, err := StrSlice([]any{
		time.Now().UnixMilli(),
		state.PauseLeaseDuration.Seconds(),
	})
	if err != nil {
		return err
	}

	status, err := scripts["leasePause"].Exec(
		ctx,
		m.pauseR,
		[]string{m.kf.PauseID(ctx, id), m.kf.PauseLease(ctx, id)},
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

func (m mgr) ConsumePause(ctx context.Context, id uuid.UUID, data any) error {
	p, err := m.PauseByID(ctx, id)
	if err != nil {
		return err
	}

	marshalledData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("cannot marshal data to store in state: %w", err)
	}

	// Add a default event here, which is null and overwritten by everything.  This is necessary
	// to keep the same cluster key.
	eventKey := m.kf.PauseEvent(ctx, p.WorkspaceID, "-")
	if p.Event != nil {
		eventKey = m.kf.PauseEvent(ctx, p.WorkspaceID, *p.Event)
	}
	keys := []string{
		m.kf.PauseID(ctx, id),
		m.kf.PauseStep(ctx, p.Identifier, p.Incoming),
		eventKey,
		m.kf.Actions(ctx, p.Identifier),
		m.kf.Stack(ctx, p.Identifier.RunID),
		m.kf.History(ctx, p.Identifier.RunID),
	}

	stepID := p.Incoming
	if p.DataKey != "" {
		stepID = p.DataKey
	}
	now := time.Now()
	log := state.History{
		ID:         state.HistoryID(),
		GroupID:    state.GroupIDFromContext(ctx),
		Type:       enums.HistoryTypeStepCompleted,
		Identifier: p.Identifier,
		CreatedAt:  now,
		Data: state.HistoryStep{
			ID:      stepID,
			Name:    p.StepName,
			Attempt: p.Attempt,
			Data:    data,
		},
	}

	args, err := StrSlice([]any{
		id.String(),
		p.DataKey,
		string(marshalledData),
		log,
		now.UnixMilli(),
	})
	if err != nil {
		return err
	}

	status, err := scripts["consumePause"].Exec(
		ctx,
		m.pauseR,
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
		return state.ErrPauseNotFound
	default:
		return fmt.Errorf("unknown response leasing pause: %d", status)
	}
}

// PausesByEvent returns all pauses for a given event within a workspace.
func (m mgr) PausesByEvent(ctx context.Context, workspaceID uuid.UUID, event string) (state.PauseIterator, error) {
	key := m.kf.PauseEvent(ctx, workspaceID, event)
	// If there are > 1000 keys in the hmap, use scanning

	cntCmd := m.pauseR.B().Hlen().Key(key).Build()
	cnt, err := m.pauseR.Do(ctx, cntCmd).AsInt64()
	if err != nil || cnt > 1000 {
		key := m.kf.PauseEvent(ctx, workspaceID, event)
		cmd := m.pauseR.B().Hscan().Key(key).Cursor(0).Count(500).Build()
		scan, err := m.pauseR.Do(ctx, cmd).AsScanEntry()
		if err != nil {
			return nil, err
		}
		return &scanIter{
			r:      m.pauseR,
			key:    key,
			i:      -1,
			vals:   scan,
			cursor: int(scan.Cursor),
		}, nil
	}

	cmd := m.pauseR.B().Hkeys().Key(key).Cache()
	// Cache this for a second
	keys, err := m.pauseR.DoCache(ctx, cmd, time.Second).AsStrSlice()
	if err != nil {
		return nil, err
	}

	return &keyIter{i: 0, keys: keys, r: m.pauseR, key: key}, nil
}

func (m mgr) EventHasPauses(ctx context.Context, workspaceID uuid.UUID, event string) (bool, error) {
	key := m.kf.PauseEvent(ctx, workspaceID, event)
	cmd := m.pauseR.B().Exists().Key(key).Build()
	return m.pauseR.Do(ctx, cmd).AsBool()
}

func (m mgr) PauseByID(ctx context.Context, id uuid.UUID) (*state.Pause, error) {
	cmd := m.pauseR.B().Get().Key(m.kf.PauseID(ctx, id)).Build()
	str, err := m.pauseR.Do(ctx, cmd).ToString()
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

// PauseByStep returns a specific pause for a given workflow run, from a given step.
//
// This is required when continuing a step function from an async step, ie. one that
// has deferred results which must be continued by resuming the specific pause set
// up for the given step ID.
func (m mgr) PauseByStep(ctx context.Context, i state.Identifier, actionID string) (*state.Pause, error) {
	cmd := m.pauseR.B().Get().Key(m.kf.PauseStep(ctx, i, actionID)).Build()
	str, err := m.pauseR.Do(ctx, cmd).ToString()

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

	cmd = m.pauseR.B().Get().Key(m.kf.PauseID(ctx, id)).Build()
	byt, err := m.pauseR.Do(ctx, cmd).AsBytes()

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

func (m mgr) History(ctx context.Context, runID ulid.ULID) ([]state.History, error) {
	cmd := m.r.B().Zrange().Key(m.kf.History(ctx, runID)).Min("-inf").Max("+inf").Byscore().Build()
	items, err := m.r.Do(ctx, cmd).AsStrSlice()
	if err != nil {
		return nil, err
	}

	history := make([]state.History, len(items))
	for n, i := range items {
		var h state.History
		err := h.UnmarshalBinary([]byte(i))
		if err != nil {
			return nil, err
		}
		history[n] = h
	}

	return history, nil
}

func (m mgr) DeleteHistory(ctx context.Context, runID ulid.ULID, historyID ulid.ULID) error {
	// Fetch the items from the zset, and remove if the ID matches.
	//
	// XXX: We can make this more efficient by recording a map and zset as separate
	// keys.
	key := m.kf.History(ctx, runID)

	cmd := m.r.B().Zrange().Key(key).Min("-inf").Max("+inf").Byscore().Build()
	items, err := m.r.Do(ctx, cmd).AsStrSlice()
	if err != nil {
		return err
	}

	for _, i := range items {
		var h state.History
		err := h.UnmarshalBinary([]byte(i))
		if err != nil {
			return err
		}
		if h.ID == historyID {
			cmd = m.r.B().Zrem().Key(key).Member(i).Build()
			if err := m.r.Do(ctx, cmd).Error(); err != nil {
				return err
			}
			return nil
		}
	}

	return nil
}

func (m mgr) SaveHistory(ctx context.Context, i state.Identifier, h state.History) error {
	hkey := m.kf.History(ctx, i.RunID)

	byt, err := json.Marshal(h)
	if err != nil {
		return err
	}

	cmd := m.r.B().Zadd().
		Key(hkey).
		ScoreMember().
		ScoreMember(float64(h.CreatedAt.UnixMilli()), string(byt)).
		Build()
	return m.r.Do(ctx, cmd).Error()
}

func (m mgr) runCallbacks(ctx context.Context, id state.Identifier, status enums.RunStatus) {
	// Replace the context so that this isn't cancelled by any parents.
	callCtx := context.Background()
	for _, f := range m.callbacks {
		go func(fn state.FunctionCallback) {
			fn(callCtx, id, status)
		}(f)
	}
}

type keyIter struct {
	l sync.Mutex

	i    int
	keys []string

	// buffer stores items returned from HMGet temporarily.
	buffer []string

	// val stores the next val
	val *state.Pause

	r   rueidis.Client
	key string
	err error
}

func (i *keyIter) Err() error {
	return i.err
}

func (i *keyIter) Next(ctx context.Context) bool {
	i.l.Lock()
	defer i.l.Unlock()

	if len(i.buffer) > 0 {
		i.getNext(ctx)
		return i.err == nil
	}

	if len(i.keys) == 0 || i.i == len(i.keys) {
		return false
	}

	amt := 100
	if 100 > len(i.keys) {
		// Fetch all remaining keys.
		amt = len(i.keys)
	}

	// Take a buffer from the keys
	buffer := i.keys[:amt]

	cmd := i.r.B().Hmget().Key(i.key).Field(buffer...).Build()
	vals, err := i.r.Do(ctx, cmd).AsStrSlice()
	if err != nil {
		i.err = err
		return false
	}
	// Remove the fetched keys from remaining list
	i.keys = i.keys[amt:]
	// Update our buffer.
	i.buffer = vals

	// get the next item into Val
	i.getNext(ctx)

	return i.err == nil
}

// Buffer by running an MGET to get the values of the pauses.
func (i *keyIter) Val(ctx context.Context) *state.Pause {
	return i.val
}

func (i *keyIter) getNext(ctx context.Context) {
	if len(i.buffer) == 0 {
		return
	}
	str := i.buffer[0]
	i.buffer = i.buffer[1:]

	pause := &state.Pause{}
	i.err = json.Unmarshal([]byte(str), pause)
	i.val = pause
}

type scanIter struct {
	r rueidis.Client

	key    string
	i      int
	cursor int
	vals   rueidis.ScanEntry
}

func (i *scanIter) fetch(ctx context.Context) error {
	cmd := i.r.B().Hscan().Key(i.key).Cursor(uint64(i.cursor)).Count(500).Build()
	scan, err := i.r.Do(ctx, cmd).AsScanEntry()
	if err != nil {
		return err
	}
	i.cursor = int(scan.Cursor)
	i.vals = scan
	i.i = -1
	return nil
}

func (i *scanIter) Next(ctx context.Context) bool {
	if i.i >= (len(i.vals.Elements)-1) && i.cursor != 0 {
		if err := i.fetch(ctx); err != nil {
			return false
		}
	}

	if len(i.vals.Elements) == 0 || i.i >= (len(i.vals.Elements)-1) {
		return false
	}

	// Skip the ID
	i.i++
	// Get the value.
	i.i++
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

func NewRunMetadata(data map[string]string) (*runMetadata, error) {
	var err error
	m := &runMetadata{}

	v, ok := data["status"]
	if !ok {
		return nil, fmt.Errorf("no status stored in metadata")
	}
	status, err := strconv.Atoi(v)
	if err != nil {
		return nil, fmt.Errorf("invalid function status stored in run metadata: %#v", v)
	}
	m.Status = enums.RunStatus(status)

	str, ok := data["pending"]
	if !ok {
		return nil, fmt.Errorf("no created at stored in run metadata")
	}
	m.Pending, err = strconv.Atoi(str)
	if err != nil {
		return nil, fmt.Errorf("invalid pending stored in run metadata")
	}

	if val, ok := data["version"]; ok && val != "" {
		v, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid metadata version detected: %#v", val)
		}

		m.Version = v
	}

	// The below fields are optional
	if val, ok := data["debugger"]; ok {
		if val == "true" {
			m.Debugger = true
		}
	}
	if val, ok := data["originalRunID"]; ok {
		m.OriginalRunID = val
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

	return m, nil
}

// runMetadata is stored for each invocation of a function.  This is inserted when
// creating a new run, and stores the triggering event as well as workflow-specific
// metadata for the invocation.
type runMetadata struct {
	Identifier state.Identifier `json:"id"`
	Status     enums.RunStatus  `json:"status"`
	// These are the fields for standard state metadata.
	Pending       int            `json:"pending"`
	Debugger      bool           `json:"debugger"`
	RunType       string         `json:"runType,omitempty"`
	OriginalRunID string         `json:"originalRunID,omitempty"`
	Version       int            `json:"version"`
	Context       map[string]any `json:"ctx,omitempty"`
}

func (r runMetadata) Map() map[string]any {
	return map[string]any{
		"id":            r.Identifier,
		"status":        int(r.Status), // Always store this as an int
		"pending":       r.Pending,
		"debugger":      r.Debugger,
		"runType":       r.RunType,
		"originalRunID": r.OriginalRunID,
		"version":       r.Version,
		"ctx":           r.Context,
	}
}

func (r runMetadata) Metadata() state.Metadata {
	m := state.Metadata{
		Identifier: r.Identifier,
		Pending:    r.Pending,
		Debugger:   r.Debugger,
		Status:     r.Status,
		Version:    r.Version,
		Context:    r.Context,
	}

	if r.RunType != "" {
		m.RunType = &r.RunType
	}
	if r.OriginalRunID != "" {
		id := ulid.MustParse(r.OriginalRunID)
		m.OriginalRunID = &id
	}
	return m
}

// output is a map which implements BinaryMarshaller, for inserting data within
// history items.
type output map[string]any

func (o output) MarshalBinary() ([]byte, error) {
	return json.Marshal(o)
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
