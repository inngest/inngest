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

	// ConfigBlended checkpoints after 3 steps or 3 seconds pass, giving a blend between
	// performance and safety.
	ConfigBlended = &Config{
		MaxSteps:    3,
		MaxInterval: 3 * time.Second,
	}
)
