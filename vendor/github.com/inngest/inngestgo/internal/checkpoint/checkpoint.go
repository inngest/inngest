package checkpoint

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngestgo/internal/opcode"
)

const (
	// AllSteps attempts to checkpoint as many steps as possible.
	AllSteps = 1_000
)

var (
	// CheckpointSafe is the safest configuration, which checkpoints after each step
	// in a blocking manner.
	//
	// By default, you should use this configuration.  You should also ALWAYS use this
	// configuration first, and only tune these parameters to further improve latency.
	ConfigSafe = &Config{}

	// ConfigPerformant is the least safe configuration, and runs as many steps as possible,
	// until a checkpoint is forced via an async step (eg. step.sleep, step.waitForEvent),
	// or the run ends.
	//
	// You should ONLY use this configuration if you care about performance over everything,
	// and are comfortable with steps potentially re-running.  Look at and use ConfigBlended,
	// or your own custom config, if you care about both performance and safety.
	//
	// It is NOT recommended to use this in serverless environments.
	ConfigPerformant = &Config{
		MaxSteps: AllSteps,
	}

	// ConfigBlended checkpoints after 3 steps or 3 seconds pass, giving a blend between performance
	// and safety.
	ConfigBlended = &Config{
		MaxSteps:    3,
		MaxInterval: 3 * time.Second,
	}
)

// Config specifies the configuration for checkpointing.
//
// The zero config is the safest checkpoitning configuration (and is the same as ConfigSage).
// This checkpoints after every step.Run, ensuring that step data is saved as quickly as possible
// so that data is not lost.
//
// Tweaking config parameters allows you to "batch" many steps into a single checkpoint.
// If your server dies before the checkpoint completes, step data will be lost and steps
// will rerun.
//
// You should ALWAYS start with the zero config (ConfigSafe) and only tweak these parameters
// to further improve latency if necessary.
type Config struct {
	// MaxSteps represents the maximum number of steps to execute before checkpointing.
	//
	// This must be higher than zero to enable batching of steps.
	MaxSteps int

	// MaxInterval represents the maximum time that we wait after a step runs before checkpointing.
	MaxInterval time.Duration

	// StopAfterSteps represents the maximum number of steps the function can execute before
	// stopping execution and waiting for another re-entry.  This is useful for serverless functions,
	// ensuring that we create a new request (and serverless function lifecycle) after a maximum number
	// of requests.
	//
	// If zero, there are no limits on the number of steps that can be executed, and the SDK will execute
	// step.run calls until an async step is reached.
	// StopAfterSteps int
}

type Opts struct {
	// RunID is the run ID being checkpointed.
	RunID string
	// FnID is the ID of the function being checkpointed.
	FnID uuid.UUID
	// QueueItemRef represents the queue item ref that's currently leased while
	// executing the SDK.
	QueueItemRef string
	// SigningKey is the signing key used to checkpoint.
	SigningKey string
	// Config is the config for the checkpointer.
	Config Config
}

func New(o Opts) Checkpointer {
	return &checkpointer{
		opts:       o,
		buffer:     []opcode.Step{},
		lock:       sync.Mutex{},
		totalSteps: atomic.Int32{},
		t:          atomic.Int64{},
	}
}

type Checkpointer interface {
	// WithStep adds a new step to be checkpointed.  This may be a blocking operation,
	// depending on how the checkoint is configured.
	//
	// The callback will be called once checkpointing completes, and will be passed the
	// checkpointed steps or the error when checkpointing.
	//
	// Because checkpotining is idempotent, it is safe to assume that checkpoints are all-or-nothing:
	// if there's an error, the caller can assume that nothing checkpointed and all steps
	// need to be saved.
	WithStep(ctx context.Context, step opcode.Step, cb Callback)
}

// Callback represents a callback which is executed whenever a checkpoint commits.
type Callback func(committed []opcode.Step, err error)

type checkpointer struct {
	opts Opts

	// buffer stores the remaining items to checkpoint as a buffer.
	buffer []opcode.Step

	// lock is held when checkpointing, ensuring we only make one call
	// to checkpiint (or add a step) at a time.
	lock sync.Mutex

	// totalSteps records the total steps checkpointed.
	totalSteps atomic.Int32

	// t returns the time  since the epoch (in milliseconds) since the first step
	// was added.  This is used to checkpoint with max intervals.
	t atomic.Int64
}

func (c *checkpointer) WithStep(ctx context.Context, step opcode.Step, cb Callback) {
	c.lock.Lock()
	c.buffer = append(c.buffer, step)
	c.lock.Unlock()

	if len(c.buffer) >= c.opts.Config.MaxSteps {
		// In this case, we've exceeded the total number of steps we can batch.
		c.checkpoint(ctx, cb)
		return
	}

	if c.opts.Config.MaxInterval > 0 && c.t.Load() == 0 {
		// Store the current time in milliseconds atomically.  Note that if this is
		// called simultaneously from two threads after c.t.Load() atomically returns
		// zero, we can assume that this is happening within the same ~millisecond or so,
		// and we don't want to pay the penalty of locks for this.
		c.t.Store(time.Now().UnixMilli())

		// Start a goroutine to checkpoint in the background.
		go func() {
			<-time.After(c.opts.Config.MaxInterval)
			c.checkpoint(ctx, cb)
		}()
	}
}

func (c *checkpointer) checkpoint(ctx context.Context, cb Callback) {
	// This ensures that the buffer is locked and steps cannot be added,
	// and also ensures that we only have one checkpoint running at a time.
	c.lock.Lock()
	defer c.lock.Unlock()

	if len(c.buffer) == 0 {
		return
	}

	err := checkpoint(ctx, c.opts.SigningKey, AsyncRequest{
		RunID:        c.opts.RunID,
		FnID:         c.opts.FnID,
		QueueItemRef: c.opts.QueueItemRef,
		Steps:        c.buffer,
	})
	if err != nil {
		// Call the callback with an error.
		cb(nil, err)
		return
	}

	// Call the callback, ensuring that the manager knows which steps we've
	// checkpointed.
	cb(c.buffer, nil)

	// Reset bookkeeping - time and buffer, after a successful checkpoint.
	// At this point the lock is held so it's safe to do this after cb.
	c.t.Store(0)
	c.buffer = []opcode.Step{}
}

// AsyncRequest represents an async checkpoint of one or more step.run
// opcodes.
type AsyncRequest struct {
	// RunID is the run ID being checkpointed.
	RunID string `json:"run_id"`
	// FnID is the ID of the function being checkpointed.
	FnID uuid.UUID `json:"fn_id"`
	// QueueItemRef represents the queue item ID that's currently leased while
	// executing the SDK.
	QueueItemRef string `json:"qi_id"`
	// Steps represents the steps being checkpointed.
	Steps []opcode.Step `json:"steps"`
}
