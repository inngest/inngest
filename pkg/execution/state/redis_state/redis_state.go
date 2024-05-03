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

	"github.com/google/uuid"
	"github.com/inngest/expr"
	"github.com/inngest/inngest/pkg/config/registration"
	"github.com/inngest/inngest/pkg/consts"
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
}

func (m mgr) New(ctx context.Context, input state.Input) (state.State, error) {
	f, err := m.LoadFunction(ctx, input.Identifier)
	if err != nil {
		return nil, fmt.Errorf("error loading function in state store: %w", err)
	}

	// We marshal this ahead of creating a redis transaction as it's necessary
	// every time and reduces the duration that the lock is held.
	events, err := json.Marshal(input.EventBatchData)
	if err != nil {
		return nil, err
	}

	metadata := runMetadata{
		Identifier:     input.Identifier,
		Debugger:       input.Debugger,
		Version:        currentVersion,
		RequestVersion: consts.RequestVersionUnknown, // Always use -1 to indicate unset hash version until first request.
		Context:        input.Context,
		Status:         enums.RunStatusScheduled,
		SpanID:         input.SpanID,
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

	args, err := StrSlice([]any{
		events,
		metadataByt,
		stepsByt,
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

func (m mgr) UpdateMetadata(ctx context.Context, runID ulid.ULID, md state.MetadataUpdate) error {
	input := []string{
		"0",
		strconv.Itoa(consts.RequestVersionUnknown),
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

	status, err := scripts["updateMetadata"].Exec(
		ctx,
		m.r,
		[]string{
			m.kf.RunMetadata(ctx, runID),
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

func (m mgr) IsComplete(ctx context.Context, runID ulid.ULID) (bool, error) {
	cmd := m.r.B().Hget().Key(m.kf.RunMetadata(ctx, runID)).Field("status").Build()
	val, err := m.r.Do(ctx, cmd).AsBytes()
	if err != nil {
		return false, err
	}
	return !bytes.Equal(val, []byte("0")), nil
}

func (m mgr) Exists(ctx context.Context, runID ulid.ULID) (bool, error) {
	cmd := m.r.B().Exists().Key(m.kf.RunMetadata(ctx, runID)).Build()
	return m.r.Do(ctx, cmd).AsBool()
}

func (m mgr) metadata(ctx context.Context, runID ulid.ULID) (*runMetadata, error) {
	cmd := m.r.B().Hgetall().Key(m.kf.RunMetadata(ctx, runID)).Build()
	val, err := m.r.Do(ctx, cmd).AsStrMap()
	if err != nil {
		return nil, err
	}
	return newRunMetadata(val)
}

func (m mgr) Cancel(ctx context.Context, id state.Identifier) error {
	status, err := scripts["cancel"].Exec(
		ctx,
		m.r,
		[]string{m.kf.RunMetadata(ctx, id.RunID)},
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

func (m mgr) SetStatus(ctx context.Context, id state.Identifier, status enums.RunStatus) error {
	args, err := StrSlice([]any{
		int(status),
	})
	if err != nil {
		return err
	}

	_, err = scripts["setStatus"].Exec(
		ctx,
		m.r,
		[]string{m.kf.RunMetadata(ctx, id.RunID)},
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error cancelling: %w", err)
	}
	return nil
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

func (m mgr) SaveResponse(ctx context.Context, i state.Identifier, stepID, marshalledOuptut string) error {

	keys := []string{
		m.kf.Actions(ctx, i),
		m.kf.RunMetadata(ctx, i.RunID),
		m.kf.Stack(ctx, i.RunID),
	}
	args := []string{stepID, marshalledOuptut}

	index, err := scripts["saveResponse"].Exec(
		ctx,
		m.r,
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

func (m mgr) SavePause(ctx context.Context, p state.Pause) error {
	packed, err := json.Marshal(p)
	if err != nil {
		return err
	}

	// `evt` is used to search for pauses based on event names. We only want to
	// do this if this pause is not part of an invoke. If it is, we don't want
	// to index it by event name as the pause will be processed by correlation
	// ID.
	evt := ""
	if p.Event != nil && (p.InvokeCorrelationID == nil || *p.InvokeCorrelationID == "") {
		evt = *p.Event
	}

	keys := []string{
		m.kf.PauseID(ctx, p.ID),
		m.kf.PauseStep(ctx, p.Identifier, p.Incoming),
		m.kf.PauseEvent(ctx, p.WorkspaceID, evt),
		m.kf.Invoke(ctx, p.WorkspaceID),
		m.kf.PauseIndex(ctx, "add", p.WorkspaceID, evt),
		m.kf.PauseIndex(ctx, "exp", p.WorkspaceID, evt),
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

	corrId := ""
	if p.InvokeCorrelationID != nil {
		corrId = *p.InvokeCorrelationID
	}

	args, err := StrSlice([]any{
		string(packed),
		p.ID.String(),
		evt,
		corrId,
		ttl,
		// Add at least 10 minutes to this pause, allowing us to process the
		// pause by ID for 10 minutes past expiry.
		int(time.Until(p.Expires.Time().Add(10 * time.Minute)).Seconds()),
		time.Now().Unix(),
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

// Delete deletes state from the state store.  Previously, we would handle this in a
// lifecycle.  Now, state stores must account for deletion directly.  Note that if the
// state store is queue-aware, it must delete queue items for the run also.  This may
// not always be the case.
func (m mgr) Delete(ctx context.Context, i state.Identifier) error {
	// Ensure this context isn't cancelled;  this is called in a goroutine.
	callCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Ensure function idempotency exists for the defined period.
	key := m.kf.Idempotency(ctx, i)

	cmd := m.r.B().Expire().Key(key).Seconds(int64(consts.FunctionIdempotencyPeriod.Seconds())).Build()
	if err := m.r.Do(callCtx, cmd).Error(); err != nil {
		return err
	}

	// Clear all other data for a job.
	keys := []string{
		m.kf.Actions(ctx, i),
		m.kf.RunMetadata(ctx, i.RunID),
		m.kf.Events(ctx, i),
		m.kf.Stack(ctx, i.RunID),

		// XXX: remove these in a state store refactor.
		m.kf.Event(ctx, i),
		m.kf.History(ctx, i.RunID),
		m.kf.Errors(ctx, i),
	}
	for _, k := range keys {
		cmd := m.r.B().Del().Key(k).Build()
		if err := m.r.Do(callCtx, cmd).Error(); err != nil {
			return err
		}
	}
	return nil
}

func (m mgr) DeletePause(ctx context.Context, p state.Pause) error {
	// Add a default event here, which is null and overwritten by everything.  This is necessary
	// to keep the same cluster key.
	eventKey := m.kf.PauseEvent(ctx, p.WorkspaceID, "-")
	if p.Event != nil {
		eventKey = m.kf.PauseEvent(ctx, p.WorkspaceID, *p.Event)
	}
	keys := []string{
		m.kf.PauseID(ctx, p.ID),
		m.kf.PauseStep(ctx, p.Identifier, p.Incoming),
		eventKey,
		m.kf.Invoke(ctx, p.WorkspaceID),
	}
	corrId := ""
	if p.InvokeCorrelationID != nil && *p.InvokeCorrelationID != "" {
		corrId = *p.InvokeCorrelationID
	}
	status, err := scripts["deletePause"].Exec(
		ctx,
		m.pauseR,
		keys,
		[]string{
			p.ID.String(),
			corrId,
		},
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error consuming pause: %w", err)
	}
	switch status {
	case 0:
		return nil
	default:
		return fmt.Errorf("unknown response deleting pause: %d", status)
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
		m.kf.Invoke(ctx, p.WorkspaceID),
		m.kf.Actions(ctx, p.Identifier),
		m.kf.Stack(ctx, p.Identifier.RunID),
	}

	corrId := ""
	if p.InvokeCorrelationID != nil && *p.InvokeCorrelationID != "" {
		corrId = *p.InvokeCorrelationID
	}
	args, err := StrSlice([]any{
		id.String(),
		corrId,
		p.DataKey,
		string(marshalledData),
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

func (m mgr) PauseByInvokeCorrelationID(ctx context.Context, wsID uuid.UUID, correlationID string) (*state.Pause, error) {
	key := m.kf.Invoke(ctx, wsID)
	cmd := m.pauseR.B().Hget().Key(key).Field(correlationID).Build()
	pauseIDstr, err := m.pauseR.Do(ctx, cmd).ToString()
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

func (m mgr) PausesByID(ctx context.Context, ids ...uuid.UUID) ([]*state.Pause, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	keys := make([]string, len(ids))
	for n, id := range ids {
		keys[n] = m.kf.PauseID(ctx, id)
	}

	cmd := m.pauseR.B().Mget().Key(keys...).Build()
	strings, err := m.pauseR.Do(ctx, cmd).AsStrSlice()
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

// PausesByEvent returns all pauses for a given event within a workspace.
func (m mgr) PausesByEvent(ctx context.Context, workspaceID uuid.UUID, event string) (state.PauseIterator, error) {
	key := m.kf.PauseEvent(ctx, workspaceID, event)
	// If there are > 1000 keys in the hmap, use scanning

	cntCmd := m.pauseR.B().Hlen().Key(key).Build()
	cnt, err := m.pauseR.Do(ctx, cntCmd).AsInt64()

	if err != nil || cnt > 1000 {
		key := m.kf.PauseEvent(ctx, workspaceID, event)
		iter := &scanIter{
			count: cnt,
			r:     m.pauseR,
		}
		err := iter.init(ctx, key, 1000)
		return iter, err
	}

	// If there are less than a thousand items, query the keys
	// for iteration.
	iter := &bufIter{r: m.pauseR}
	err = iter.init(ctx, key)
	return iter, err
}

func (m mgr) PausesByEventSince(ctx context.Context, workspaceID uuid.UUID, event string, since time.Time) (state.PauseIterator, error) {
	if since.IsZero() {
		return m.PausesByEvent(ctx, workspaceID, event)
	}

	// Load all items in the set.
	cmd := m.r.B().
		Zrangebyscore().
		Key(m.kf.PauseIndex(ctx, "add", workspaceID, event)).
		Min(strconv.Itoa(int(since.Unix()))).
		Max("+inf").
		Build()
	ids, err := m.r.Do(ctx, cmd).AsStrSlice()
	if err != nil {
		return nil, err
	}

	iter := &keyIter{
		r:  m.r,
		kf: m.kf,
	}
	err = iter.init(ctx, ids, 100)
	return iter, err
}

func (m mgr) EvaluablesByID(ctx context.Context, ids ...uuid.UUID) ([]expr.Evaluable, error) {
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

func (m mgr) LoadEvaluablesSince(ctx context.Context, workspaceID uuid.UUID, eventName string, since time.Time, do func(context.Context, expr.Evaluable) error) error {

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
			expired = append(expired, pause)
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

	val *state.Pause
	err error

	l sync.Mutex
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

func (i *bufIter) Next(ctx context.Context) bool {
	i.l.Lock()
	defer i.l.Unlock()

	if len(i.items) == 0 {
		i.err = context.Canceled
		return false
	}

	pause := &state.Pause{}
	i.err = json.Unmarshal([]byte(i.items[0]), pause)
	i.val = pause
	// Remove one from the slice.
	i.items = i.items[1:]
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
	vals   rueidis.ScanEntry
	err    error

	l sync.Mutex
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
	kf KeyGenerator
	// chunk is the size of scans to load in one.
	chunk int64
	// keys stores pause IDs to fetch in batches
	keys []string
	// vals stores pauses as strings from MGET
	vals []string
	err  error
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

func (i *keyIter) fetch(ctx context.Context) error {
	if len(i.keys) == 0 {
		// No more present.
		i.err = context.Canceled
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
		load[n] = i.kf.PauseID(ctx, uuid.MustParse(id))
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
	Pending                   int            `json:"pending"`
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
		"pending":  r.Pending,
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
