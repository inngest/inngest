package conformance

import (
	"fmt"
	"path/filepath"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	Transport Transport     `koanf:"transport" json:"transport"`
	Suites    []string      `koanf:"suites" json:"suites"`
	Cases     []string      `koanf:"cases" json:"cases"`
	Features  []string      `koanf:"features" json:"features"`
	Report    ReportConfig  `koanf:"report" json:"report"`
	Fixtures  FixtureConfig `koanf:"fixtures" json:"fixtures"`
	Golden    GoldenConfig  `koanf:"golden" json:"golden"`
}

type ReportConfig struct {
	Format ReportFormat `koanf:"format" json:"format"`
	Output string       `koanf:"output" json:"output"`
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
	return nil
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
