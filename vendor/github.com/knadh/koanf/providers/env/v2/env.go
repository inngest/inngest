// Package env implements a koanf.Provider that reads environment
// variables as conf maps.
package env

import (
	"errors"
	"os"
	"strings"

	"github.com/knadh/koanf/maps"
)

// Env implements an environment variables provider.
type Env struct {
	prefix    string
	delim     string
	transform func(key string, value string) (string, any)
	environ   func() []string
}

// Opt represents optional configuration passed to the provider.
type Opt struct {
	// If specified (case-sensitive), only env vars beginning with
	// the prefix are captured. eg: "APP_"
	Prefix string

	// TransformFunc is an optional callback that takes an environment
	// variable's string name and value, runs arbitrary transformations
	// on them and returns a transformed string key and value of any type.
	// Common usecase are stripping prefixes from keys, lowercasing variable names,
	// replacing _ with . etc. Eg: APP_DB_HOST -> db.host
	// If the returned variable name is an empty string (""), it is ignored altogether.
	TransformFunc func(k, v string) (string, any)

	// EnvironFunc is the optional function that provides the environment
	// variables to the provider. If it's not set, os.Environ is used.
	// This can be used to inject environment variables in tests and mocks.
	EnvironFunc func() []string
}

// Provider returns an environment variables provider that returns
// a nested map[string]interface{} of environment variable where the
// nesting hierarchy of keys is defined by delim. For instance, the
// delim "." will convert the key `parent.child.key: 1`
// to `{parent: {child: {key: 1}}}`.
//
// It takes an optional Opt argument containing a function to override
// the default source for environment variables, which can be useful
// for mocking and parallel unit tests.
func Provider(delim string, o Opt) *Env {
	e := &Env{
		delim:     delim,
		prefix:    o.Prefix,
		environ:   o.EnvironFunc,
		transform: o.TransformFunc,
	}

	// No environ function provided, use the default os.Environ.
	if e.environ == nil {
		e.environ = os.Environ
	}

	return e
}

// ReadBytes is not supported by the env provider.
func (e *Env) ReadBytes() ([]byte, error) {
	return nil, errors.New("env provider does not support this method")
}

// Read reads all available environment variables into a key:value map
// and returns it.
func (e *Env) Read() (map[string]any, error) {
	// Collect the environment variable keys.
	var keys []string
	for _, k := range e.environ() {
		if e.prefix != "" {
			if strings.HasPrefix(k, e.prefix) {
				keys = append(keys, k)
			}
		} else {
			keys = append(keys, k)
		}
	}

	mp := make(map[string]any)
	for _, k := range keys {
		parts := strings.SplitN(k, "=", 2)

		// If there's a transformation callback,
		// run it through every key/value.
		if e.transform != nil {
			key, value := e.transform(parts[0], parts[1])
			// If the callback blanked the key, omit it.
			if key == "" {
				continue
			}

			mp[key] = value
		} else {
			mp[parts[0]] = parts[1]
		}

	}

	if e.delim != "" {
		return maps.Unflatten(mp, e.delim), nil
	}

	return mp, nil
}
