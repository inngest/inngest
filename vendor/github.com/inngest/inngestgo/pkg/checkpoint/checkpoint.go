package checkpoint

import (
	"time"
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
// You should ALWAYS start with the zero config and only tweak these parameters
// to further improve latency if necessary.
type Config struct {
	// BatchSteps represents the maximum number of steps to execute before checkpointing. When this
	// limit is hit, checkpointing will occur and the SDK will block until checkpointing completes.
	//
	// This must be higher than zero to enable batching of steps.  When enabled with BatchInterval,
	// whichever limit is hit first will checkpoint steps.
	BatchSteps int

	// BatchInterval represents the maximum time that we wait after a step runs before checkpointing.
	// When this limit is hit, checkpointing will occur and the SDK will block until checkpointing completes.
	//
	// This must be higher than zero to enable batching based off of durations.  When enabled with BatchSteps,
	// whichever lmiit is hit first will checkpoint steps.
	BatchInterval time.Duration

	// MaxRuntime indicates the maximum duration that a function can execute for before Inngest requires
	// a fresh re-entry.  This is useful for serverless functions, ensuring that a fresh request is made
	// after a maximum amount of time.
	MaxRuntime time.Duration

	// MaxSteps represents the maximum number of steps the function can execute before
	// stopping execution and waiting for another re-entry.  This is useful for serverless functions,
	// ensuring that we create a new request (and serverless function lifecycle) after a maximum number
	// of requests.
	//
	// If zero, there are no limits on the number of steps that can be executed, and the SDK will execute
	// step.run calls until an async step is reached.
	// MaxSteps int
}

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
		BatchSteps: AllSteps,
	}

	// ConfigBlended checkpoints after 3 steps or 3 seconds pass, giving a blend between
	// performance and safety.
	ConfigBlended = &Config{
		BatchSteps:    3,
		BatchInterval: 3 * time.Second,
	}
)
