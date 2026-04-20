// Package config defines the load-test run configuration that is shared
// between the harness, the REST API, and worker subprocesses.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Mode selects how the worker SDK treats the target server's signatures.
type Mode string

const (
	// ModeDev targets `inngest dev`. The worker sets INNGEST_DEV, and the
	// SDK skips signature verification — required because the dev server
	// may send unsigned or differently-signed invoke requests.
	ModeDev Mode = "dev"
	// ModeSelfHosted targets `inngest start`. The worker leaves INNGEST_DEV
	// unset and the SDK performs strict signature verification using the
	// configured SigningKey, which MUST match the server's.
	ModeSelfHosted Mode = "selfhosted"
)

// Target describes the Inngest server under test.
type Target struct {
	URL        string  `json:"url" yaml:"url"`
	Mode       Mode    `json:"mode,omitempty" yaml:"mode,omitempty"`
	EventKey   *string `json:"eventKey,omitempty" yaml:"eventKey,omitempty"`
	SigningKey *string `json:"signingKey,omitempty" yaml:"signingKey,omitempty"`
}

// Shape identifies a function-shape template from the shapes package.
type Shape string

const (
	ShapeNoop         Shape = "noop"
	ShapeSteps3       Shape = "steps-3"
	ShapeSteps10      Shape = "steps-10"
	ShapeSleep1s      Shape = "sleep-1s"
	ShapeFanout5      Shape = "fanout-5"
	ShapeRetryForced  Shape = "retry-forced"
)

// ShapeMix maps a shape to the weight of events that should be sent for it.
// Weights are relative; the firer normalizes them.
type ShapeMix map[Shape]int

// RunConfig is the full configuration for one load-test run.
type RunConfig struct {
	Name           string        `json:"name" yaml:"name"`
	Target         Target        `json:"target" yaml:"target"`
	Apps           int           `json:"apps" yaml:"apps"`
	FunctionsPerApp int          `json:"functionsPerApp" yaml:"functionsPerApp"`
	ShapeMix       ShapeMix      `json:"shapeMix" yaml:"shapeMix"`
	Concurrency    int           `json:"concurrency" yaml:"concurrency"`
	EventRate      int           `json:"eventRate" yaml:"eventRate"`       // events / sec
	Duration       time.Duration `json:"duration" yaml:"duration"`
	Warmup         time.Duration `json:"warmup" yaml:"warmup"`
	BatchSize      int           `json:"batchSize" yaml:"batchSize"`       // events per POST
}

// Defaults returns a sensible starting configuration.
func Defaults() RunConfig {
	return RunConfig{
		Name:            "default",
		Target:          Target{URL: "http://127.0.0.1:8288", Mode: ModeDev},
		Apps:            1,
		FunctionsPerApp: 1,
		ShapeMix:        ShapeMix{ShapeNoop: 1},
		Concurrency:     10,
		EventRate:       100,
		Duration:        60 * time.Second,
		Warmup:          10 * time.Second,
		BatchSize:       50,
	}
}

// Validate checks that the configuration is usable. It does not reach out to
// the target — it only checks shape. Fills in Mode=ModeDev when unset.
func (c *RunConfig) Validate() error {
	if c.Target.URL == "" {
		return errors.New("target.url is required")
	}
	if c.Target.Mode == "" {
		c.Target.Mode = ModeDev
	}
	switch c.Target.Mode {
	case ModeDev:
		// no key requirements
	case ModeSelfHosted:
		if c.Target.SigningKey == nil || *c.Target.SigningKey == "" {
			return errors.New("target.signingKey is required when target.mode is selfhosted")
		}
	default:
		return fmt.Errorf("target.mode %q is not a known mode", c.Target.Mode)
	}
	if c.Apps < 1 {
		return errors.New("apps must be >= 1")
	}
	if c.FunctionsPerApp < 1 {
		return errors.New("functionsPerApp must be >= 1")
	}
	if len(c.ShapeMix) == 0 {
		return errors.New("shapeMix must have at least one entry")
	}
	total := 0
	for s, w := range c.ShapeMix {
		if w < 0 {
			return fmt.Errorf("shapeMix[%s] weight must be >= 0", s)
		}
		total += w
	}
	if total == 0 {
		return errors.New("shapeMix total weight must be > 0")
	}
	if c.Concurrency < 1 {
		return errors.New("concurrency must be >= 1")
	}
	if c.EventRate < 1 {
		return errors.New("eventRate must be >= 1")
	}
	if c.Duration <= 0 {
		return errors.New("duration must be > 0")
	}
	if c.Warmup < 0 {
		return errors.New("warmup must be >= 0")
	}
	if c.Warmup >= c.Duration {
		return errors.New("warmup must be < duration")
	}
	if c.BatchSize < 1 {
		return errors.New("batchSize must be >= 1")
	}
	return nil
}

// LoadYAML reads a YAML config file from disk.
func LoadYAML(path string) (RunConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return RunConfig{}, err
	}
	var c RunConfig
	if err := yaml.Unmarshal(b, &c); err != nil {
		return RunConfig{}, err
	}
	return c, nil
}

// LoadJSON reads a JSON config blob from a byte slice (used by the REST API).
func LoadJSON(b []byte) (RunConfig, error) {
	var c RunConfig
	if err := json.Unmarshal(b, &c); err != nil {
		return RunConfig{}, err
	}
	return c, nil
}

// WorkerConfig is the JSON payload sent to a worker subprocess on stdin. It
// describes exactly what that worker should register and how to phone home.
type WorkerConfig struct {
	RunID            string   `json:"runId"`
	WorkerID         string   `json:"workerId"`
	AppID            string   `json:"appId"`
	Target           Target   `json:"target"`
	Shapes           []Shape  `json:"shapes"`
	TelemetrySocket  string   `json:"telemetrySocket"`
	HTTPPort         int      `json:"httpPort"`
}
