package config

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"cuelang.org/go/cue"
	cueerrors "cuelang.org/go/cue/errors"
	"github.com/inngest/inngest/pkg/cuedefs"
	"github.com/inngest/inngest/pkg/logger"
)

var (
	// Locations sets the default locations in which configuration is read.
	//
	// We attept to read these files with a .cue and .json prefix,
	// in that order.
	Locations = []string{
		"./inngest",
		"/etc/inngest",
	}

	Exts = []string{".cue", ".json"}

	suffixRegex = regexp.MustCompile(`/.*\.(json|cue)$`)
)

const (
	header = `package main

import (
	c "inngest.com/defs/config"
)

config: c.#Config & `
)

func loadAll(ctx context.Context, locs ...string) (*Config, error) {
	if len(locs) == 0 {
		locs = Locations
	}

	for _, l := range locs {
		if suffixRegex.MatchString(l) {
			// Attempt to load this file directly.
			return read(ctx, l)
		}

		// Try all available suffixes.
		for _, ext := range Exts {
			c, err := read(ctx, l+ext)
			if err == nil {
				return c, nil
			}
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return nil, err
			}
		}
	}

	if os.Getenv("INNGEST_CONFIG") != "" {
		return Parse([]byte(os.Getenv("INNGEST_CONFIG")))
	}

	logger.From(ctx).Warn().Msg("no config file found in search paths, using default")

	// Return the default config.
	return Parse(nil)
}

func read(ctx context.Context, path string) (*Config, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	// The cue file exists.
	byt, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(byt)
}

// parse parses configuration and returns the parsed config.
func Parse(input []byte) (*Config, error) {
	// We want to interpolate environment variables within our configuration file.
	str := os.ExpandEnv(string(input))

	cuedefs.Lock()
	defer cuedefs.Unlock()

	i, err := prepare([]byte(str))
	if err != nil {
		return nil, err
	}
	return decode(i)
}

// prepare generates a cue instance for the configuration.
func prepare(input []byte) (*cue.Instance, error) {
	// Trim the input, and ensure that the JSON has a #Config prefix.
	input = bytes.TrimSpace(input)
	if len(input) == 0 {
		input = []byte("{}")
	}
	if len(input) > 0 && input[0] == '{' {
		input = append([]byte(header), input...)
	}
	return cuedefs.Prepare(input)
}

// decode attempts to decode the input within a cue instance.
func decode(i *cue.Instance) (*Config, error) {
	// Initialize our definition as the root value of the cue instance.  This is
	// the root, top-level object.
	def := i.Value()

	field, err := i.LookupField("config")
	if err == nil {
		// This is a cue definition which contains a function definition.  Cue
		// definitions always have a root level object, and we define the function
		// using the "function" identifier.
		def = field.Value
	}

	if err := def.Validate(cue.Final(), cue.Concrete(true)); err != nil {
		buf := &bytes.Buffer{}
		cueerrors.Print(buf, err, nil)
		return nil, fmt.Errorf("config is invalid: %s", buf.String())
	}

	c := &Config{}
	if err := def.Decode(c); err != nil {
		return nil, fmt.Errorf("error decoding config: %w", err)
	}

	// Ensure we set log levels and formats here, globally.
	// XXX: This could/should be refactored;  this is messy (but minimal impact rn).
	logger.SetFormat(c.Log.Format)

	return c, nil
}
