package redis_state

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/inngest/inngest/inngest"
	"github.com/inngest/inngest/pkg/config/registration"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/inmemory"
)

const (
	// defaultExpiry is used as a placeholder while we configure open-source
	// workflow timeouts.  Right now, data stored in redis never has a TTL.
	defaultExpiry = 0
)

func init() {
	registration.RegisterState(func() any { return &Config{} })
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
		WithKeyGenerator(defaultKeyFunc{prefix: c.KeyPrefix}),
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
		kf: defaultKeyFunc{},
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

// WithKeyGenerator specifies the function to use when creating keys for
// each stored data type.
func WithKeyGenerator(kf KeyGenerator) Opt {
	return func(m *mgr) {
		m.kf = kf
	}
}

// KeyFunc returns a unique string based off of given data, which is used
// as the key for data stored in redis for workflows, events, actions, and
// errors.
type KeyGenerator interface {
	// Workflow returns the key for the current workflow ID and version.
	Workflow(ctx context.Context, workflowID uuid.UUID, version int) string

	// Idempotency stores the idempotency key for atomic lookup.
	Idempotency(context.Context, state.Identifier) string

	// RunMetadata stores state regarding the current run identifier, such
	// as the workflow version, the time the run started, etc.
	RunMetadata(context.Context, state.Identifier) string

	// Event returns the key used to store the specific event for the
	// given workflow run.
	Event(context.Context, state.Identifier) string

	// Actions returns the key used to store the action response map used
	// for given workflow run - ie. the results for individual steps.
	Actions(context.Context, state.Identifier) string

	// Errors returns the key used to store the error hash map used
	// for given workflow run.
	Errors(context.Context, state.Identifier) string

	// PauseLease stores the key which references a pause's lease.
	//
	// This is stored independently as we may store more than one copy of a pause
	// for easy iteration.
	PauseLease(context.Context, uuid.UUID) string

	// PauseID returns the key used to store an individual pause from its ID.
	PauseID(context.Context, uuid.UUID) string

	// PauseEvent returns the key used to store data for
	PauseEvent(context.Context, string) string

	// PauseStep returns the key used to store a pause ID by the run ID and step ID.
	PauseStep(context.Context, state.Identifier, string) string
}

type mgr struct {
	kf KeyGenerator
	r  *redis.Client
}

func (m mgr) New(ctx context.Context, workflow inngest.Workflow, id state.Identifier, input map[string]any) (state.State, error) {
	// TODO: We could probably optimize the commands here by storing the event
	// within run metadata.  We want step output (actions) and errors to be
	// their own redis hash for fast inserts (HSET on individual step results).
	// However, the input event is immutable.
	metadata := runMetadata{
		Version:   workflow.Version,
		CreatedAt: time.Now().Truncate(time.Second),
		Pending:   1,
	}

	// We marshal this ahead of creating a redis transaction as it's necessary
	// every time and reduces the duration that the lock is held.
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	ikey := m.kf.Idempotency(ctx, id)

	err = m.r.Watch(ctx, func(tx *redis.Tx) error {
		// Ensure that the workflow exists within the state store.
		//
		// XXX: Could this use SETNX to combine these steps or is it more performant
		//      not to have to marshal the workflow every new run?
		key := m.kf.Workflow(ctx, workflow.UUID, workflow.Version)
		val, err := tx.Exists(ctx, key).Uint64()
		if err != nil {
			return err
		}
		if val == 0 {
			// Set the workflow.
			byt, err := json.Marshal(workflow)
			if err != nil {
				return err
			}
			if err := tx.Set(ctx, key, byt, defaultExpiry).Err(); err != nil {
				return err
			}
		}

		set, err := tx.SetNX(ctx, ikey, "", 0).Result()
		if err != nil {
			return err
		}
		if !set {
			return state.ErrIdentifierExists
		}

		// Save metadata about this particular run.
		if err := tx.HSet(ctx, m.kf.RunMetadata(ctx, id), metadata.Map()).Err(); err != nil {
			return err
		}

		// XXX: If/when we enforce limits on function durations here (eg.
		// 1, 5, 10 years) this should have a similar TTL.
		key = m.kf.Event(ctx, id)
		if err := tx.Set(ctx, key, inputJSON, defaultExpiry).Err(); err != nil {
			return err
		}

		return nil
	}, ikey)
	if err != nil {
		return nil, fmt.Errorf("error storing run state in redis: %w", err)
	}

	// We return a new in-memory state instance with the workflow, ID, and input
	// pre-filled.
	return inmemory.NewStateInstance(
			workflow,
			id,
			state.Metadata{
				StartedAt: metadata.CreatedAt,
			},
			input,
			map[string]map[string]any{},
			map[string]error{},
		),
		nil
}

func (m mgr) IsComplete(ctx context.Context, id state.Identifier) (bool, error) {
	val, err := m.r.HGet(ctx, m.kf.RunMetadata(ctx, id), "pending").Result()
	if err != nil {
		return false, err
	}
	return val == "0", nil
}

func (m mgr) metadata(ctx context.Context, id state.Identifier) (*runMetadata, error) {
	val, err := m.r.HGetAll(ctx, m.kf.RunMetadata(ctx, id)).Result()
	if err != nil {
		return nil, err
	}
	return NewRunMetadata(val)
}

func (m mgr) Load(ctx context.Context, id state.Identifier) (state.State, error) {
	// XXX: Use a pipeliner to improve speed.
	metadata, err := m.metadata(ctx, id)
	if err != nil {
		return nil, err
	}

	// Load the workflow.
	byt, err := m.r.Get(ctx, m.kf.Workflow(ctx, id.WorkflowID, metadata.Version)).Bytes()
	if err != nil {
		return nil, err
	}
	w := &inngest.Workflow{}
	if err := json.Unmarshal(byt, w); err != nil {
		return nil, err
	}

	// TODO: We must ensure that the workflow UUID and Version are marshalled in JSON.
	w.UUID = id.WorkflowID
	w.Version = metadata.Version

	// Load the event.
	byt, err = m.r.Get(ctx, m.kf.Event(ctx, id)).Bytes()
	if err != nil {
		return nil, err
	}
	event := map[string]any{}
	if err := json.Unmarshal(byt, &event); err != nil {
		return nil, err
	}

	// Load the actions.  This is a map of step IDs to JSON-encoded results.
	rmap, err := m.r.HGetAll(ctx, m.kf.Actions(ctx, id)).Result()
	if err != nil {
		return nil, err
	}
	actions := map[string]map[string]any{}
	for stepID, marshalled := range rmap {
		data := map[string]any{}
		err = json.Unmarshal([]byte(marshalled), &data)
		if err != nil {
			return nil, err
		}
		actions[stepID] = data
	}
	if err := json.Unmarshal(byt, &event); err != nil {
		return nil, err
	}

	// Load the errors.  This is a map of step IDs to error strings.
	// The original error type is not preserved.
	rmap, err = m.r.HGetAll(ctx, m.kf.Errors(ctx, id)).Result()
	if err != nil {
		return nil, err
	}
	errors := map[string]error{}
	for stepID, str := range rmap {
		errors[stepID] = fmt.Errorf(str)
	}
	if err := json.Unmarshal(byt, &event); err != nil {
		return nil, err
	}

	meta := state.Metadata{
		StartedAt: metadata.CreatedAt,
		Pending:   metadata.Pending,
	}

	return inmemory.NewStateInstance(*w, id, meta, event, actions, errors), nil
}

func (m mgr) SaveResponse(ctx context.Context, i state.Identifier, r state.DriverResponse, attempt int) (state.State, error) {
	if r.Err == nil {
		// Save the output.
		str, err := json.Marshal(r.Output)
		if err != nil {
			return nil, err
		}
		if err := m.r.HSet(ctx, m.kf.Actions(ctx, i), r.Step.ID, str).Err(); err != nil {
			return nil, err
		}
		return m.Load(ctx, i)
	}

	// Save the error.
	err := m.r.Watch(ctx, func(tx *redis.Tx) error {
		// Save the error.
		if err := m.r.HSet(ctx, m.kf.Errors(ctx, i), r.Step.ID, r.Err.Error()).Err(); err != nil {
			return err
		}
		if r.Final() {
			// Increase finalized.
			return m.r.HIncrBy(ctx, m.kf.RunMetadata(ctx, i), "pending", -1).Err()
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return m.Load(ctx, i)
}

func (m mgr) Finalized(ctx context.Context, i state.Identifier, stepID string) error {
	return m.r.HIncrBy(ctx, m.kf.RunMetadata(ctx, i), "pending", -1).Err()
}

func (m mgr) Scheduled(ctx context.Context, i state.Identifier, stepID string) error {
	return m.r.HIncrBy(ctx, m.kf.RunMetadata(ctx, i), "pending", 1).Err()
}

func (m mgr) SaveActionOutput(ctx context.Context, id state.Identifier, actionID string, data map[string]interface{}) (state.State, error) {
	str, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	err = m.r.Watch(ctx, func(tx *redis.Tx) error {
		// Delete the existing error.
		if err := m.r.HDel(ctx, m.kf.Errors(ctx, id), actionID).Err(); err != nil {
			return err
		}
		if err := m.r.HSet(ctx, m.kf.Actions(ctx, id), actionID, str).Err(); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return m.Load(ctx, id)
}

func (m mgr) SaveActionError(ctx context.Context, id state.Identifier, actionID string, err error) (state.State, error) {
	if err := m.r.HSet(ctx, m.kf.Errors(ctx, id), actionID, err.Error()).Err(); err != nil {
		return nil, err
	}
	return m.Load(ctx, id)
}

func (m mgr) SavePause(ctx context.Context, p state.Pause) error {
	packed, err := json.Marshal(p)
	if err != nil {
		return err
	}

	return m.r.Watch(ctx, func(tx *redis.Tx) error {
		err := tx.SetNX(
			ctx,
			m.kf.PauseID(ctx, p.ID),
			string(packed),
			// Add at least 10 minutes to this pause, allowing us to process the
			// pause by ID for 10 minutes past expiry.
			time.Until(p.Expires)+(10*time.Minute),
		).Err()
		if err != nil {
			return err
		}

		// Set a reference to the stored pause within the run-id step-id key.  This allows us
		// to resume workflows from a given Identifer and Step easily.
		err = tx.Set(
			ctx,
			m.kf.PauseStep(ctx, p.Identifier, p.Outgoing),
			p.ID.String(),
			time.Until(p.Expires),
		).Err()
		if err != nil {
			return err
		}

		// If we have an event, add this to the event's hash, keyed by ID.
		//
		// We store all pauses that are triggered by an event in a key containing
		// the event name, allowing us to easily load all pauses for an event and
		// easily remove keys once consumed.
		//
		// NOTE: Because we return an iterator to this set directly for returning pauses
		// matching an event, we must store the pause within this event again.
		if p.Event != nil {
			if err = tx.HSet(ctx, m.kf.PauseEvent(ctx, *p.Event), map[string]any{
				p.ID.String(): string(packed),
			}).Err(); err != nil {
				return err
			}
		}

		return nil
	})

}

func (m mgr) LeasePause(ctx context.Context, id uuid.UUID) error {
	pauseKey := m.kf.PauseID(ctx, id)
	leaseKey := m.kf.PauseLease(ctx, id)
	return m.r.Watch(ctx, func(tx *redis.Tx) error {
		exists, err := tx.Exists(ctx, pauseKey).Uint64()
		if err != nil {
			return err
		}
		if exists == 0 {
			return state.ErrPauseNotFound
		}

		// Fetch the lease
		str, err := tx.Get(ctx, leaseKey).Result()
		if err != redis.Nil && err != nil {
			return err
		}
		if str != "" {
			// We have a lease.
			leasedUntil, err := time.Parse(time.RFC3339Nano, str)
			if err != nil {
				return err
			}

			if time.Now().Before(leasedUntil) {
				return state.ErrPauseLeased
			}
		}

		// Lease the pause.
		lease := time.Now().Add(state.PauseLeaseDuration).Format(time.RFC3339Nano)
		return tx.Set(ctx, leaseKey, lease, time.Until(time.Now().Add(state.PauseLeaseDuration))).Err()
	}, leaseKey)
}

func (m mgr) ConsumePause(ctx context.Context, id uuid.UUID) error {
	key := m.kf.PauseID(ctx, id)

	return m.r.Watch(ctx, func(tx *redis.Tx) error {
		str, err := tx.Get(ctx, key).Result()
		if err == redis.Nil {
			return state.ErrPauseNotFound
		}
		if err != nil {
			return err
		}

		pause := &state.Pause{}
		if err = json.Unmarshal([]byte(str), pause); err != nil {
			return err
		}
		if err := tx.Del(ctx, key).Err(); err != nil {
			return err
		}
		if pause.Event != nil {
			// Remove this from any event, also.
			return tx.HDel(ctx, m.kf.PauseEvent(ctx, *pause.Event), pause.ID.String()).Err()
		}
		return nil
	}, key)
}

// PausesByEvent returns all pauses for a given event.
func (m mgr) PausesByEvent(ctx context.Context, event string) (state.PauseIterator, error) {
	cmd := m.r.HScan(ctx, m.kf.PauseEvent(ctx, event), 0, "", 0)
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

type iter struct {
	ri *redis.ScanIterator
}

func (i *iter) Next(ctx context.Context) bool {
	return i.ri.Next(ctx)
}

func (i *iter) Val(ctx context.Context) *state.Pause {
	// Skip over the key;  we're using HScan which returns key then value.
	_ = i.ri.Next(ctx)

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

	v, ok := data["version"]
	if !ok {
		return nil, fmt.Errorf("no workflow version stored in run metadata")
	}
	m.Version, err = strconv.Atoi(v)
	if err != nil {
		return nil, fmt.Errorf("invalid workflow version stored in run metadata")
	}

	str, ok := data["createdAt"]
	if !ok {
		return nil, fmt.Errorf("no created at stored in run metadata")
	}

	m.CreatedAt, err = time.Parse(time.RFC3339, str)
	if err != nil {
		return nil, fmt.Errorf("invalid created at stored in run metadata")
	}

	str, ok = data["pending"]
	if !ok {
		return nil, fmt.Errorf("no created at stored in run metadata")
	}
	m.Pending, err = strconv.Atoi(str)
	if err != nil {
		return nil, fmt.Errorf("invalid pending stored in run metadata")
	}

	return m, nil
}

// runMetadata is stored for each invocation of a function.  This is inserted when
// creating a new run, and stores the triggering event as well as workflow-specific
// metadata for the invocation.
type runMetadata struct {
	// Version is required to load the correct workflow Version
	// for the specific run.
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"createdAt"`
	Pending   int       `json:"pending"`
}

func (r runMetadata) Map() map[string]any {
	return map[string]any{
		"version":   r.Version,
		"createdAt": r.CreatedAt.Format(time.RFC3339),
		"pending":   r.Pending,
	}
}

type defaultKeyFunc struct {
	prefix string
}

func (d defaultKeyFunc) Idempotency(ctx context.Context, id state.Identifier) string {
	return fmt.Sprintf("%s:key:%s", d.prefix, id.IdempotencyKey())
}

func (d defaultKeyFunc) RunMetadata(ctx context.Context, id state.Identifier) string {
	return fmt.Sprintf("%s:metadata:%s", d.prefix, id.RunID)
}

func (d defaultKeyFunc) Workflow(ctx context.Context, id uuid.UUID, version int) string {
	return fmt.Sprintf("%s:workflows:%s-%d", d.prefix, id, version)
}

func (d defaultKeyFunc) Event(ctx context.Context, id state.Identifier) string {
	return fmt.Sprintf("%s:events:%s:%s", d.prefix, id.WorkflowID, id.RunID)
}

func (d defaultKeyFunc) Actions(ctx context.Context, id state.Identifier) string {
	return fmt.Sprintf("%s:actions:%s:%s", d.prefix, id.WorkflowID, id.RunID)
}

func (d defaultKeyFunc) Errors(ctx context.Context, id state.Identifier) string {
	return fmt.Sprintf("%s:errors:%s:%s", d.prefix, id.WorkflowID, id.RunID)
}

func (d defaultKeyFunc) PauseID(ctx context.Context, id uuid.UUID) string {
	return fmt.Sprintf("%s:pauses:%s", d.prefix, id.String())
}

func (d defaultKeyFunc) PauseLease(ctx context.Context, id uuid.UUID) string {
	return fmt.Sprintf("%s:pause-lease:%s", d.prefix, id.String())
}

func (d defaultKeyFunc) PauseEvent(ctx context.Context, event string) string {
	return fmt.Sprintf("%s:pause-events:%s", d.prefix, event)
}

func (d defaultKeyFunc) PauseStep(ctx context.Context, id state.Identifier, step string) string {
	return fmt.Sprintf("%s:pause-steps:%s-%s", d.prefix, id.RunID, step)
}
