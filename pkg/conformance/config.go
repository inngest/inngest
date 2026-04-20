package conformance

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// Config describes the user-facing conformance configuration.
//
// Phase 2 only needs the serve transport to be usable, but the shape is kept
// transport-neutral so connect can reuse the same top-level contract later.
type Config struct {
	Transport Transport     `koanf:"transport" json:"transport"`
	Suites    []string      `koanf:"suites" json:"suites"`
	Cases     []string      `koanf:"cases" json:"cases"`
	Features  []string      `koanf:"features" json:"features"`
	Timeout   string        `koanf:"timeout" json:"timeout"`
	SDK       SDKConfig     `koanf:"sdk" json:"sdk"`
	Dev       DevConfig     `koanf:"dev" json:"dev"`
	Report    ReportConfig  `koanf:"report" json:"report"`
	Fixtures  FixtureConfig `koanf:"fixtures" json:"fixtures"`
	Golden    GoldenConfig  `koanf:"golden" json:"golden"`
}

// SDKConfig captures the transport-facing SDK endpoint configuration.
//
// For serve mode the canonical URL is the SDK serve endpoint, for example
// http://127.0.0.1:3000/api/inngest.
type ReportConfig struct {
	Format ReportFormat `koanf:"format" json:"format"`
	Output string       `koanf:"output" json:"output"`
}

type SDKConfig struct {
	URL            string `koanf:"url" json:"url"`
	IntrospectPath string `koanf:"introspect_path" json:"introspect_path"`
}

// DevConfig captures the server-side endpoints and credentials the runner needs
// in order to register functions and publish events.
type DevConfig struct {
	URL        string `koanf:"url" json:"url"`
	APIURL     string `koanf:"api_url" json:"api_url"`
	EventURL   string `koanf:"event_url" json:"event_url"`
	EventKey   string `koanf:"event_key" json:"event_key"`
	SigningKey string `koanf:"signing_key" json:"signing_key"`
}

type FixtureConfig struct {
	Root string `koanf:"root" json:"root"`
}

type GoldenConfig struct {
	Root string     `koanf:"root" json:"root"`
	Mode GoldenMode `koanf:"mode" json:"mode"`
}

func LoadConfig(path string) (Config, error) {
	if path == "" {
		return Config{}, nil
	}

	parser, err := parserForPath(path)
	if err != nil {
		return Config{}, err
	}

	k := koanf.New(".")
	if err := k.Load(file.Provider(path), parser); err != nil {
		return Config{}, fmt.Errorf("error parsing config file: %w", err)
	}

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return Config{}, fmt.Errorf("error unmarshaling config: %w", err)
	}

	if cfg.Golden.Mode == "" {
		cfg.Golden.Mode = GoldenModeSemantic
	}
	if cfg.Report.Format == "" {
		cfg.Report.Format = ReportFormatPretty
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	if c.Transport != "" && !IsValidTransport(c.Transport) {
		return fmt.Errorf("unknown transport %q", c.Transport)
	}
	if c.Report.Format != "" && !IsValidReportFormat(c.Report.Format) {
		return fmt.Errorf("unknown report format %q", c.Report.Format)
	}
	if c.Golden.Mode != "" && c.Golden.Mode != GoldenModeSemantic {
		return fmt.Errorf("unknown golden mode %q", c.Golden.Mode)
	}
	if c.Timeout != "" {
		if _, err := time.ParseDuration(c.Timeout); err != nil {
			return fmt.Errorf("invalid timeout %q: %w", c.Timeout, err)
		}
	}
	return nil
}

// TimeoutOrDefault resolves the configured timeout to a concrete duration.
//
// The default is intentionally conservative so the first runnable showcase
// cases do not inherit a zero timeout when users omit the field.
func (c Config) TimeoutOrDefault(fallback time.Duration) time.Duration {
	if c.Timeout == "" {
		return fallback
	}

	d, err := time.ParseDuration(c.Timeout)
	if err != nil {
		return fallback
	}

	return d
}

func parserForPath(path string) (koanf.Parser, error) {
	switch ext := filepath.Ext(path); ext {
	case ".yaml", ".yml":
		return yaml.Parser(), nil
	case ".json":
		return json.Parser(), nil
	default:
		return nil, fmt.Errorf("unsupported config format %q", ext)
	}
}
