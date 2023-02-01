package redis_state

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	json "github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/inngest/inngest/inngest"
	"github.com/inngest/inngest/pkg/config/registration"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/inmemory"
	"github.com/oklog/ulid/v2"
)

//go:embed lua/*
var embedded embed.FS

var (
	// scripts stores all embedded lua scripts on initialization
	scripts = map[string]*redis.Script{}
	include = regexp.MustCompile(`-- \$include\(([\w.]+)\)`)
)

func init() {
	// register the redis driver
	registration.RegisterState(func() any { return &Config{} })

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
		scripts[name] = redis.NewScript(val)
	}
}

// Config registers the configuration for the in-memory state store,
// and provides a factory for the state manager based off of the config.
type Config struct {
	// DSN contains the entire configuration in a single string, if
	// provided (eg. redis://user:pass@host:port/db)
	DSN *string

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
		WithConnectOpts(*opts),
		WithExpiration(c.Expiry),
		WithKeyGenerator(DefaultKeyFunc{Prefix: c.KeyPrefix}),
	)
}

func (c Config) ConnectOpts() (*redis.Options, error) {
	if c.DSN != nil {
		return redis.ParseURL(*c.DSN)
	}

	opts := redis.Options{
		Addr:     fmt.Sprintf("%s:%d", c.Host, c.Port),
		DB:       c.DB,
		Username: c.Username,
		Password: c.Password,
	}
	if c.MaxRetries != nil {
		opts.MaxRetries = *c.MaxRetries
	}
	if c.PoolSize != nil {
		opts.PoolSize = *c.PoolSize
	}
	return &opts, nil
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
		m.r = redis.NewClient(&redis.Options{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
		})
	}

	return m, m.r.Ping(ctx).Err()
}

// WithConnectOpts allows you to customize the options used to connect to Redis.
func WithConnectOpts(o redis.Options) Opt {
	return func(m *mgr) {
		m.r = redis.NewClient(&o)
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
func WithRedisClient(r redis.UniversalClient) Opt {
	return func(m *mgr) {
		m.r = r
	}
}

// WithKeyGenerator specifies the function to use when creating keys for
// each stored data type.
func WithKeyGenerator(kf KeyGenerator) Opt {
	return func(m *mgr) {
		m.kf = kf
	}
}

// WithExpiration specifies the TTL to use when setting values in Redis.
func WithExpiration(ttl time.Duration) Opt {
	return func(m *mgr) {
		m.expiry = ttl
	}
}

// WithOnComplete supplies a callback which is triggered any time a function
// run completes.
func WithFunctionCallbacks(f ...state.FunctionCallback) Opt {
	return func(m *mgr) {
		m.callbacks = f
	}
}

type mgr struct {
	expiry time.Duration
	kf     KeyGenerator
	r      redis.UniversalClient

	callbacks []state.FunctionCallback
}

// OnFunctionStatus adds a callback to be called whenever functions
// transition status.
func (m *mgr) OnFunctionStatus(f state.FunctionCallback) {
	m.callbacks = append(m.callbacks, f)
}

func (m mgr) New(ctx context.Context, input state.Input) (state.State, error) {
	// We marshal this ahead of creating a redis transaction as it's necessary
	// every time and reduces the duration that the lock is held.
	event, err := json.Marshal(input.EventData)
	if err != nil {
		return nil, err
	}

	// Set the workflow.
	workflow, err := json.Marshal(input.Workflow)
	if err != nil {
		return nil, err
	}
	metadata := runMetadata{
		Identifier: input.Identifier,
		Pending:    1,
		Debugger:   input.Debugger,
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

	status, err := scripts["new"].Eval(
		ctx,
		m.r,
		[]string{
			m.kf.Idempotency(ctx, input.Identifier),
			m.kf.Event(ctx, input.Identifier),
			m.kf.Workflow(ctx, input.Workflow.UUID, input.Workflow.Version),
			m.kf.RunMetadata(ctx, input.Identifier.RunID),
			m.kf.Actions(ctx, input.Identifier),
			m.kf.History(ctx, input.Identifier.RunID),
		},
		event,
		workflow,
		metadataByt,
		stepsByt,
		m.expiry,
		history,
		history.CreatedAt.UnixMilli(),
	).Int64()

	if err != nil {
		return nil, fmt.Errorf("error storing run state in redis: %w", err)
	}

	if status == 1 {
		return nil, state.ErrIdentifierExists
	}

	go m.runCallbacks(ctx, input.Identifier, enums.RunStatusRunning)

	return inmemory.NewStateInstance(
			input.Workflow,
			input.Identifier,
			metadata.Metadata(),
			input.EventData,
			input.Steps,
			map[string]error{},
		),
		nil
}

func (m mgr) IsComplete(ctx context.Context, runID ulid.ULID) (bool, error) {
	val, err := m.r.HGet(ctx, m.kf.RunMetadata(ctx, runID), "pending").Result()
	if err != nil {
		return false, err
	}
	return val == "0", nil
}

func (m mgr) metadata(ctx context.Context, runID ulid.ULID) (*runMetadata, error) {
	val, err := m.r.HGetAll(ctx, m.kf.RunMetadata(ctx, runID)).Result()
	if err != nil {
		return nil, err
	}
	return NewRunMetadata(val)
}

func (m mgr) Cancel(ctx context.Context, id state.Identifier) error {
	now := time.Now()
	status, err := scripts["cancel"].Eval(
		ctx,
		m.r,
		[]string{m.kf.RunMetadata(ctx, id.RunID), m.kf.History(ctx, id.RunID)},
		state.History{
			ID:         state.HistoryID(),
			Type:       enums.HistoryTypeFunctionCancelled,
			Identifier: id,
			CreatedAt:  now,
		},
		now.UnixMilli(),
	).Int64()
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

func (m mgr) Metadata(ctx context.Context, runID ulid.ULID) (*state.Metadata, error) {
	metadata, err := m.metadata(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to load metadata: %w", err)
	}
	meta := metadata.Metadata()
	return &meta, nil
}

func (m mgr) Load(ctx context.Context, runID ulid.ULID) (state.State, error) {
	// XXX: Use a pipeliner to improve speed.
	metadata, err := m.metadata(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to load metadata; %w", err)
	}

	id := metadata.Identifier

	// Load the workflow.
	byt, err := m.r.Get(ctx, m.kf.Workflow(ctx, id.WorkflowID, metadata.Identifier.WorkflowVersion)).Bytes()
	if err != nil {
		return nil, fmt.Errorf("failed to load workflow; %w", err)
	}
	w := &inngest.Workflow{}
	if err := json.Unmarshal(byt, w); err != nil {
		return nil, fmt.Errorf("failed to unmarshal workflow; %w", err)
	}

	// We must ensure that the workflow UUID and Version are marshalled in JSON.
	// In the dev server these are blank, so we force-add them here.
	w.UUID = metadata.Identifier.WorkflowID
	w.Version = metadata.Identifier.WorkflowVersion

	// Load the event.
	byt, err = m.r.Get(ctx, m.kf.Event(ctx, id)).Bytes()
	if err != nil {
		return nil, fmt.Errorf("failed to get event; %w", err)
	}
	event := map[string]any{}
	if err := json.Unmarshal(byt, &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event; %w", err)
	}

	// Load the actions.  This is a map of step IDs to JSON-encoded results.
	rmap, err := m.r.HGetAll(ctx, m.kf.Actions(ctx, id)).Result()
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
	rmap, err = m.r.HGetAll(ctx, m.kf.Errors(ctx, id)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to load errors; %w", err)
	}
	errors := map[string]error{}
	for stepID, str := range rmap {
		errors[stepID] = fmt.Errorf(str)
	}

	meta := metadata.Metadata()

	return inmemory.NewStateInstance(*w, id, meta, event, actions, errors), nil
}

func (m mgr) SaveResponse(ctx context.Context, i state.Identifier, r state.DriverResponse, attempt int) (state.State, error) {
	var (
		data            any
		err             error
		typ             enums.HistoryType
		funcFailHistory state.History
	)

	now := time.Now()

	if r.Err == nil {
		typ = enums.HistoryTypeStepCompleted
		if data, err = json.Marshal(r.Output); err != nil {
			return nil, fmt.Errorf("error marshalling step output: %w", err)
		}
	} else {
		typ = enums.HistoryTypeStepErrored
		data = output(map[string]any{
			"error":  r.Err.Error(),
			"output": r.Output,
		})
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

	stepHistory := state.History{
		ID:         state.HistoryID(),
		Type:       typ,
		Identifier: i,
		CreatedAt:  now,
		Data: state.HistoryStep{
			ID:      r.Step.ID,
			Name:    r.Step.Name,
			Attempt: attempt,
			Data:    data,
		},
	}

	err = scripts["saveResponse"].Eval(
		ctx,
		m.r,
		[]string{
			m.kf.Actions(ctx, i),
			m.kf.Errors(ctx, i),
			m.kf.RunMetadata(ctx, i.RunID),
			m.kf.History(ctx, i.RunID),
		},
		data,
		r.Step.ID,
		r.Err != nil,
		r.Final(),
		stepHistory,
		funcFailHistory,
		now.UnixMilli(),
	).Err()
	if err != nil {
		return nil, fmt.Errorf("error finalizing: %w", err)
	}

	if r.Err != nil && r.Final() {
		// Trigger error callbacks
		go m.runCallbacks(ctx, i, enums.RunStatusFailed)
	}

	return m.Load(ctx, i.RunID)
}

func (m mgr) Started(ctx context.Context, id state.Identifier, stepID string, attempt int) error {
	now := time.Now()

	return m.r.ZAdd(ctx, m.kf.History(ctx, id.RunID), &redis.Z{
		Score: float64(now.UnixMilli()),
		Member: state.History{
			ID:         state.HistoryID(),
			Type:       enums.HistoryTypeStepStarted,
			Identifier: id,
			CreatedAt:  now,
			Data: state.HistoryStep{
				ID:      stepID,
				Attempt: attempt,
			},
		},
	}).Err()
}

func (m mgr) Scheduled(ctx context.Context, i state.Identifier, stepID string, attempt int, at *time.Time) error {
	now := time.Now()

	err := scripts["scheduled"].Eval(
		ctx,
		m.r,
		[]string{m.kf.RunMetadata(ctx, i.RunID), m.kf.History(ctx, i.RunID)},
		state.History{
			ID:         state.HistoryID(),
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
	).Err()
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

	status, err := scripts["finalize"].Eval(
		ctx,
		m.r,
		[]string{m.kf.RunMetadata(ctx, i.RunID), m.kf.History(ctx, i.RunID)},
		state.History{
			ID:         state.HistoryID(),
			Type:       history,
			Identifier: i,
			CreatedAt:  now,
		},
		now.UnixMilli(),
		int(finalStatus),
	).Int64()
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
	}

	status, err := scripts["savePause"].Eval(
		ctx,
		m.r,
		keys,

		string(packed),
		p.ID.String(),
		evt,
		int(time.Until(p.Expires.Time()).Seconds()),
		// Add at least 10 minutes to this pause, allowing us to process the
		// pause by ID for 10 minutes past expiry.
		int(time.Until(p.Expires.Time().Add(10*time.Minute)).Seconds()),
	).Int64()
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
	status, err := scripts["leasePause"].Eval(
		ctx,
		m.r,
		[]string{m.kf.PauseID(ctx, id), m.kf.PauseLease(ctx, id)},
		time.Now().UnixMilli(),
		state.PauseLeaseDuration.Seconds(),
	).Int64()
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

	eventKey := ""
	if p.Event != nil {
		eventKey = m.kf.PauseEvent(ctx, p.WorkspaceID, *p.Event)
	}
	keys := []string{
		m.kf.PauseID(ctx, id),
		m.kf.PauseStep(ctx, p.Identifier, p.Incoming),
		eventKey,
		m.kf.Actions(ctx, p.Identifier),
	}

	status, err := scripts["consumePause"].Eval(
		ctx,
		m.r,
		keys,

		id.String(),
		p.DataKey,
		string(marshalledData),
	).Int64()
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
	cmd := m.r.HScan(ctx, m.kf.PauseEvent(ctx, workspaceID, event), 0, "", 0)
	if err := cmd.Err(); err != nil {
		return nil, err
	}

	i := cmd.Iterator()
	if i == nil {
		return nil, fmt.Errorf("unable to create event iterator")
	}

	return &iter{ri: i}, nil
}

func (m mgr) PauseByID(ctx context.Context, id uuid.UUID) (*state.Pause, error) {
	str, err := m.r.Get(ctx, m.kf.PauseID(ctx, id)).Result()
	if err == redis.Nil {
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
	str, err := m.r.Get(ctx, m.kf.PauseStep(ctx, i, actionID)).Result()
	if err == redis.Nil {
		return nil, state.ErrPauseNotFound
	}
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(str)
	if err != nil {
		return nil, err
	}

	str, err = m.r.Get(ctx, m.kf.PauseID(ctx, id)).Result()
	if err == redis.Nil {
		return nil, state.ErrPauseNotFound
	}
	if err != nil {
		return nil, err
	}

	pause := &state.Pause{}
	err = json.Unmarshal([]byte(str), pause)
	return pause, err
}

func (m mgr) History(ctx context.Context, runID ulid.ULID) ([]state.History, error) {
	items, err := m.r.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key:     m.kf.History(ctx, runID),
		Start:   "-inf",
		Stop:    "+inf",
		ByScore: true,
	}).Result()
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
	items, err := m.r.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key:     key,
		Start:   "-inf",
		Stop:    "+inf",
		ByScore: true,
	}).Result()
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
			if err := m.r.ZRem(ctx, key, i).Err(); err != nil {
				return err
			}
			return nil
		}
	}

	return nil
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

type iter struct {
	ri *redis.ScanIterator
}

func (i *iter) Next(ctx context.Context) bool {
	// Skip over the key;  we're using HScan which returns key then value.
	_ = i.ri.Next(ctx)
	return i.ri.Next(ctx)
}

func (i *iter) Val(ctx context.Context) *state.Pause {
	val := i.ri.Val()
	if val == "" {
		return nil
	}

	pause := &state.Pause{}
	if err := json.Unmarshal([]byte(val), pause); err != nil {
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
		"ctx":           r.Context,
	}
}

func (r runMetadata) Metadata() state.Metadata {
	m := state.Metadata{
		Identifier: r.Identifier,
		Pending:    r.Pending,
		Debugger:   r.Debugger,
		Status:     r.Status,
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
